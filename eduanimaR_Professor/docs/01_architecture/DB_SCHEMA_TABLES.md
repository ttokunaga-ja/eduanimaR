# DB_SCHEMA_TABLES

## 目的
eduanima-professor（Professor）が管理するPostgreSQL 18.1 + pgvectorのテーブル定義（案）を記載する。

> 本ファイルは `DB_SCHEMA_DESIGN.md` の原則に基づいた具体的なテーブル定義のSSOT。

## 前提・参照ドキュメント
- [DB_SCHEMA_DESIGN.md](./DB_SCHEMA_DESIGN.md) - DB設計原則（SSOT）
- [MICROSERVICES_MAP.md](./MICROSERVICES_MAP.md) - サービス境界
- [SERVICE_SPEC_EDUANIMA_PROFESSOR.md](../../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md) - Professor仕様

---

## テーブル一覧

### コアエンティティ
1. **users** - ユーザー（OAuth/OIDC由来）
2. **subjects** - 科目/コース  
3. **materials** - 資料（文書/ファイル）メタデータ
4. **material_pages** - 資料のページ情報（OCR/抽出結果の単位）
5. **chunks** - 意味単位のテキスト断片（Markdown化済み）
6. **chunk_embeddings** - チャンクのベクトル表現（pgvector）

### セッション・推論
7. **reasoning_sessions** - 質問・検索セッション
8. **search_steps** - セッション内の検索ステップ（Librarian連携履歴）
9. **session_evidence** - セッションで選定された根拠（引用）

### 非同期処理
10. **ingest_jobs** - 資料取込ジョブ（Kafka連携）

---

## ENUM型定義

```sql
-- ユーザーロール
CREATE TYPE user_role AS ENUM (
  'student',
  'instructor',
  'admin'
);

-- 資料タイプ
CREATE TYPE material_type AS ENUM (
  'pdf',
  'powerpoint',
  'word',
  'image',
  'video',
  'url',
  'other'
);

-- 資料ステータス
CREATE TYPE material_status AS ENUM (
  'uploading',
  'pending_ingestion',
  'ingesting',
  'ready',
  'failed',
  'archived'
);

-- IngestJobステータス
CREATE TYPE ingest_job_status AS ENUM (
  'pending',
  'processing',
  'completed',
  'failed',
  'cancelled'
);

-- 検索モード（proto/librarian.protoと一致）
CREATE TYPE search_mode AS ENUM (
  'keyword',
  'vector',
  'hybrid'
);

-- 推論セッションステータス
CREATE TYPE session_status AS ENUM (
  'active',
  'completed',
  'failed',
  'cancelled'
);

-- Gemini使用フェーズ
CREATE TYPE gemini_phase AS ENUM (
  'ingestion',
  'planning',
  'search',
  'answer'
);
```

---

## 1. users

```sql
CREATE TABLE users (
  -- 主キー（内部UUID）
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  
  -- 外部公開ID（短縮ID）
  nanoid TEXT NOT NULL UNIQUE,
  
  -- OAuth/OIDC識別子（例: Google sub、学内IdP uid）
  provider TEXT NOT NULL,
  provider_user_id TEXT NOT NULL,
  
  -- ユーザー情報
  email TEXT NOT NULL,
  display_name TEXT,
  role user_role NOT NULL DEFAULT 'student',
  
  -- 監査
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_login_at TIMESTAMPTZ,
  
  -- ソフトデリート
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at TIMESTAMPTZ,
  
  UNIQUE (provider, provider_user_id)
);

CREATE INDEX idx_users_provider ON users(provider, provider_user_id) WHERE is_active;
CREATE INDEX idx_users_email ON users(email) WHERE is_active;
CREATE INDEX idx_users_nanoid ON users(nanoid);
```

**設計メモ**:
- `uuidv7()` で時系列順のUUID生成
- `nanoid` は外部公開ID（URL/ログ/問い合わせ）
- `provider / provider_user_id` で複数のIdP対応可能
- `is_active` + `deleted_at` でソフトデリート

---

## 2. subjects

```sql
CREATE TABLE subjects (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  nanoid TEXT NOT NULL UNIQUE,
  
  -- 所有者（インストラクター/作成者）
  owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  
  -- 科目情報
  title TEXT NOT NULL,
  description TEXT,
  academic_year TEXT,    -- 例: "2026"
  semester TEXT,         -- 例: "Spring"
  course_code TEXT,      -- 例: "CS101"
  
  -- 監査
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  
  -- ソフトデリート
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_subjects_owner ON subjects(owner_user_id) WHERE is_active;
CREATE INDEX idx_subjects_nanoid ON subjects(nanoid);
CREATE INDEX idx_subjects_course_code ON subjects(course_code) WHERE is_active;
```

