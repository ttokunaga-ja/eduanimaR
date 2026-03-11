// Package ports はユースケース層が依存するインターフェース（抽象）を定義する。
// adapters パッケージがこれらを実装する。
package ports

import (
	"context"

	"github.com/google/uuid"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
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
	// excludeIDs: 既読チャンク ID リスト（DB レベルで除外）。空スライスの場合は除外なし。
	SearchByVector(ctx context.Context, subjectID uuid.UUID, embedding pgvector.Vector, limit int, excludeIDs []uuid.UUID) ([]*domain.SearchResult, error)
	// SearchByText: PostgreSQL 全文検索（subject_id で物理絞り込み）
	// excludeIDs: 既読チャンク ID リスト（DB レベルで除外）。空スライスの場合は除外なし。
	SearchByText(ctx context.Context, subjectID uuid.UUID, query string, limit int, excludeIDs []uuid.UUID) ([]*domain.SearchResult, error)
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

// ─── Chat Analytics ────────────────────────────────────────────────

// ChatAnalyticsUpdate は chats テーブルの analytics カラム更新データ
type ChatAnalyticsUpdate struct {
	// Answerability は質問への回答可否: "answerable" | "unanswerable" | "partial"
	Answerability string
	// DocumentSummary はコースマテリアルの概要
	DocumentSummary string
	// LoopTerminationReason はループ終了理由: "sufficient" | "loop_limit" | "error" | "no_evidence"
	LoopTerminationReason string
	// TotalLoopCount は実行した検索ループ数
	TotalLoopCount int
	// LibrarianDurationMS は Librarian gRPC ストリーミング全体の実行時間（ミリ秒）
	LibrarianDurationMS int
	// ProfessorDurationMS は最終回答生成の実行時間（ミリ秒）
	ProfessorDurationMS int
}

// ChatLoopDetail は chat_loop_details テーブルの 1 行分データ
type ChatLoopDetail struct {
	// ChatID は対応する chats.id
	ChatID uuid.UUID
	// LoopNumber は 1 始まりのループ番号
	LoopNumber int
	// QueriesText は SearchAction.queries_text（Librarian が生成したクエリリスト）
	QueriesText []string
	// IsSufficient は SubAgent-C の充足判断（proto 変更前は nil）
	IsSufficient *bool
	// MissingKeywords は SubAgent-C が検出した不足キーワード（proto 変更前は空スライス）
	MissingKeywords []string
}

// ChatAccumulatedChunk は chat_accumulated_chunks テーブルの 1 行分データ
type ChatAccumulatedChunk struct {
	// ChatID は対応する chats.id
	ChatID uuid.UUID
	// LoopNumber は対応する検索ループ番号
	LoopNumber int
	// ChunkID は materials.id の UUID 文字列
	ChunkID string
	// FileName はチャンクが属するファイル名
	FileName string
	// PageNumber はページ番号（nil = 不明）
	PageNumber *string
	// SearchScore はハイブリッド検索スコア（nil = 未記録）
	SearchScore *float32
	// TextSnippet はチャンクの全文（is_useful=TRUE のみ設定、FALSE は nil）
	TextSnippet *string
	// IsUseful はエビデンスとして選択されたかどうか
	IsUseful bool
}

// ChatAnalyticsRepository は chat analytics テーブル群の書き込みを抽象化する。
// ChatUseCase に WithAnalyticsRepo() でオプションとして注入する。
type ChatAnalyticsRepository interface {
	// UpdateChatAnalytics は chats テーブルの analytics カラムを更新する。
	// chat_id は永続化済みの chats.id を指定する。
	UpdateChatAnalytics(ctx context.Context, chatID uuid.UUID, data ChatAnalyticsUpdate) error

	// InsertLoopDetail は 1 ループ分の詳細を chat_loop_details に挿入する。
	// onConflict=IGNORE（同一 chat_id + loop_number は無視する）
	InsertLoopDetail(ctx context.Context, detail ChatLoopDetail) error

	// InsertAccumulatedChunks は複数チャンクを chat_accumulated_chunks にバッチ挿入する。
	// onConflict=IGNORE（同一 chat_id + loop_number + chunk_id は無視する）
	InsertAccumulatedChunks(ctx context.Context, chunks []ChatAccumulatedChunk) error
}
