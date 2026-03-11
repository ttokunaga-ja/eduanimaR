-- 003_add_question_analysis.sql
-- Professor が Librarian 呼び出し前に実行する「質問分析」結果を chats テーブルに追加する。
-- clarity: 質問の明確さ ENUM（"clear" = 検索実行 / "ambiguous" = 選択肢提示）
-- interpreted_query: Professor が解釈した質問文
-- completion_criteria: 検索完了の終了基準リスト（JSONB []string）
-- clarification_options: 曖昧な質問に対する選択肢リスト（JSONB []string）

-- ─── ENUM ─────────────────────────────────────────────────────
CREATE TYPE question_clarity_enum AS ENUM ('clear', 'ambiguous');

-- ─── chats テーブルにカラム追加 ────────────────────────────────
ALTER TABLE chats
    ADD COLUMN question_clarity      question_clarity_enum,
    ADD COLUMN interpreted_query     TEXT,
    ADD COLUMN completion_criteria   JSONB,
    ADD COLUMN clarification_options JSONB;

-- インデックス: 曖昧な質問の集計・分析用
CREATE INDEX idx_chats_question_clarity
    ON chats(question_clarity)
    WHERE is_active AND question_clarity IS NOT NULL;