**設計メモ**:
- **1科目 = 1つのスコープ境界**（物理制約の基準）
- `owner_user_id` でアクセス管理（将来的に共有機能追加時に `subject_users` テーブルを追加）

---

## 3. materials

```sql
CREATE TABLE materials (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  nanoid TEXT NOT NULL UNIQUE,
  
  -- 物理制約（MUST）
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  
  -- 資料メタデータ
  title TEXT NOT NULL,
  original_filename TEXT NOT NULL,
  material_type material_type NOT NULL,
  file_size_bytes BIGINT,
  
  -- GCS保存先（原本）
  gcs_bucket TEXT NOT NULL,
  gcs_object_path TEXT NOT NULL,
  
  -- 派生データの世代管理（LLM再生成対応）
  ingestion_version INTEGER NOT NULL DEFAULT 1,
  
  -- ステータス
  status material_status NOT NULL DEFAULT 'pending_ingestion',
  
  -- ページ数（PDF/PowerPoint等）
  total_pages INTEGER,
  
  -- 監査
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ingested_at TIMESTAMPTZ,
  
  -- ソフトデリート
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at TIMESTAMPTZ,
  
  UNIQUE (gcs_bucket, gcs_object_path)
);

-- 物理制約を強制するための複合インデックス（検索の主経路）
CREATE INDEX idx_materials_subject_user ON materials(subject_id, user_id) WHERE is_active;
CREATE INDEX idx_materials_nanoid ON materials(nanoid);
CREATE INDEX idx_materials_status ON materials(status) WHERE is_active;
CREATE INDEX idx_materials_ingested_at ON materials(ingested_at DESC) WHERE is_active;
```

**設計メモ**:
- **user_id + subject_id は必須**（物理制約の境界）
- `ingestion_version` で将来のモデル更新による再生成を管理
- `gcs_bucket/gcs_object_path` で原本参照（Professor独占）

---

## 4. material_pages

```sql
CREATE TABLE material_pages (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  
  -- 親資料
  material_id UUID NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
  
  -- ページ番号（1始まり）
  page_number INTEGER NOT NULL,
  
  -- OCR/Vision Reasoning結果（Markdown）
  content_markdown TEXT,
  
  -- メタデータ（構造化情報: JSON）
  metadata JSONB,
  
  -- 監査
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  
  UNIQUE (material_id, page_number)
);

CREATE INDEX idx_material_pages_material ON material_pages(material_id);
CREATE INDEX idx_material_pages_metadata ON material_pages USING GIN (metadata);
```

**設計メモ**:
- **ページ単位で管理**（PDF/PowerPoint等はページ構造を保持）
- `content_markdown`: Vision Reasoning/OCRの結果
- `metadata`: Structured Outputsで抽出した構造情報（章タイトル、図表番号等）をJSONBで保存

---

## 5. chunks

```sql
CREATE TABLE chunks (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  nanoid TEXT NOT NULL UNIQUE,
  
  -- 親資料（物理制約継承）
  material_id UUID NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
  
  -- ページ範囲（チャンクが跨る場合）
  page_start INTEGER NOT NULL,
  page_end INTEGER NOT NULL,
  
  -- チャンク内容（Markdown形式）
  content_markdown TEXT NOT NULL,
  
  -- チャンクのシーケンス（資料内の順序）
  sequence_in_material INTEGER NOT NULL,
  
  -- 世代管理（LLM更新で再生成）
  generation INTEGER NOT NULL DEFAULT 1,
  
  -- 文字数（検索スニペット長の判定用）
  char_count INTEGER NOT NULL,
  
  -- 監査
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  
  -- ソフトデリート
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at TIMESTAMPTZ
);

-- 資料単位でのチャンク取得（順序保証）
CREATE INDEX idx_chunks_material_seq ON chunks(material_id, sequence_in_material) WHERE is_active;
CREATE INDEX idx_chunks_nanoid ON chunks(nanoid);

-- 全文検索インデックス（PostgreSQL組み込み）
CREATE INDEX idx_chunks_content_fts ON chunks USING GIN (to_tsvector('english', content_markdown)) WHERE is_active;
```

**設計メモ**:
- **意味単位のチャンク**（Gemini 3 Flashで分割）
- `sequence_in_material` で資料内の順序を保持（前後文脈の取得に使用）
- `generation` で将来のモデル更新による再生成を管理
- **全文検索はPostgreSQL標準のGINインデックス**（Elasticsearchは将来拡張）

---

## 6. chunk_embeddings

