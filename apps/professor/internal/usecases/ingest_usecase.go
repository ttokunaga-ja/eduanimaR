package usecases

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	pgvector "github.com/pgvector/pgvector-go"
	"golang.org/x/sync/errgroup"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/ports"
)

// IngestUseCase は OCR/Embedding パイプラインのビジネスロジックを担う。
// Kafka コンシューマーがメッセージを受信するたびに ProcessJob を呼び出す。
type IngestUseCase struct {
	files                ports.FileRepository
	jobs                 ports.IngestJobRepository
	chunks               ports.ChunkRepository
	storage              ports.ObjectStorage
	llm                  ports.LLMClient
	embeddingConcurrency int
}

// NewIngestUseCase は IngestUseCase を生成する。
//
// embeddingConcurrency: 1ファイル内でチャンク Embedding を並列生成する数。
// 0 以下の場合はデフォルト値 5 を使用。
func NewIngestUseCase(
	files ports.FileRepository,
	jobs ports.IngestJobRepository,
	chunks ports.ChunkRepository,
	storage ports.ObjectStorage,
	llm ports.LLMClient,
	embeddingConcurrency int,
) *IngestUseCase {
	if embeddingConcurrency <= 0 {
		embeddingConcurrency = 5
	}
	return &IngestUseCase{
		files:                files,
		jobs:                 jobs,
		chunks:               chunks,
		storage:              storage,
		llm:                  llm,
		embeddingConcurrency: embeddingConcurrency,
	}
}

