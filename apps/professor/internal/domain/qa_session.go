package domain

import (
	"time"

	"github.com/google/uuid"
)

// QASession は質問応答セッションエンティティ
type QASession struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	SubjectID  uuid.UUID
	Question   string
	Answer     *string  // SSE ストリーミング完了後に保存
	Sources    []Source // JSONB として永続化
	Feedback   *int     // -1: bad, 1: good, nil: 未評価
	CreatedAt  time.Time
	AnsweredAt *time.Time
}

// Source は回答の参照元チャンク情報
type Source struct {
	FileID     uuid.UUID `json:"file_id"`
	ChunkID    uuid.UUID `json:"chunk_id"`
	FileName   string    `json:"file_name"`
	PageNumber *int      `json:"page_number,omitempty"`
	Excerpt    string    `json:"excerpt"` // 抜粋テキスト（最大 300 文字程度）
}

// SSEEvent は SSE ストリーミングで送信するイベント型
type SSEEventType string

const (
	SSEEventThinking  SSEEventType = "thinking"  // Librarian が推論中
	SSEEventSearching SSEEventType = "searching" // 検索クエリ実行中
	SSEEventEvidence  SSEEventType = "evidence"  // エビデンスチャンク発見
	SSEEventAnswer    SSEEventType = "answer"    // 回答テキスト（チャンク送信）
	SSEEventDone      SSEEventType = "done"      // ストリーミング完了
	SSEEventError     SSEEventType = "error"     // エラー発生
)