```sql
CREATE TABLE chunk_embeddings (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  
  -- チャンク参照
  chunk_id UUID NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
  
  -- ベクトル（pgvector: 例: 768次元）
  embedding vector(768) NOT NULL,
  
  -- Embeddingモデル情報（世代管理）
  model_name TEXT NOT NULL,
  model_version TEXT NOT NULL,
  
  -- 監査
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  
  UNIQUE (chunk_id, model_name, model_version)
);

-- HNSW インデックス（高速近似検索）
-- m=16, ef_construction=64 は推奨初期値（要件に応じて調整）
CREATE INDEX idx_chunk_embeddings_vector ON chunk_embeddings 
  USING hnsw (embedding vector_cosine_ops) 
  WITH (m = 16, ef_construction = 64);

CREATE INDEX idx_chunk_embeddings_chunk ON chunk_embeddings(chunk_id);
```

**設計メモ**:
- **pgvector 0.8.1のHNSWインデックス**採用
- `model_name/model_version` でEmbeddingモデル更新に対応
- `vector_cosine_ops` でコサイン類似度検索
- パラメータ `m`, `ef_construction` は初期値（ベンチマーク後に調整）

---

## 7. reasoning_sessions

```sql
CREATE TABLE reasoning_sessions (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  nanoid TEXT NOT NULL UNIQUE,
  
  -- 物理制約（MUST）
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  
  -- ユーザー質問
  question TEXT NOT NULL,
  
  -- Phase 2（Plan）結果（Structured Outputs JSON）
  plan_json JSONB,
  
  -- Phase 4（Answer）結果
  final_answer_markdown TEXT,
  
  -- セッション状態
  status session_status NOT NULL DEFAULT 'active',
  
  -- Librarian連携情報
  max_search_steps INTEGER NOT NULL DEFAULT 5,
  actual_search_steps INTEGER NOT NULL DEFAULT 0,
  
  -- 監査
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  
  -- ソフトデリート
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at TIMESTAMPTZ
);

-- 物理制約を強制するための複合インデックス
CREATE INDEX idx_sessions_subject_user ON reasoning_sessions(subject_id, user_id) WHERE is_active;
CREATE INDEX idx_sessions_nanoid ON reasoning_sessions(nanoid);
CREATE INDEX idx_sessions_status ON reasoning_sessions(status) WHERE is_active;
CREATE INDEX idx_sessions_created ON reasoning_sessions(created_at DESC) WHERE is_active;
```

**設計メモ**:
- **質問・検索セッション**（Phase 2〜4の状態管理）
- `plan_json` にPhase 2の計画（調査項目、停止条件等）を保存
- `max_search_steps` でLibrarianループの最大回数を制御

---

## 8. search_steps

```sql
CREATE TABLE search_steps (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  
  -- 親セッション
  session_id UUID NOT NULL REFERENCES reasoning_sessions(id) ON DELETE CASCADE,
  
  -- ステップ情報
  step_number INTEGER NOT NULL,
  step_id TEXT NOT NULL,  -- proto/librarian.proto の SearchRequest.step_id
  
  -- 検索パラメータ
  search_mode search_mode NOT NULL,
  query_text TEXT NOT NULL,
  top_k INTEGER NOT NULL,
  keywords TEXT[],
  
  -- 検索結果（メタデータ）
  result_count INTEGER NOT NULL DEFAULT 0,
  
  -- 監査
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  
  UNIQUE (session_id, step_number)
);

CREATE INDEX idx_search_steps_session ON search_steps(session_id, step_number);
CREATE INDEX idx_search_steps_step_id ON search_steps(step_id);
```

**設計メモ**:
- **Librarianとの検索ループの履歴**を記録
- `step_id` はLibrarianプロトコルとの紐付け
- デバッグ・分析用途（どのクエリがヒットしたか追跡）

---

## 9. session_evidence

```sql
CREATE TABLE session_evidence (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  
  -- 親セッション
  session_id UUID NOT NULL REFERENCES reasoning_sessions(id) ON DELETE CASCADE,
  
  -- 引用元
  chunk_id UUID NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
  material_id UUID NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
  
  -- 引用箇所（ページ範囲）
  page_start INTEGER NOT NULL,
  page_end INTEGER NOT NULL,
  
  -- スニペット（最終回答で表示する部分）
  snippet_markdown TEXT NOT NULL,
  
  -- 選定理由（Librarian由来、任意）
  selection_reason TEXT,
  
  -- 検索スコア（参考値）
  search_score FLOAT,
  
  -- 監査
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_evidence_session ON session_evidence(session_id);
CREATE INDEX idx_session_evidence_chunk ON session_evidence(chunk_id);
CREATE INDEX idx_session_evidence_material ON session_evidence(material_id);
```

**設計メモ**:
- **Librarianが選定した根拠セット**を保存
- 最終回答（Phase 4）での引用箇所表示に使用
- 透明性・追跡可能性の確保

---

## 10. ingest_jobs

