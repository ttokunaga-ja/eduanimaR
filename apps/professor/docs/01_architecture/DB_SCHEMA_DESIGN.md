# DB_SCHEMA_DESIGN

## 目的
**PostgreSQL 18 + pgvector 0.8.1 + Atlas + sqlc** 前提で、スキーマ設計の意思決定（型/制約/インデックス）を統一する。

> **PostgreSQL バージョン根拠（PG18 採用理由）**
> - `uuidv7()` ネイティブ関数（時系列ソート可能 UUID、B-tree 効率が v4 より優れる）
> - `UNIQUE NULLS NOT DISTINCT`（NULL を区別しない一意制約、PG15+）
> - B-tree Skip Scan（部分インデックス効率向上）
> - pgvector 0.8.1 は PG18 対応済み（`pgvector/pgvector:pg18` Docker イメージ）
> - Cloud SQL for PostgreSQL 18 が利用可能

## データ所有（最重要）
- Cloud SQL（PostgreSQL）と GCS の直接アクセス権限は **Professor のみに付与**する
- Librarian は DB/GCS の資格情報を持たない（設計・運用の不変条件）

## ID戦略（UUID + NanoID）
- 内部主キー: UUID（推奨: UUIDv7 / `uuidv7()` を利用）
- 外部公開ID: NanoID（URL/ログ/問い合わせで扱いやすい短ID）
- ルール:
  - 参照整合性は内部UUIDで維持する
  - 外部公開IDはユニーク制約 + 露出するAPIのみで使用する

## ENUMの積極採用
- 固定値（status/type/category）は **PostgreSQL ENUM を必須** とする
  - **VARCHAR で固定値を管理する設計は禁止**（typo/バリデーション漏れ/性能劣化を誘発する）
- 利点: 型安全性、制約の明確化、アプリ側の分岐漏れ検知
- 変更方針: 追加は許容、削除/名前変更は慎重に（互換性を壊しやすい）

## NULLとデフォルト
- 原則: `NOT NULL` + `DEFAULT` を優先
- NULLが必要な場合:
  - sqlc/pgx が生成する nullable 型を統一して使う
  - APIのJSON表現（省略/明示null）も合わせて決める

## インデックス
- B-tree を基本とし、検索要件に応じて GIN / GiST / HNSW(pgvector) を選定
- 18.1の機能（例: B-tree Skip Scan 等）は「要件を満たす場合のみ」採用し、必ずベンチマークを残す

## ベクトル検索（pgvector 0.8.1）
- OLTPとベクトル検索を同居させる場合は、テーブル分離/更新頻度/インデックス再構築コストを考慮
- HNSW を使う場合:
  - 取り込みバッチ/再構築戦略（オフピーク）を定める
  - 近似検索の許容誤差（recall）を要件化する

## Atlas運用前提
- スキーマ変更は `schema.hcl` が唯一の正
- 手動 `ALTER TABLE` は禁止（差分が壊れる）

## マルチテナント/物理制約（MUST）
- 検索・参照の主経路は「user_id / subject_id による物理絞り込み」を前提にする
- 主要テーブルは原則として以下のカラムを持つこと
  - `user_id`
  - `subject_id`
  - `is_active` または `deleted_at`

## LLM派生データの世代管理（推奨）
- OCR/構造化/Embedding は将来のモデル更新で再生成される
- 「原本（GCS）」と「派生（DB）」を分け、派生は version/generation を持てる設計にする

---

## 関連ドキュメント

### スキーマ設計の詳細
- **[DB_SCHEMA_TABLES.md](./DB_SCHEMA_TABLES.md)** - 具体的なテーブル定義（10テーブル + ENUM）
- **[DB_SCHEMA_ERD.md](./DB_SCHEMA_ERD.md)** - ER図とデータフロー（Mermaid）
- **[DB_SCHEMA_DISCUSSION.md](./DB_SCHEMA_DISCUSSION.md)** - 議論ポイントと意思決定事項

### アーキテクチャ
- [CLEAN_ARCHITECTURE.md](./CLEAN_ARCHITECTURE.md) - レイヤー構造と依存方向
- [MICROSERVICES_MAP.md](./MICROSERVICES_MAP.md) - サービス境界とデータ所有

