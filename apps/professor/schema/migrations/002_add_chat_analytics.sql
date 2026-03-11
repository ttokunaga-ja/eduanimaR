-- ===================================================================
-- 002_add_chat_analytics.sql
-- chat analytics: チャットセッションの解析データ記録
-- 前提: PostgreSQL 18 + 001_init.sql 実行済み
-- ===================================================================

-- answerability_enum: 質問への回答可否を表す ENUM
CREATE TYPE answerability_enum AS ENUM ('answerable', 'unanswerable', 'partial');

-- loop_termination_enum: Librarian ループ終了理由を表す ENUM
CREATE TYPE loop_termination_enum AS ENUM ('sufficient', 'loop_limit', 'error', 'no_evidence');

-- chats テーブルに analytics カラムを追加
ALTER TABLE chats
    ADD COLUMN answerability           answerability_enum,
    ADD COLUMN document_summary        TEXT,
    ADD COLUMN loop_termination_reason loop_termination_enum,
    ADD COLUMN total_loop_count        INTEGER,
    ADD COLUMN librarian_duration_ms   INTEGER,
    ADD COLUMN professor_duration_ms   INTEGER;

CREATE INDEX idx_chats_answerability ON chats(answerability)
    WHERE is_active AND answerability IS NOT NULL;

-- chat_loop_details: Librarian の各検索ループ詳細
--   loop_number: 1始まりのループ番号（searchRounds と一致）
--   queries_text: SearchAction で送信したクエリのリスト（JSONB）
--   is_sufficient: evaluate_parallel の SubAgent-C 判断（proto変更前は NULL）
--   missing_keywords: 不足キーワードのリスト（proto変更前は []）
CREATE TABLE chat_loop_details (
    id               UUID        PRIMARY KEY DEFAULT uuidv7(),
    chat_id          UUID        NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    loop_number      INTEGER     NOT NULL,
    queries_text     JSONB       NOT NULL DEFAULT '[]'::jsonb,
    is_sufficient    BOOLEAN,
    missing_keywords JSONB       NOT NULL DEFAULT '[]'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(chat_id, loop_number)
);

CREATE INDEX idx_chat_loop_details_chat_id ON chat_loop_details(chat_id);

-- chat_accumulated_chunks: Librarian が蓄積したチャンクの記録
--   is_useful=FALSE: chunk_id/file_name/page_number/search_score のみ保存（text_snippet=NULL）
--   is_useful=TRUE:  全フィールド保存（text_snippet = chunk の全文）
--   chunk_id は materials.id の UUID 文字列表現（TEXT 型で保存）
CREATE TABLE chat_accumulated_chunks (
    id           UUID     PRIMARY KEY DEFAULT uuidv7(),
    chat_id      UUID     NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    loop_number  INTEGER  NOT NULL,
    chunk_id     TEXT     NOT NULL,
    file_name    TEXT     NOT NULL,
    page_number  TEXT,
    search_score REAL,
    text_snippet TEXT,
    is_useful    BOOLEAN  NOT NULL DEFAULT TRUE,
    UNIQUE(chat_id, loop_number, chunk_id)
);

CREATE INDEX idx_chat_accumulated_chunks_chat_id ON chat_accumulated_chunks(chat_id);
CREATE INDEX idx_chat_accumulated_chunks_useful ON chat_accumulated_chunks(chat_id, is_useful);
