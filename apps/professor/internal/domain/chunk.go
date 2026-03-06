package domain

import (
	"time"

	"github.com/google/uuid"
	pgvector "github.com/pgvector/pgvector-go"
)

// Chunk は OCR/構造化 後のテキストチャンク（ベクトル検索の最小単位）
type Chunk struct {
	ID         uuid.UUID
	FileID     uuid.UUID
	SubjectID  uuid.UUID
	PageNumber *int            // PDF ページ番号（画像スライドは nil）
	ChunkIndex int             // ファイル内連番
	Content    string          // OCR/抽出テキスト
	Embedding  pgvector.Vector // Gemini Embedding（768次元）
	CreatedAt  time.Time
}

// SearchResult は検索クエリに対するチャンク検索結果
// embedding フィールドを除いた軽量な構造体（通信コスト削減）
type SearchResult struct {
	ChunkID    uuid.UUID
	FileID     uuid.UUID
	SubjectID  uuid.UUID
	PageNumber *int
	ChunkIndex int
	Content    string
	FileName   string // JOIN で取得（files.name）
	CreatedAt  time.Time
}