### 実装
- [STACK.md](../02_tech_stack/STACK.md) - 技術スタック（PostgreSQL 18.1, Atlas, sqlc, pgx, pgvector）
- [SKILL_DB_ATLAS_SQLC_PGX.md](../skills/SKILL_DB_ATLAS_SQLC_PGX.md) - 実装ガイド

---

## Phase 1 テーブル定義

Last-updated: 2026-02-18
Status: Published
Owner: @ttokunaga-ja

### users（ユーザー）

```sql
CREATE TYPE auth_provider AS ENUM ('google', 'meta', 'microsoft', 'line');

CREATE TABLE users (
  user_id          UUID         PRIMARY KEY DEFAULT uuidv7(),
  email            TEXT         UNIQUE NOT NULL,
  -- SSO カラム: Phase 1 では NULL、Phase 2 で実際に使用
  provider         auth_provider            NULL,
  provider_user_id TEXT                     NULL,
  created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  CONSTRAINT users_provider_unique UNIQUE (provider, provider_user_id)
);

-- Phase 1固定ユーザーの初期データ
INSERT INTO users (user_id, email) VALUES
  ('00000000-0000-0000-0000-000000000001', 'dev@example.com');
```

**設計意図**:
- `provider`, `provider_user_id` は Phase 1 から定義（NULLABLE）。Phase 2（SSO）で実際に使用する。
- Phase 2 への移行時に ALTER TABLE 不要。固定 dev-user を DELETE するのみ。
- `email` は UNIQUE 制約で重複を防止

### subjects（科目）

```sql
CREATE TABLE subjects (
  subject_id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  lms_course_id TEXT, -- Moodle course ID（将来の自動紐付け用）
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subjects_user_id ON subjects(user_id);
```

**設計意図**:
- 1ユーザーが複数の科目を管理可能
- `lms_course_id` は将来のLMS連携用（Phase 1では未使用）
- user_id による物理絞り込みを想定したインデックス

### files（アップロードファイル）

```sql
CREATE TYPE file_status AS ENUM ('pending', 'processing', 'ready', 'failed');

CREATE TABLE files (
  file_id       UUID        PRIMARY KEY DEFAULT uuidv7(),
  subject_id    UUID        NOT NULL REFERENCES subjects(subject_id) ON DELETE CASCADE,
  name          TEXT        NOT NULL,
  gcs_path      TEXT        NOT NULL,  -- GCS上のパス: gs://bucket/user_id/subject_id/file_id.pdf
  mime_type     TEXT        NOT NULL,
  size_bytes    BIGINT      NOT NULL,
  status        file_status NOT NULL DEFAULT 'pending',
  error_message TEXT,                  -- status='failed'時のエラー詳細
  uploaded_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  processed_at  TIMESTAMPTZ
);

CREATE INDEX idx_files_subject_id ON files(subject_id);
CREATE INDEX idx_files_status     ON files(status);
```

**設計意図**:
- `status` は ENUM 型（typo・バリデーション漏れを防止）
- `gcs_path` は原本の所在を示すSSOT
- `processed_at` は処理完了時刻（NULL = 未完了）

