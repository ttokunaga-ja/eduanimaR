-- ===================================================================
-- 001_init.sql
-- eduanima-professor pre-release canonical schema (SSOT: DB_SCHEMA_TABLES.md)
-- 前提: PostgreSQL 18 + pgvector 0.8.1
-- ===================================================================

CREATE EXTENSION IF NOT EXISTS "vector";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE OR REPLACE FUNCTION nanoid20() RETURNS TEXT AS $$
  SELECT substring(translate(encode(gen_random_bytes(20), 'base64'), E'+/=\n', '____') for 20);
$$ LANGUAGE SQL;

CREATE TYPE user_role AS ENUM ('student', 'instructor', 'admin');

CREATE TYPE file_type AS ENUM (
  'pdf', 'text',
  'python', 'go', 'javascript', 'html', 'css', 'json', 'markdown', 'csv',
  'png', 'jpeg', 'webp', 'heic', 'heif',
  'docx', 'xlsx', 'pptx',
  'google_docs', 'google_sheets', 'google_slides',
  'other'
);

CREATE TYPE file_status AS ENUM (
  'uploading', 'uploaded', 'processing', 'completed', 'failed', 'archived'
);

CREATE TYPE job_type AS ENUM (
  'file_ingestion', 'ai_batch_processing', 'search_optimization', 'maintenance'
);

CREATE TYPE job_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled');
CREATE TYPE search_mode AS ENUM ('keyword', 'vector', 'hybrid');
CREATE TYPE chat_feedback AS ENUM ('good', 'bad');
CREATE TYPE gemini_phase AS ENUM ('ingestion', 'planning', 'search', 'answer');

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  nanoid TEXT NOT NULL UNIQUE DEFAULT nanoid20() CHECK (length(nanoid) = 20),
  provider TEXT NOT NULL,
  provider_user_id TEXT NOT NULL,
  role user_role NOT NULL DEFAULT 'student',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_login_at TIMESTAMPTZ,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at TIMESTAMPTZ,
  UNIQUE (provider, provider_user_id)
);

CREATE INDEX idx_users_provider ON users(provider, provider_user_id) WHERE is_active;
CREATE INDEX idx_users_nanoid ON users(nanoid);

INSERT INTO users (id, nanoid, provider, provider_user_id, role)
VALUES ('00000000-0000-0000-0000-000000000001', 'devuser0000000000000', 'development', 'dev-user-001', 'student')
ON CONFLICT DO NOTHING;

CREATE TABLE subjects (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  nanoid TEXT NOT NULL UNIQUE DEFAULT nanoid20() CHECK (length(nanoid) = 20),
  owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  description TEXT,
  academic_year TEXT,
  semester TEXT,
  course_code TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_subjects_owner ON subjects(owner_user_id) WHERE is_active;
CREATE INDEX idx_subjects_nanoid ON subjects(nanoid);
CREATE INDEX idx_subjects_course_code ON subjects(course_code) WHERE is_active;

CREATE TABLE raw_files (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  nanoid TEXT NOT NULL UNIQUE DEFAULT nanoid20() CHECK (length(nanoid) = 20),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  original_filename TEXT NOT NULL,
  file_type file_type NOT NULL,
  file_size_bytes BIGINT NOT NULL,
  source_url TEXT,
  gcs_bucket TEXT NOT NULL,
  gcs_object_path TEXT NOT NULL,
  gcs_signed_url_expires_at TIMESTAMPTZ,
  status file_status NOT NULL DEFAULT 'uploading',
  total_pages INTEGER,
  mime_type TEXT,
  processed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at TIMESTAMPTZ,
  UNIQUE (gcs_bucket, gcs_object_path)
);

CREATE INDEX idx_raw_files_subject_user ON raw_files(subject_id, user_id) WHERE is_active;
CREATE INDEX idx_raw_files_nanoid ON raw_files(nanoid);
CREATE INDEX idx_raw_files_status ON raw_files(status) WHERE is_active;
CREATE INDEX idx_raw_files_processed ON raw_files(processed_at DESC) WHERE is_active;
CREATE INDEX idx_raw_files_source_url ON raw_files(source_url) WHERE source_url IS NOT NULL;

CREATE TABLE materials (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  raw_file_id UUID NOT NULL REFERENCES raw_files(id) ON DELETE CASCADE,
  sequence_in_file INTEGER NOT NULL,
  page_start INTEGER,
  page_end INTEGER,
  content_markdown TEXT NOT NULL,
  char_count INTEGER NOT NULL,
  embedding vector(1536) NOT NULL,
  embedding_model TEXT NOT NULL DEFAULT 'gemini-embedding-001',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_materials_file_seq ON materials(raw_file_id, sequence_in_file) WHERE is_active;
CREATE INDEX idx_materials_content_fts ON materials USING GIN (to_tsvector('english', content_markdown)) WHERE is_active;
CREATE INDEX idx_materials_embedding_vector ON materials USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64) WHERE is_active;

CREATE TABLE chats (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  nanoid TEXT NOT NULL UNIQUE DEFAULT nanoid20() CHECK (length(nanoid) = 20),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  parent_chat_id UUID REFERENCES chats(id) ON DELETE SET NULL,
  question TEXT NOT NULL,
  plan_json JSONB,
  termination_reason TEXT,
  final_answer_markdown TEXT,
  feedback chat_feedback,
  feedback_at TIMESTAMPTZ,
  actual_search_steps INTEGER NOT NULL DEFAULT 0,
  search_step_1_hit_material_ids UUID[],
  search_step_2_hit_material_ids UUID[],
  search_step_3_hit_material_ids UUID[],
  search_step_4_hit_material_ids UUID[],
  search_step_5_hit_material_ids UUID[],
  evidence_snippets JSONB,
  used_raw_file_ids UUID[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_chats_subject_user ON chats(subject_id, user_id) WHERE is_active;
CREATE INDEX idx_chats_nanoid ON chats(nanoid);
CREATE INDEX idx_chats_created ON chats(created_at DESC) WHERE is_active;
CREATE INDEX idx_chats_feedback ON chats(feedback) WHERE is_active AND feedback IS NOT NULL;
CREATE INDEX idx_chats_used_raw_files ON chats USING GIN (used_raw_file_ids) WHERE is_active;
CREATE INDEX idx_chats_evidence_snippets ON chats USING GIN (evidence_snippets);
CREATE INDEX idx_chats_termination ON chats(termination_reason) WHERE is_active AND termination_reason IS NOT NULL;
CREATE INDEX idx_chats_parent ON chats(parent_chat_id) WHERE is_active;

CREATE TABLE jobs (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  job_type job_type NOT NULL DEFAULT 'file_ingestion',
  target_raw_file_id UUID REFERENCES raw_files(id) ON DELETE CASCADE,
  idempotency_key TEXT NOT NULL UNIQUE,
  status job_status NOT NULL DEFAULT 'pending',
  gemini_model TEXT,
  gemini_phase gemini_phase,
  error_message TEXT,
  retry_count INTEGER NOT NULL DEFAULT 0,
  max_retries INTEGER NOT NULL DEFAULT 3,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ
);

CREATE INDEX idx_jobs_target_file ON jobs(target_raw_file_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_idempotency ON jobs(idempotency_key);
CREATE INDEX idx_jobs_created ON jobs(created_at DESC);
CREATE INDEX idx_jobs_type ON jobs(job_type);