// ProcessJob は Kafka から受信した IngestMessage を処理する。
//
// フロー:
//  1. IngestJob を "processing" に更新
//  2. FileStatus を "processing" に更新
//  3. MinIO からファイルをダウンロード
//  4. LLM.OCRAndChunk でテキスト抽出・チャンク分割
//  5. 各チャンクの Embedding 生成（embeddingConcurrency 件並列、失敗チャンクはスキップ）
//  6. ChunkRepository.BatchCreate でバルク保存
//  7. FileStatus → "completed", IngestJob → "completed"
//
// エラー時: FileStatus → "failed", IngestJob → "failed"（defer で確実に実行）
func (uc *IngestUseCase) ProcessJob(ctx context.Context, msg ports.IngestMessage) error {
	jobID, err := uuid.Parse(msg.JobID)
	if err != nil {
		return fmt.Errorf("invalid job_id %q: %w", msg.JobID, err)
	}
	fileID, err := uuid.Parse(msg.FileID)
	if err != nil {
		return fmt.Errorf("invalid file_id %q: %w", msg.FileID, err)
	}
	subjectID, err := uuid.Parse(msg.SubjectID)
	if err != nil {
		return fmt.Errorf("invalid subject_id %q: %w", msg.SubjectID, err)
	}

	slog.Info("ingest job started",
		"job_id", jobID,
		"file_id", fileID,
		"mime_type", msg.MimeType,
	)

	// 1. IngestJob → "processing"
	if _, err := uc.jobs.UpdateStatus(ctx, jobID, domain.JobStatusProcessing, nil); err != nil {
		return fmt.Errorf("update job processing: %w", err)
	}

	// エラー発生時のロールバック処理（defer で確実に実行）
	var processErr error
	defer func() {
		if processErr != nil {
			errMsg := processErr.Error()
			if _, e := uc.jobs.UpdateStatus(ctx, jobID, domain.JobStatusFailed, &errMsg); e != nil {
				slog.Error("failed to mark job as failed", "job_id", jobID, "error", e)
			}
			if _, e := uc.files.UpdateStatus(ctx, fileID, domain.FileStatusFailed, &errMsg); e != nil {
				slog.Error("failed to mark file as failed", "file_id", fileID, "error", e)
			}
			slog.Error("ingest job failed",
				"job_id", jobID,
				"file_id", fileID,
				"error", processErr,
			)
		}
	}()

	// 2. FileStatus → "processing"
	if _, err := uc.files.UpdateStatus(ctx, fileID, domain.FileStatusProcessing, nil); err != nil {
		processErr = fmt.Errorf("update file processing: %w", err)
		return processErr
	}

	// 3. MinIO からファイルコンテンツをダウンロード
	rc, err := uc.storage.Download(ctx, msg.StoragePath)
	if err != nil {
		processErr = fmt.Errorf("storage download %q: %w", msg.StoragePath, err)
		return processErr
	}
	defer func() {
		if err := rc.Close(); err != nil {
			slog.Warn("failed to close storage reader", "job_id", jobID, "error", err)
		}
	}()

	fileContent, err := io.ReadAll(rc)
	if err != nil {
		processErr = fmt.Errorf("read file content: %w", err)
		return processErr
	}
	slog.Info("file downloaded", "job_id", jobID, "size_bytes", len(fileContent))

	// 4. OCR & チャンク分割
	ocrResult, err := uc.llm.OCRAndChunk(ctx, fileContent, msg.MimeType)
	if err != nil {
		processErr = fmt.Errorf("ocr and chunk: %w", err)
		return processErr
	}
	if len(ocrResult.Chunks) == 0 {
		processErr = fmt.Errorf("ocr produced no chunks for file %s", fileID)
		return processErr
	}
	slog.Info("ocr completed",
		"job_id", jobID,
		"chunk_count", len(ocrResult.Chunks),
	)

	// 5. 各チャンクの Embedding 生成（embeddingConcurrency 件並列）
	//
	// 並列化の仕組み:
	//   - errgroup で全チャンクに goroutine を割り当て
	//   - セマフォ（sem）で同時実行数を embeddingConcurrency に制限
	//   - Embedding 失敗は警告のみ（そのチャンクをスキップ）
	//   - mu（Mutex）で chunks スライスへの並列書き込みを保護
	now := time.Now().UTC()
	var mu sync.Mutex
	chunks := make([]*domain.Chunk, 0, len(ocrResult.Chunks))

	g, gCtx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, uc.embeddingConcurrency)

	for _, c := range ocrResult.Chunks {
		if c.Content == "" {
			continue
		}
		c := c // ループ変数を goroutine にキャプチャ

		g.Go(func() error {
			// セマフォ取得: スロットが埋まっているか ctx キャンセル待ち
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-gCtx.Done():
				return nil // コンテキストキャンセル時はスキップ
			}

			// ctx を使用（gCtx は errgroup 内部のキャンセル伝播用のみ）
			emb, embErr := uc.llm.GenerateDocumentEmbedding(ctx, c.Content)
			if embErr != nil {
				// Embedding 失敗は警告のみ（そのチャンクをスキップ）
				slog.Warn("embedding failed, skipping chunk",
					"job_id", jobID,
					"chunk_index", c.Index,
					"error", embErr,
				)
				return nil
			}

			mu.Lock()
			chunks = append(chunks, &domain.Chunk{
				ID:         uuid.New(),
				FileID:     fileID,
				SubjectID:  subjectID,
				PageNumber: c.PageNumber,
				ChunkIndex: c.Index,
				Content:    c.Content,
				Embedding:  pgvector.NewVector(emb),
				CreatedAt:  now,
			})
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		processErr = fmt.Errorf("parallel embedding: %w", err)
		return processErr
	}

	slog.Info("embeddings generated",
		"job_id", jobID,
		"embedded_chunks", len(chunks),
		"total_chunks", len(ocrResult.Chunks),
		"embedding_concurrency", uc.embeddingConcurrency,
	)

	// 6. DB にバルク保存
	if len(chunks) == 0 {
		processErr = fmt.Errorf("all chunks failed embedding for file %s", fileID)
		return processErr
	}
	if err := uc.chunks.BatchCreate(ctx, chunks); err != nil {
		processErr = fmt.Errorf("batch create chunks: %w", err)
		return processErr
	}
	slog.Info("chunks saved to db",
		"job_id", jobID,
		"count", len(chunks),
	)

	// 7. FileStatus → "completed", IngestJob → "completed"
	if _, err := uc.files.UpdateStatus(ctx, fileID, domain.FileStatusCompleted, nil); err != nil {
		processErr = fmt.Errorf("update file completed: %w", err)
		return processErr
	}
	if _, err := uc.jobs.UpdateStatus(ctx, jobID, domain.JobStatusCompleted, nil); err != nil {
		// completed 更新失敗はログのみ（ファイルは ready 済みのため致命的ではない）
		slog.Warn("failed to update job status to completed",
			"job_id", jobID,
			"error", err,
		)
	}

	slog.Info("ingest job completed",
		"job_id", jobID,
		"file_id", fileID,
		"chunks_stored", len(chunks),
	)
	return nil
}
