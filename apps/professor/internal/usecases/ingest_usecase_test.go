package usecases_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/testhelper"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/usecases"
)

// ─── テストヘルパー ────────────────────────────────────────────────

// newIngestUseCase はテスト用依存を注入した IngestUseCase を返す。
func newIngestUseCase(
	files *testhelper.MockFileRepository,
	jobs *testhelper.MockIngestJobRepository,
	chunks *testhelper.MockChunkRepository,
	storage *testhelper.MockObjectStorage,
	llm *testhelper.MockLLMClient,
) *usecases.IngestUseCase {
	return usecases.NewIngestUseCase(files, jobs, chunks, storage, llm)
}

// validIngestMessage は標準的なテスト用 IngestMessage を返す。
func validIngestMessage() ports.IngestMessage {
	return ports.IngestMessage{
		JobID:       testhelper.FixtureJobID.String(),
		FileID:      testhelper.FixtureFileID.String(),
		SubjectID:   testhelper.FixtureSubjectID.String(),
		UserID:      testhelper.FixtureUserID.String(),
		StoragePath: "subjects/test/test.pdf",
		MimeType:    "application/pdf",
	}
}

// fakePDFContent はテスト用のダミーファイルバイト列。
var fakePDFContent = []byte("%PDF-1.4 fake content for testing")

// ─── ProcessJob 正常系 ────────────────────────────────────────────

func TestIngestUseCase_ProcessJob_Success(t *testing.T) {
	ctx := context.Background()
	msg := validIngestMessage()
	jobID := testhelper.FixtureJobID
	fileID := testhelper.FixtureFileID

	files := &testhelper.MockFileRepository{}
	jobs := &testhelper.MockIngestJobRepository{}
	chunks := &testhelper.MockChunkRepository{}
	storage := &testhelper.MockObjectStorage{}
	llmClient := &testhelper.MockLLMClient{}

	// 1. IngestJob → processing
	jobs.On("UpdateStatus", ctx, jobID, domain.JobStatusProcessing, (*string)(nil)).
		Return(testhelper.NewIngestJob(domain.JobStatusProcessing), nil)

	// 2. FileStatus → processing
	files.On("UpdateStatus", ctx, fileID, domain.FileStatusProcessing, (*string)(nil)).
		Return(testhelper.NewFile(domain.FileStatusProcessing), nil)

	// 3. MinIO ダウンロード
	rc := io.NopCloser(bytes.NewReader(fakePDFContent))
	storage.On("Download", ctx, msg.StoragePath).Return(rc, nil)

	// 4. OCR & チャンク
	pageNum := 1
	ocrResult := &ports.OCRResult{
		Chunks: []ports.ChunkData{
			{Index: 0, Content: "チャンク1のテキスト", PageNumber: &pageNum},
			{Index: 1, Content: "チャンク2のテキスト", PageNumber: &pageNum},
		},
	}
	llmClient.On("OCRAndChunk", ctx, fakePDFContent, msg.MimeType).Return(ocrResult, nil)

	// 5. Embedding 生成（各チャンクに対して）
	emb1 := make([]float32, 768)
	emb2 := make([]float32, 768)
	llmClient.On("GenerateEmbedding", ctx, "チャンク1のテキスト").Return(emb1, nil)
	llmClient.On("GenerateEmbedding", ctx, "チャンク2のテキスト").Return(emb2, nil)

	// 6. DB バルク保存
	chunks.On("BatchCreate", ctx, mock.Anything).Return(nil)

	// 7. FileStatus → ready
	files.On("UpdateStatus", ctx, fileID, domain.FileStatusReady, (*string)(nil)).
		Return(testhelper.NewFile(domain.FileStatusReady), nil)

	// 7. IngestJob → completed
	jobs.On("UpdateStatus", ctx, jobID, domain.JobStatusCompleted, (*string)(nil)).
		Return(testhelper.NewIngestJob(domain.JobStatusCompleted), nil)

	uc := newIngestUseCase(files, jobs, chunks, storage, llmClient)
	err := uc.ProcessJob(ctx, msg)

	require.NoError(t, err)

	jobs.AssertExpectations(t)
	files.AssertExpectations(t)
	storage.AssertExpectations(t)
	llmClient.AssertExpectations(t)
	chunks.AssertExpectations(t)
}

// ─── ProcessJob: 不正な UUID ─────────────────────────────────────

