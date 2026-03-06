// Package testhelper はテスト専用のヘルパー（モック・フィクスチャ）を提供する。
// このパッケージは _test.go 以外のファイルからは import しないこと。
package testhelper

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	pgvector "github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/mock"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
)

// ─── SubjectRepository ────────────────────────────────────────────

type MockSubjectRepository struct{ mock.Mock }

func (m *MockSubjectRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Subject, error) {
	args := m.Called(ctx, userID)
	v, _ := args.Get(0).([]*domain.Subject)
	return v, args.Error(1)
}
func (m *MockSubjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subject, error) {
	args := m.Called(ctx, id)
	v, _ := args.Get(0).(*domain.Subject)
	return v, args.Error(1)
}
func (m *MockSubjectRepository) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.Subject, error) {
	args := m.Called(ctx, id, userID)
	v, _ := args.Get(0).(*domain.Subject)
	return v, args.Error(1)
}
func (m *MockSubjectRepository) Create(ctx context.Context, subject *domain.Subject) error {
	return m.Called(ctx, subject).Error(0)
}
func (m *MockSubjectRepository) UpdateName(ctx context.Context, id, userID uuid.UUID, name string) (*domain.Subject, error) {
	args := m.Called(ctx, id, userID, name)
	v, _ := args.Get(0).(*domain.Subject)
	return v, args.Error(1)
}
func (m *MockSubjectRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return m.Called(ctx, id, userID).Error(0)
}

// ─── QASessionRepository ──────────────────────────────────────────

type MockQASessionRepository struct{ mock.Mock }

func (m *MockQASessionRepository) Create(ctx context.Context, session *domain.QASession) error {
	return m.Called(ctx, session).Error(0)
}
func (m *MockQASessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.QASession, error) {
	args := m.Called(ctx, id)
	v, _ := args.Get(0).(*domain.QASession)
	return v, args.Error(1)
}
func (m *MockQASessionRepository) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.QASession, error) {
	args := m.Called(ctx, id, userID)
	v, _ := args.Get(0).(*domain.QASession)
	return v, args.Error(1)
}
func (m *MockQASessionRepository) ListBySubjectID(ctx context.Context, subjectID, userID uuid.UUID, limit, offset int) ([]*domain.QASession, error) {
	args := m.Called(ctx, subjectID, userID, limit, offset)
	v, _ := args.Get(0).([]*domain.QASession)
	return v, args.Error(1)
}
func (m *MockQASessionRepository) CountBySubjectID(ctx context.Context, subjectID, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, subjectID, userID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockQASessionRepository) UpdateAnswer(ctx context.Context, id uuid.UUID, answer string, sources []domain.Source) (*domain.QASession, error) {
	args := m.Called(ctx, id, answer, sources)
	v, _ := args.Get(0).(*domain.QASession)
	return v, args.Error(1)
}
func (m *MockQASessionRepository) UpdateFeedback(ctx context.Context, id, userID uuid.UUID, feedback int) (*domain.QASession, error) {
	args := m.Called(ctx, id, userID, feedback)
	v, _ := args.Get(0).(*domain.QASession)
	return v, args.Error(1)
}

// ─── ChunkRepository ──────────────────────────────────────────────

type MockChunkRepository struct{ mock.Mock }

func (m *MockChunkRepository) ListByFileID(ctx context.Context, fileID uuid.UUID) ([]*domain.Chunk, error) {
	args := m.Called(ctx, fileID)
	v, _ := args.Get(0).([]*domain.Chunk)
	return v, args.Error(1)
}
func (m *MockChunkRepository) BatchCreate(ctx context.Context, chunks []*domain.Chunk) error {
	return m.Called(ctx, chunks).Error(0)
}
func (m *MockChunkRepository) SearchByVector(ctx context.Context, subjectID uuid.UUID, embedding pgvector.Vector, limit int) ([]*domain.SearchResult, error) {
	args := m.Called(ctx, subjectID, embedding, limit)
	v, _ := args.Get(0).([]*domain.SearchResult)
	return v, args.Error(1)
}
func (m *MockChunkRepository) SearchByText(ctx context.Context, subjectID uuid.UUID, query string, limit int) ([]*domain.SearchResult, error) {
	args := m.Called(ctx, subjectID, query, limit)
	v, _ := args.Get(0).([]*domain.SearchResult)
	return v, args.Error(1)
}
func (m *MockChunkRepository) DeleteByFileID(ctx context.Context, fileID uuid.UUID) error {
	return m.Called(ctx, fileID).Error(0)
}

