// Package ports はユースケース層が依存するインターフェース（抽象）を定義する。
// adapters パッケージがこれらを実装する。
package ports

import (
	"context"

	"github.com/google/uuid"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
)

// UserRepository はユーザーの永続化操作を抽象化する
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
}

// SubjectRepository は科目の永続化操作を抽象化する
type SubjectRepository interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Subject, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Subject, error)
	GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.Subject, error)
	Create(ctx context.Context, subject *domain.Subject) error
	UpdateName(ctx context.Context, id, userID uuid.UUID, name string) (*domain.Subject, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// FileRepository はアップロードファイルの永続化操作を抽象化する
type FileRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.File, error)
	GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.File, error)
	ListBySubjectID(ctx context.Context, subjectID uuid.UUID) ([]*domain.File, error)
	Create(ctx context.Context, file *domain.File) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.FileStatus, errMsg *string) (*domain.File, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// ChunkRepository はチャンク（pgvector）の永続化・検索操作を抽象化する
type ChunkRepository interface {
	ListByFileID(ctx context.Context, fileID uuid.UUID) ([]*domain.Chunk, error)
	BatchCreate(ctx context.Context, chunks []*domain.Chunk) error
	// SearchByVector: HNSW コサイン類似度検索（subject_id で物理絞り込み）
	SearchByVector(ctx context.Context, subjectID uuid.UUID, embedding pgvector.Vector, limit int) ([]*domain.SearchResult, error)
	// SearchByText: PostgreSQL 全文検索（subject_id で物理絞り込み）
	SearchByText(ctx context.Context, subjectID uuid.UUID, query string, limit int) ([]*domain.SearchResult, error)
	DeleteByFileID(ctx context.Context, fileID uuid.UUID) error
}

// IngestJobRepository はインジェストジョブの永続化操作を抽象化する
type IngestJobRepository interface {
	Create(ctx context.Context, job *domain.IngestJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.IngestJob, error)
	GetByFileID(ctx context.Context, fileID uuid.UUID) (*domain.IngestJob, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus, errMsg *string) (*domain.IngestJob, error)
}

// QASessionRepository は質問応答セッションの永続化操作を抽象化する
type QASessionRepository interface {
	Create(ctx context.Context, session *domain.QASession) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.QASession, error)
	GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.QASession, error)
	ListBySubjectID(ctx context.Context, subjectID, userID uuid.UUID, limit, offset int) ([]*domain.QASession, error)
	CountBySubjectID(ctx context.Context, subjectID, userID uuid.UUID) (int64, error)
	UpdateAnswer(ctx context.Context, id uuid.UUID, answer string, sources []domain.Source) (*domain.QASession, error)
	UpdateFeedback(ctx context.Context, id, userID uuid.UUID, feedback int) (*domain.QASession, error)
}
