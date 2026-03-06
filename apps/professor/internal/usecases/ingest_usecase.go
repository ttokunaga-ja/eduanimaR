package usecases

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
)

// IngestUseCase は OCR/Embedding パイプラインのビジネスロジックを担う。
// Kafka コンシューマーがメッセージを受信するたびに ProcessJob を呼び出す。
type IngestUseCase struct {
	files   ports.FileRepository
	jobs    ports.IngestJobRepository
	chunks  ports.ChunkRepository
	storage ports.ObjectStorage
	llm     ports.LLMClient
}

// NewIngestUseCase は IngestUseCase を生成する。
func NewIngestUseCase(
	files ports.FileRepository,
	jobs ports.IngestJobRepository,
	chunks ports.ChunkRepository,
	storage ports.ObjectStorage,
	llm ports.LLMClient,
) *IngestUseCase {
	return &IngestUseCase{
		files:   files,
		jobs:    jobs,
		chunks:  chunks,
		storage: storage,
		llm:     llm,
	}
}

// ProcessJob は Kafka から受信した IngestMessage を処理する。
//
// フロー:
//  1. IngestJob を "processing" に更新
//  2. FileStatus を "processing" に更新
//  3. MinIO からファイルをダウンロード
//  4. LLM.OCRAndChunk でテキスト抽出・チャンク分割
//  5. 各チャンクの Embedding 生成（失敗チャンクはスキップ）
//  6. ChunkRepository.BatchCreate でバルク保存
//  7. FileStatus → "ready", IngestJob → "completed"
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
	defer rc.Close()

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

	// 5. 各チャンクの Embedding 生成
	now := time.Now().UTC()
	chunks := make([]*domain.Chunk, 0, len(ocrResult.Chunks))

	for _, c := range ocrResult.Chunks {
		if c.Content == "" {
			continue
		}

		emb, embErr := uc.llm.GenerateEmbedding(ctx, c.Content)
		if embErr != nil {
			// Embedding 失敗は警告のみ（そのチャンクをスキップ）
			slog.Warn("embedding failed, skipping chunk",
				"job_id", jobID,
				"chunk_index", c.Index,
				"error", embErr,
			)
			continue
		}

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
	}

	slog.Info("embeddings generated",
		"job_id", jobID,
		"embedded_chunks", len(chunks),
		"total_chunks", len(ocrResult.Chunks),
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

	// 7. FileStatus → "ready", IngestJob → "completed"
	if _, err := uc.files.UpdateStatus(ctx, fileID, domain.FileStatusReady, nil); err != nil {
		processErr = fmt.Errorf("update file ready: %w", err)
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
