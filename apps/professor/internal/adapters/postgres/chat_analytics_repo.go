package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/ports"
)

// chatAnalyticsRepo は ports.ChatAnalyticsRepository の postgres 実装。
// sqlcgen を使わず raw SQL で実装する（新規テーブルのため sqlcgen 再生成が不要）。
type chatAnalyticsRepo struct {
	db *sql.DB
}

// NewChatAnalyticsRepo は ports.ChatAnalyticsRepository の postgres 実装を返す。
func NewChatAnalyticsRepo(db *sql.DB) ports.ChatAnalyticsRepository {
	return &chatAnalyticsRepo{db: db}
}

// UpdateChatAnalytics は chats テーブルの analytics カラムを更新する。
// 002_add_chat_analytics.sql で追加された 6 カラムを SET する。
func (r *chatAnalyticsRepo) UpdateChatAnalytics(ctx context.Context, chatID uuid.UUID, data ports.ChatAnalyticsUpdate) error {
	const query = `
UPDATE chats SET
    answerability           = $2::answerability_enum,
    document_summary        = $3,
    loop_termination_reason = $4::loop_termination_enum,
    total_loop_count        = $5,
    librarian_duration_ms   = $6,
    professor_duration_ms   = $7,
    updated_at              = NOW()
WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		chatID,
		nullableStr(data.Answerability),
		nullableStr(data.DocumentSummary),
		nullableStr(data.LoopTerminationReason),
		nullableInt(data.TotalLoopCount),
		nullableInt(data.LibrarianDurationMS),
		nullableInt(data.ProfessorDurationMS),
	)
	if err != nil {
		return fmt.Errorf("chat_analytics_repo: update analytics: %w", err)
	}
	return nil
}

// InsertLoopDetail は 1 ループ分の詳細を chat_loop_details に挿入する。
// ON CONFLICT DO NOTHING: 同一 chat_id + loop_number は無視（冪等性保証）。
func (r *chatAnalyticsRepo) InsertLoopDetail(ctx context.Context, detail ports.ChatLoopDetail) error {
	queriesJSON, err := json.Marshal(detail.QueriesText)
	if err != nil {
		queriesJSON = []byte("[]")
	}

	missingJSON, err := json.Marshal(detail.MissingKeywords)
	if err != nil {
		missingJSON = []byte("[]")
	}

	const query = `
INSERT INTO chat_loop_details (chat_id, loop_number, queries_text, is_sufficient, missing_keywords)
VALUES ($1, $2, $3::jsonb, $4, $5::jsonb)
ON CONFLICT (chat_id, loop_number) DO NOTHING`

	_, err = r.db.ExecContext(ctx, query,
		detail.ChatID,
		detail.LoopNumber,
		queriesJSON,
		detail.IsSufficient, // *bool: nil → NULL
		missingJSON,
	)
	if err != nil {
		return fmt.Errorf("chat_analytics_repo: insert loop detail (loop=%d): %w", detail.LoopNumber, err)
	}
	return nil
}

// InsertAccumulatedChunks は複数チャンクを chat_accumulated_chunks にバッチ挿入する。
// ON CONFLICT DO NOTHING: 同一 chat_id + loop_number + chunk_id は無視（冪等性保証）。
// チャンクが空の場合は何もしない。
func (r *chatAnalyticsRepo) InsertAccumulatedChunks(ctx context.Context, chunks []ports.ChatAccumulatedChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	const query = `
INSERT INTO chat_accumulated_chunks
    (chat_id, loop_number, chunk_id, file_name, page_number, search_score, text_snippet, is_useful)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (chat_id, loop_number, chunk_id) DO NOTHING`

	for _, c := range chunks {
		_, err := r.db.ExecContext(ctx, query,
			c.ChatID,
			c.LoopNumber,
			c.ChunkID,
			c.FileName,
			c.PageNumber,  // *string: nil → NULL
			c.SearchScore, // *float32: nil → NULL
			c.TextSnippet, // *string: nil → NULL（is_useful=FALSE の場合）
			c.IsUseful,
		)
		if err != nil {
			return fmt.Errorf("chat_analytics_repo: insert chunk %s (loop=%d): %w", c.ChunkID, c.LoopNumber, err)
		}
	}
	return nil
}

// UpdateQuestionAnalysis は chats テーブルの question_analysis カラムを更新する。
// 003_add_question_analysis.sql で追加された 4 カラムを SET する。
func (r *chatAnalyticsRepo) UpdateQuestionAnalysis(ctx context.Context, chatID uuid.UUID, analysis ports.QuestionAnalysisUpdate) error {
	criteriaJSON, err := json.Marshal(analysis.CompletionCriteria)
	if err != nil {
		criteriaJSON = []byte("[]")
	}

	var optionsJSON []byte
	if len(analysis.ClarificationOptions) > 0 {
		optionsJSON, err = json.Marshal(analysis.ClarificationOptions)
		if err != nil {
			optionsJSON = []byte("null")
		}
	} else {
		optionsJSON = []byte("null")
	}

	const query = `
UPDATE chats SET
    question_clarity      = $2::question_clarity_enum,
    interpreted_query     = $3,
    completion_criteria   = $4::jsonb,
    clarification_options = $5::jsonb,
    updated_at            = NOW()
WHERE id = $1`

	_, err = r.db.ExecContext(ctx, query,
		chatID,
		nullableStr(analysis.Clarity),
		nullableStr(analysis.InterpretedQuery),
		criteriaJSON,
		optionsJSON,
	)
	if err != nil {
		return fmt.Errorf("chat_analytics_repo: update question analysis: %w", err)
	}
	return nil
}

// ─── ヘルパー ─────────────────────────────────────────────────────

// nullableStr は空文字列を sql.NullString の NULL として扱うヘルパー。
// answerability や loop_termination_reason が未設定の場合に DB へ NULL を書き込む。
func nullableStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullableInt は 0 を sql.NullInt32 の NULL として扱うヘルパー。
func nullableInt(n int) sql.NullInt32 {
	if n == 0 {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: int32(n), Valid: true}
}