### chunks（ベクトル検索用チャンク）

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE chunks (
  chunk_id UUID PRIMARY KEY DEFAULT uuidv7(),
  file_id UUID NOT NULL REFERENCES files(file_id) ON DELETE CASCADE,
  page_number INT, -- PDFページ番号（画像の場合はNULL）
  chunk_index INT NOT NULL, -- ファイル内での連番
  content TEXT NOT NULL, -- OCR/抽出されたテキスト
  embedding vector(768) NOT NULL, -- Gemini Embedding（次元数は要確認）
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chunks_file_id ON chunks(file_id);
CREATE INDEX idx_chunks_embedding ON chunks USING hnsw (embedding vector_cosine_ops);
```

**設計意図**:
- pgvector 0.8.1 の HNSW インデックスを使用
- `embedding` の次元数（768）は Gemini Embedding の仕様に依存
- `chunk_index` でファイル内の順序を保持

### ingest_jobs（非同期処理ジョブ）

```sql
CREATE TYPE job_status AS ENUM ('pending', 'processing', 'completed', 'failed');

CREATE TABLE ingest_jobs (
  job_id        UUID       PRIMARY KEY DEFAULT uuidv7(),
  file_id       UUID       NOT NULL REFERENCES files(file_id) ON DELETE CASCADE,
  status        job_status NOT NULL DEFAULT 'pending',
  retry_count   INT        NOT NULL DEFAULT 0,
  max_retries   INT        NOT NULL DEFAULT 3,
  error_message TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  started_at    TIMESTAMPTZ,
  completed_at  TIMESTAMPTZ
);

CREATE INDEX idx_ingest_jobs_status  ON ingest_jobs(status);
CREATE INDEX idx_ingest_jobs_file_id ON ingest_jobs(file_id);
```

**設計意図**:
- Kafka非同期パイプライン（OCR/Embedding）のジョブ管理
- `status` は ENUM 型（typo・バリデーション漏れを防止）
- `retry_count` と `max_retries` でリトライ制御

### qa_sessions（質問応答セッション）

```sql
CREATE TABLE qa_sessions (
  session_id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  subject_id UUID NOT NULL REFERENCES subjects(subject_id) ON DELETE CASCADE,
  question TEXT NOT NULL,
  answer TEXT, -- 最終回答（SSE完了後に保存）
  sources JSONB, -- 参照元: [{ file_id, chunk_id, page_number, excerpt }]
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  answered_at TIMESTAMPTZ
);

CREATE INDEX idx_qa_sessions_user_id ON qa_sessions(user_id);
CREATE INDEX idx_qa_sessions_subject_id ON qa_sessions(subject_id);
```

**設計意図**:
- 質問応答の履歴を保存（分析・改善用）
- `sources` は JSONB で柔軟に参照元を記録
- `answered_at` が NULL の場合は未回答（タイムアウト/エラー）

## マイグレーション方針

- **ツール**: Atlas（`atlas migrate diff`, `atlas migrate apply`）
- **Phase 1 初期セットアップ**:
  1. `uuidv7()` は **PostgreSQL 18 のネイティブ組み込み関数**（拡張インストール不要）
  2. `vector` 拡張をインストール（pgvector 0.8.1）: `CREATE EXTENSION IF NOT EXISTS vector;`
  3. 上記6テーブルを作成
  4. 固定ユーザーを INSERT
  5. **Atlas ハッシュ更新**: migration ファイル変更後は必ず `atlas migrate hash` を実行すること

> **注意**: `uuidv7()` は PostgreSQL 18 の組み込み関数であり、拡張インストール不要。旧来の外部拡張とは異なり、`CREATE EXTENSION` なしで即座に使用できる。

## Phase 1→Phase 2 移行時のDB変更

### マイグレーション方針

1. Phase 1の固定dev-user（`dev@example.com`）を削除
2. 既存の `subjects`, `files`, `qa_sessions` は保持（user_idの再紐付けは不要）
3. `users.provider`, `users.provider_user_id` は Phase 1 から定義済み（ALTER TABLE 不要）

```sql
-- Phase 1固定ユーザーの削除（Phase 2移行時）
DELETE FROM users WHERE user_id = '00000000-0000-0000-0000-000000000001';
```

### Phase 2以降のSSO認証フロー

1. ユーザーがSSOプロバイダーで認証
2. `provider` と `provider_user_id` でユーザーを検索
3. 存在しなければ新規ユーザーを作成（拡張機能のみ）
4. 存在すればセッション発行（Web版・拡張機能両方）

**注**: Web版からの新規登録は全Phase禁止。拡張機能でのみユーザー登録可能。

---

- **Phase 2→Phase 3 移行時の変更点**:
  - 変更なし（Chrome Web Store公開のみ、バックエンド変更なし）

- **Phase 3→Phase 4 移行時の変更点**:
  - 画面解説機能用のテーブル追加（未確定）
    ```sql
    CREATE TABLE screen_analyses (
      analysis_id UUID PRIMARY KEY DEFAULT uuidv7(),
      user_id UUID NOT NULL REFERENCES users(user_id),
      subject_id UUID NOT NULL REFERENCES subjects(subject_id),
      screen_html TEXT NOT NULL,
      screen_images JSONB, -- [{ image_id, url, description }]
      analysis TEXT, -- LLMによる解析結果
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
    ```
  - **重要**: 画面データは短期保存のみ（プライバシー配慮）
    - `created_at` から7日後に自動削除するトリガーまたはバッチ処理を設定

- **Phase 4→Phase 5 移行時の変更点**（構想段階）:
  - 学習計画機能用のテーブル追加（未確定）
  - 小テスト結果の保存テーブル追加（未確定）
  - プライバシー配慮のための匿名化処理を実装
