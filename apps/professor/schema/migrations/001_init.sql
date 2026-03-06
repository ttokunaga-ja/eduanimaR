-- ===================================================================
-- 001_init.sql
-- eduanima-professor Phase 1 初期スキーマ
-- 前提: PostgreSQL 18 + pgvector 0.8.1
-- 適用方法: atlas migrate apply --dir "file://schema/migrations" --url $DATABASE_URL
-- ===================================================================

-- ── 拡張機能 ────────────────────────────────────────────────────────
-- pgvector: HNSW インデックス・vector 型・<=> 演算子（コサイン類似度）に必要
-- PG18 でも pgvector は引き続き必要（HNSW / IVFFlat の実装は pgvector が提供）
CREATE EXTENSION IF NOT EXISTS "vector";      -- pgvector 0.8.1 (pg18 対応)

-- UUIDv7: PG18 組み込みネイティブ関数 uuidv7() を使用（拡張インストール不要）
-- 時系列ソート可能・B-tree インデックス効率が UUID v4 より優れる

-- ── ENUM 型定義 ──────────────────────────────────────────────────────
CREATE TYPE auth_provider AS ENUM (
    'google',
    'meta',
    'microsoft',
    'line'
);

CREATE TYPE file_status AS ENUM (
    'pending',
    'processing',
    'ready',
    'failed'
);

CREATE TYPE job_status AS ENUM (
    'pending',
    'processing',
    'completed',
    'failed'
);

-- ── users ────────────────────────────────────────────────────────────
-- SSO カラム (provider, provider_user_id) は Phase 1 で NULLABLE として先行定義。
-- Phase 2 (SSO) 移行時に ALTER TABLE 不要。
CREATE TABLE users (
    user_id          UUID         NOT NULL DEFAULT uuidv7(),
    email            TEXT         NOT NULL,
    -- Phase 2 で使用（SSOプロバイダー識別）
    provider         auth_provider         NULL,
    provider_user_id TEXT                  NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT users_pkey            PRIMARY KEY (user_id),
    CONSTRAINT users_email_unique    UNIQUE (email),
    -- provider が両方 NOT NULL の場合のみ一意制約を適用（NULL同士は除外）
    CONSTRAINT users_provider_unique UNIQUE NULLS NOT DISTINCT (provider, provider_user_id)
);

-- Phase 1 固定開発ユーザー（X-Dev-User: dev-user-001 ヘッダーに対応）
INSERT INTO users (user_id, email)
VALUES ('00000000-0000-0000-0000-000000000001', 'dev@example.com')
ON CONFLICT DO NOTHING;

-- ── subjects ─────────────────────────────────────────────────────────
CREATE TABLE subjects (
    subject_id    UUID        NOT NULL DEFAULT uuidv7(),
    user_id       UUID        NOT NULL,
    name          TEXT        NOT NULL,
    lms_course_id TEXT        NULL,    -- 将来の LMS 連携用（Phase 1 未使用）
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT subjects_pkey    PRIMARY KEY (subject_id),
    CONSTRAINT subjects_user_fk FOREIGN KEY (user_id)
        REFERENCES users (user_id) ON DELETE CASCADE
);

CREATE INDEX idx_subjects_user_id ON subjects (user_id);

-- ── files ─────────────────────────────────────────────────────────────
-- storage_path: Phase 1 は MinIO パス / Phase 2 は GCS パス
CREATE TABLE files (
    file_id       UUID        NOT NULL DEFAULT uuidv7(),
    subject_id    UUID        NOT NULL,
    user_id       UUID        NOT NULL,
    name          TEXT        NOT NULL,
    storage_path  TEXT        NOT NULL,   -- minio://bucket/path (Phase 1)
    mime_type     TEXT        NOT NULL,
    size_bytes    BIGINT      NOT NULL,
    status        file_status NOT NULL DEFAULT 'pending',
    error_message TEXT        NULL,
    uploaded_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at  TIMESTAMPTZ NULL,

    CONSTRAINT files_pkey       PRIMARY KEY (file_id),
    CONSTRAINT files_subject_fk FOREIGN KEY (subject_id)
        REFERENCES subjects (subject_id) ON DELETE CASCADE,
    CONSTRAINT files_user_fk    FOREIGN KEY (user_id)
        REFERENCES users (user_id) ON DELETE CASCADE
);