func TestIngestUseCase_ProcessJob_InvalidJobID(t *testing.T) {
	ctx := context.Background()
	msg := validIngestMessage()
	msg.JobID = "not-a-uuid"

	files := &testhelper.MockFileRepository{}
	jobs := &testhelper.MockIngestJobRepository{}
	chunks := &testhelper.MockChunkRepository{}
	storage := &testhelper.MockObjectStorage{}
	llmClient := &testhelper.MockLLMClient{}

	uc := newIngestUseCase(files, jobs, chunks, storage, llmClient)
	err := uc.ProcessJob(ctx, msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid job_id")
	jobs.AssertNotCalled(t, "UpdateStatus")
}

func TestIngestUseCase_ProcessJob_InvalidFileID(t *testing.T) {
	ctx := context.Background()
	msg := validIngestMessage()
	msg.FileID = "not-a-uuid"

	files := &testhelper.MockFileRepository{}
	jobs := &testhelper.MockIngestJobRepository{}
	chunks := &testhelper.MockChunkRepository{}
	storage := &testhelper.MockObjectStorage{}
	llmClient := &testhelper.MockLLMClient{}

	uc := newIngestUseCase(files, jobs, chunks, storage, llmClient)
	err := uc.ProcessJob(ctx, msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid file_id")
}

// ─── ProcessJob: ダウンロード失敗 ────────────────────────────────

func TestIngestUseCase_ProcessJob_DownloadFails_MarksJobAndFileFailed(t *testing.T) {
	ctx := context.Background()
	msg := validIngestMessage()
	jobID := testhelper.FixtureJobID
	fileID := testhelper.FixtureFileID

	files := &testhelper.MockFileRepository{}
	jobs := &testhelper.MockIngestJobRepository{}
	chunks := &testhelper.MockChunkRepository{}
	storage := &testhelper.MockObjectStorage{}
	llmClient := &testhelper.MockLLMClient{}

	// 1. IngestJob → processing
	jobs.On("UpdateStatus", ctx, jobID, domain.JobStatusProcessing, (*string)(nil)).
		Return(testhelper.NewIngestJob(domain.JobStatusProcessing), nil)

	// 2. FileStatus → processing
	files.On("UpdateStatus", ctx, fileID, domain.FileStatusProcessing, (*string)(nil)).
		Return(testhelper.NewFile(domain.FileStatusProcessing), nil)

	// 3. ダウンロード失敗
	downloadErr := errors.New("MinIO connection refused")
	storage.On("Download", ctx, msg.StoragePath).
		Return((io.ReadCloser)(nil), downloadErr)

	// defer: job → failed, file → failed（errMsg はメッセージが入るため mock.Anything）
	jobs.On("UpdateStatus", ctx, jobID, domain.JobStatusFailed, mock.Anything).
		Return(testhelper.NewIngestJob(domain.JobStatusFailed), nil)
	files.On("UpdateStatus", ctx, fileID, domain.FileStatusFailed, mock.Anything).
		Return(testhelper.NewFile(domain.FileStatusFailed), nil)

	uc := newIngestUseCase(files, jobs, chunks, storage, llmClient)
	err := uc.ProcessJob(ctx, msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage download")

	// defer で failed にマークされること
	jobs.AssertCalled(t, "UpdateStatus", ctx, jobID, domain.JobStatusFailed, mock.Anything)
	files.AssertCalled(t, "UpdateStatus", ctx, fileID, domain.FileStatusFailed, mock.Anything)
	llmClient.AssertNotCalled(t, "OCRAndChunk")
}

// ─── ProcessJob: OCR チャンク0件 ─────────────────────────────────

func TestIngestUseCase_ProcessJob_OCRProducesNoChunks(t *testing.T) {
	ctx := context.Background()
	msg := validIngestMessage()
	jobID := testhelper.FixtureJobID
	fileID := testhelper.FixtureFileID

	files := &testhelper.MockFileRepository{}
	jobs := &testhelper.MockIngestJobRepository{}
	chunks := &testhelper.MockChunkRepository{}
	storage := &testhelper.MockObjectStorage{}
	llmClient := &testhelper.MockLLMClient{}

	jobs.On("UpdateStatus", ctx, jobID, domain.JobStatusProcessing, (*string)(nil)).
		Return(testhelper.NewIngestJob(domain.JobStatusProcessing), nil)
	files.On("UpdateStatus", ctx, fileID, domain.FileStatusProcessing, (*string)(nil)).
		Return(testhelper.NewFile(domain.FileStatusProcessing), nil)

	rc := io.NopCloser(bytes.NewReader(fakePDFContent))
	storage.On("Download", ctx, msg.StoragePath).Return(rc, nil)

	// OCR がチャンク0件を返す
	llmClient.On("OCRAndChunk", ctx, fakePDFContent, msg.MimeType).
		Return(&ports.OCRResult{Chunks: []ports.ChunkData{}}, nil)

	// defer: job → failed, file → failed
	jobs.On("UpdateStatus", ctx, jobID, domain.JobStatusFailed, mock.Anything).
		Return(testhelper.NewIngestJob(domain.JobStatusFailed), nil)
	files.On("UpdateStatus", ctx, fileID, domain.FileStatusFailed, mock.Anything).
		Return(testhelper.NewFile(domain.FileStatusFailed), nil)

	uc := newIngestUseCase(files, jobs, chunks, storage, llmClient)
	err := uc.ProcessJob(ctx, msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no chunks")
	chunks.AssertNotCalled(t, "BatchCreate")
}

// ─── ProcessJob: Embedding 全チャンク失敗 ────────────────────────

func TestIngestUseCase_ProcessJob_AllEmbeddingsFail(t *testing.T) {
	ctx := context.Background()
	msg := validIngestMessage()
	jobID := testhelper.FixtureJobID
	fileID := testhelper.FixtureFileID

	files := &testhelper.MockFileRepository{}
	jobs := &testhelper.MockIngestJobRepository{}
	chunks := &testhelper.MockChunkRepository{}
	storage := &testhelper.MockObjectStorage{}
	llmClient := &testhelper.MockLLMClient{}

	jobs.On("UpdateStatus", ctx, jobID, domain.JobStatusProcessing, (*string)(nil)).
		Return(testhelper.NewIngestJob(domain.JobStatusProcessing), nil)
	files.On("UpdateStatus", ctx, fileID, domain.FileStatusProcessing, (*string)(nil)).
		Return(testhelper.NewFile(domain.FileStatusProcessing), nil)

	rc := io.NopCloser(bytes.NewReader(fakePDFContent))
	storage.On("Download", ctx, msg.StoragePath).Return(rc, nil)

	pageNum := 1
	ocrResult := &ports.OCRResult{
		Chunks: []ports.ChunkData{
			{Index: 0, Content: "チャンクA", PageNumber: &pageNum},
		},
	}
	llmClient.On("OCRAndChunk", ctx, fakePDFContent, msg.MimeType).Return(ocrResult, nil)

	// Embedding が全チャンクで失敗
	embErr := errors.New("embedding API timeout")
	llmClient.On("GenerateEmbedding", ctx, "チャンクA").Return(([]float32)(nil), embErr)

	// defer: job → failed, file → failed
	jobs.On("UpdateStatus", ctx, jobID, domain.JobStatusFailed, mock.Anything).
		Return(testhelper.NewIngestJob(domain.JobStatusFailed), nil)
	files.On("UpdateStatus", ctx, fileID, domain.FileStatusFailed, mock.Anything).
		Return(testhelper.NewFile(domain.FileStatusFailed), nil)

	uc := newIngestUseCase(files, jobs, chunks, storage, llmClient)
	err := uc.ProcessJob(ctx, msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all chunks failed embedding")
	chunks.AssertNotCalled(t, "BatchCreate")
}

// ─── ProcessJob: BatchCreate 失敗 ────────────────────────────────

func TestIngestUseCase_ProcessJob_BatchCreateFails(t *testing.T) {
	ctx := context.Background()
	msg := validIngestMessage()
	jobID := testhelper.FixtureJobID
	fileID := testhelper.FixtureFileID

	files := &testhelper.MockFileRepository{}
	jobs := &testhelper.MockIngestJobRepository{}
	chunks := &testhelper.MockChunkRepository{}
	storage := &testhelper.MockObjectStorage{}
	llmClient := &testhelper.MockLLMClient{}

	jobs.On("UpdateStatus", ctx, jobID, domain.JobStatusProcessing, (*string)(nil)).
		Return(testhelper.NewIngestJob(domain.JobStatusProcessing), nil)
	files.On("UpdateStatus", ctx, fileID, domain.FileStatusProcessing, (*string)(nil)).
		Return(testhelper.NewFile(domain.FileStatusProcessing), nil)

	rc := io.NopCloser(bytes.NewReader(fakePDFContent))
	storage.On("Download", ctx, msg.StoragePath).Return(rc, nil)

	pageNum := 1
	ocrResult := &ports.OCRResult{
		Chunks: []ports.ChunkData{
			{Index: 0, Content: "チャンクX", PageNumber: &pageNum},
		},
	}
	llmClient.On("OCRAndChunk", ctx, fakePDFContent, msg.MimeType).Return(ocrResult, nil)
	llmClient.On("GenerateEmbedding", ctx, "チャンクX").Return(make([]float32, 768), nil)

	// DB バルク保存が失敗
	dbErr := errors.New("db write error")
	chunks.On("BatchCreate", ctx, mock.Anything).Return(dbErr)

	// defer: job → failed, file → failed
	jobs.On("UpdateStatus", ctx, jobID, domain.JobStatusFailed, mock.Anything).
		Return(testhelper.NewIngestJob(domain.JobStatusFailed), nil)
	files.On("UpdateStatus", ctx, fileID, domain.FileStatusFailed, mock.Anything).
		Return(testhelper.NewFile(domain.FileStatusFailed), nil)

	uc := newIngestUseCase(files, jobs, chunks, storage, llmClient)
	err := uc.ProcessJob(ctx, msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch create chunks")
	files.AssertCalled(t, "UpdateStatus", ctx, fileID, domain.FileStatusFailed, mock.Anything)
}