// ─── LLMClient ───────────────────────────────────────────────────

type MockLLMClient struct{ mock.Mock }

func (m *MockLLMClient) OCRAndChunk(ctx context.Context, fileContent []byte, mimeType string) (*ports.OCRResult, error) {
	args := m.Called(ctx, fileContent, mimeType)
	v, _ := args.Get(0).(*ports.OCRResult)
	return v, args.Error(1)
}
func (m *MockLLMClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	v, _ := args.Get(0).([]float32)
	return v, args.Error(1)
}
func (m *MockLLMClient) GenerateAnswer(ctx context.Context, question string, evidences []string) (string, error) {
	args := m.Called(ctx, question, evidences)
	return args.String(0), args.Error(1)
}
func (m *MockLLMClient) GenerateAnswerStream(ctx context.Context, question string, evidences []string, onChunk func(text string) error) error {
	args := m.Called(ctx, question, evidences, onChunk)
	return args.Error(0)
}

// ─── LibrarianClient ─────────────────────────────────────────────

type MockLibrarianClient struct{ mock.Mock }

func (m *MockLibrarianClient) Think(
	ctx context.Context,
	requestID string,
	userQuery string,
	subjectID uuid.UUID,
	userID uuid.UUID,
	onSearchRequest func(req ports.LibrarianSearchRequest) (*ports.LibrarianSearchResponse, error),
) (*ports.LibrarianThinkResult, error) {
	args := m.Called(ctx, requestID, userQuery, subjectID, userID, onSearchRequest)
	v, _ := args.Get(0).(*ports.LibrarianThinkResult)
	return v, args.Error(1)
}

// ─── FileRepository ───────────────────────────────────────────────

type MockFileRepository struct{ mock.Mock }

func (m *MockFileRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.File, error) {
	args := m.Called(ctx, id)
	v, _ := args.Get(0).(*domain.File)
	return v, args.Error(1)
}
func (m *MockFileRepository) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.File, error) {
	args := m.Called(ctx, id, userID)
	v, _ := args.Get(0).(*domain.File)
	return v, args.Error(1)
}
func (m *MockFileRepository) ListBySubjectID(ctx context.Context, subjectID uuid.UUID) ([]*domain.File, error) {
	args := m.Called(ctx, subjectID)
	v, _ := args.Get(0).([]*domain.File)
	return v, args.Error(1)
}
func (m *MockFileRepository) Create(ctx context.Context, file *domain.File) error {
	return m.Called(ctx, file).Error(0)
}
func (m *MockFileRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.FileStatus, errMsg *string) (*domain.File, error) {
	args := m.Called(ctx, id, status, errMsg)
	v, _ := args.Get(0).(*domain.File)
	return v, args.Error(1)
}
func (m *MockFileRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return m.Called(ctx, id, userID).Error(0)
}

// ─── IngestJobRepository ─────────────────────────────────────────

type MockIngestJobRepository struct{ mock.Mock }

func (m *MockIngestJobRepository) Create(ctx context.Context, job *domain.IngestJob) error {
	return m.Called(ctx, job).Error(0)
}
func (m *MockIngestJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.IngestJob, error) {
	args := m.Called(ctx, id)
	v, _ := args.Get(0).(*domain.IngestJob)
	return v, args.Error(1)
}
func (m *MockIngestJobRepository) GetByFileID(ctx context.Context, fileID uuid.UUID) (*domain.IngestJob, error) {
	args := m.Called(ctx, fileID)
	v, _ := args.Get(0).(*domain.IngestJob)
	return v, args.Error(1)
}
func (m *MockIngestJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus, errMsg *string) (*domain.IngestJob, error) {
	args := m.Called(ctx, id, status, errMsg)
	v, _ := args.Get(0).(*domain.IngestJob)
	return v, args.Error(1)
}

// ─── ObjectStorage ────────────────────────────────────────────────

type MockObjectStorage struct{ mock.Mock }

func (m *MockObjectStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	args := m.Called(ctx, key, reader, size, contentType)
	return args.String(0), args.Error(1)
}
func (m *MockObjectStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, key)
	v, _ := args.Get(0).(io.ReadCloser)
	return v, args.Error(1)
}
func (m *MockObjectStorage) Delete(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}
func (m *MockObjectStorage) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, key, expiry)
	return args.String(0), args.Error(1)
}