CREATE INDEX idx_files_subject_id ON files (subject_id);
CREATE INDEX idx_files_user_id    ON files (user_id);
CREATE INDEX idx_files_status     ON files (status);

-- ── chunks ────────────────────────────────────────────────────────────
-- pgvector HNSW インデックス（コサイン類似度検索）
-- embedding 次元数: 768（Gemini Embedding）
CREATE TABLE chunks (
    chunk_id    UUID         NOT NULL DEFAULT uuidv7(),
    file_id     UUID         NOT NULL,
    subject_id  UUID         NOT NULL,
    page_number INT          NULL,     -- PDF ページ番号（画像スライドは NULL）
    chunk_index INT          NOT NULL, -- ファイル内連番
    content     TEXT         NOT NULL, -- OCR/抽出テキスト
    embedding   vector(768)  NOT NULL, -- Gemini Embedding（768次元）
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chunks_pkey       PRIMARY KEY (chunk_id),
    CONSTRAINT chunks_file_fk    FOREIGN KEY (file_id)
        REFERENCES files (file_id) ON DELETE CASCADE,
    CONSTRAINT chunks_subject_fk FOREIGN KEY (subject_id)
        REFERENCES subjects (subject_id) ON DELETE CASCADE
);

CREATE INDEX idx_chunks_file_id    ON chunks (file_id);
CREATE INDEX idx_chunks_subject_id ON chunks (subject_id);
-- HNSW インデックス（m=16, ef_construction=64 はデフォルト推奨値）
CREATE INDEX idx_chunks_embedding_hnsw
    ON chunks USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- ── ingest_jobs ───────────────────────────────────────────────────────
-- Kafka 非同期パイプライン（OCR/Embedding）のジョブ管理テーブル
CREATE TABLE ingest_jobs (
    job_id        UUID       NOT NULL DEFAULT uuidv7(),
    file_id       UUID       NOT NULL,
    status        job_status NOT NULL DEFAULT 'pending',
    retry_count   INT        NOT NULL DEFAULT 0,
    max_retries   INT        NOT NULL DEFAULT 3,
    error_message TEXT       NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at    TIMESTAMPTZ NULL,
    completed_at  TIMESTAMPTZ NULL,

    CONSTRAINT ingest_jobs_pkey    PRIMARY KEY (job_id),
    CONSTRAINT ingest_jobs_file_fk FOREIGN KEY (file_id)
        REFERENCES files (file_id) ON DELETE CASCADE
);

CREATE INDEX idx_ingest_jobs_status  ON ingest_jobs (status);
CREATE INDEX idx_ingest_jobs_file_id ON ingest_jobs (file_id);

-- ── qa_sessions ───────────────────────────────────────────────────────
-- Q&A セッション（SSE ストリーミング完了後に answer を永続化）
CREATE TABLE qa_sessions (
    session_id  UUID        NOT NULL DEFAULT uuidv7(),
    user_id     UUID        NOT NULL,
    subject_id  UUID        NOT NULL,
    question    TEXT        NOT NULL,
    answer      TEXT        NULL,    -- SSE 完了後に保存
    sources     JSONB       NULL,    -- [{file_id, chunk_id, page_number, excerpt}]
    feedback    SMALLINT    NULL,    -- -1: bad, 1: good (NULL: 未評価)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    answered_at TIMESTAMPTZ NULL,

    CONSTRAINT qa_sessions_pkey          PRIMARY KEY (session_id),
    CONSTRAINT qa_sessions_user_fk       FOREIGN KEY (user_id)
        REFERENCES users (user_id) ON DELETE CASCADE,
    CONSTRAINT qa_sessions_subject_fk    FOREIGN KEY (subject_id)
        REFERENCES subjects (subject_id) ON DELETE CASCADE,
    CONSTRAINT qa_sessions_feedback_chk  CHECK (feedback IS NULL OR feedback IN (-1, 1))
);

CREATE INDEX idx_qa_sessions_user_id    ON qa_sessions (user_id);
CREATE INDEX idx_qa_sessions_subject_id ON qa_sessions (subject_id);
CREATE INDEX idx_qa_sessions_created_at ON qa_sessions (created_at DESC);