```sql
CREATE TABLE ingest_jobs (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  nanoid TEXT NOT NULL UNIQUE,
  
  -- 処理対象
  material_id UUID NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
  
  -- 冪等性キー（Kafka再送対応）
  idempotency_key TEXT NOT NULL UNIQUE,
  
  -- ジョブ状態
  status ingest_job_status NOT NULL DEFAULT 'pending',
  
  -- 使用モデル（環境変数 PROFESSOR_GEMINI_MODEL_INGESTION）
  gemini_model TEXT NOT NULL,
  gemini_phase gemini_phase NOT NULL DEFAULT 'ingestion',
  
  -- エラー情報
  error_message TEXT,
  retry_count INTEGER NOT NULL DEFAULT 0,
  max_retries INTEGER NOT NULL DEFAULT 3,
  
  -- 監査
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ
);

CREATE INDEX idx_ingest_jobs_material ON ingest_jobs(material_id);
CREATE INDEX idx_ingest_jobs_status ON ingest_jobs(status);
CREATE INDEX idx_ingest_jobs_idempotency ON ingest_jobs(idempotency_key);
CREATE INDEX idx_ingest_jobs_created ON ingest_jobs(created_at DESC);
```

**設計メモ**:
- **Kafka経由の非同期処理ジョブ**を管理
- `idempotency_key` でKafkaの再送を吸収（冪等性保証）
- `retry_count` でリトライ制御

---

## マイグレーション方針

### Atlas運用
- スキーマ定義は `schema.hcl` が唯一の正
- 手動 `ALTER TABLE` は禁止
- expand/contract パターンで破壊的変更を回避

### 初回セットアップ手順
1. ENUM型を定義
2. コアエンティティ（users, subjects）を作成
3. 資料・チャンク関連テーブルを作成
4. インデックス（B-tree, GIN, HNSW）を作成
5. セッション・非同期ジョブテーブルを作成

### 推奨拡張（MVP後）
- `subject_users`: 科目の共有・ロール管理
- `user_preferences`: ユーザー設定
- `audit_logs`: 詳細監査ログ
- `material_summaries`: 大量ファイル高速絞り込み用の要約（必要時のみ）

---

## パフォーマンス最適化指針

### 検索の主経路
```sql
-- keyword検索（全文検索 + 物理制約）
SELECT c.* FROM chunks c
JOIN materials m ON c.material_id = m.id
WHERE m.subject_id = :subject_id
  AND m.user_id = :user_id
  AND m.is_active = TRUE
  AND c.is_active = TRUE
  AND to_tsvector('english', c.content_markdown) @@ plainto_tsquery('english', :query)
ORDER BY ts_rank(to_tsvector('english', c.content_markdown), plainto_tsquery('english', :query)) DESC
LIMIT :top_k;

-- vector検索（ベクトル類似度 + 物理制約）
SELECT c.*, ce.embedding <=> :query_vector AS distance
FROM chunks c
JOIN chunk_embeddings ce ON c.id = ce.chunk_id
JOIN materials m ON c.material_id = m.id
WHERE m.subject_id = :subject_id
  AND m.user_id = :user_id
  AND m.is_active = TRUE
  AND c.is_active = TRUE
ORDER BY ce.embedding <=> :query_vector
LIMIT :top_k;
```

### ベンチマーク要件
- 10万チャンク規模でベクトル検索 < 100ms（p95）
- 全文検索 < 50ms（p95）
- 科目あたり1000ファイル想定

---

## チェックリスト

### 設計レビュー時の確認事項
- [ ] 全テーブルに `user_id` または `subject_id` による物理制約があるか
- [ ] 固定値は全てENUM型で定義されているか（VARCHAR禁止）
- [ ] 主キーは `uuidv7()` で時系列順に生成されるか
- [ ] 外部公開IDは `nanoid` でユニーク制約が付いているか
- [ ] ソフトデリートは `is_active` + `deleted_at` で統一されているか
- [ ] インデックスは検索要件に基づいて適切に設計されているか
- [ ] `created_at`, `updated_at` は全テーブルに存在するか

### 実装時の確認事項
- [ ] Atlas `schema.hcl` にENUM定義が反映されているか
- [ ] sqlcの設定で nullable型が統一されているか
- [ ] Testcontainersでのマイグレーションテストが通るか
- [ ] pgvectorのHNSWインデックスパラメータをベンチマークしたか

---

## 関連ドキュメント
- [DB_SCHEMA_DESIGN.md](./DB_SCHEMA_DESIGN.md) - 設計原則
- [CLEAN_ARCHITECTURE.md](./CLEAN_ARCHITECTURE.md) - レイヤー構造
- [STACK.md](../02_tech_stack/STACK.md) - 技術スタック詳細
- [SKILL_DB_ATLAS_SQLC_PGX.md](../skills/SKILL_DB_ATLAS_SQLC_PGX.md) - 実装ガイド
