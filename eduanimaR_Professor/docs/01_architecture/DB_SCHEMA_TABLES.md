# DB_SCHEMA_TABLES

## 目的
最終承認されたDB設計を反映する。eduanima-professor（Professor）が管理するPostgreSQL 18.1 + pgvectorのテーブル定義。

> 本ファイルは `DB_SCHEMA_DESIGN.md` の原則に基づいた具体的なテーブル定義のSSOT。

## 前提・参照ドキュメント
- [DB_SCHEMA_DESIGN.md](./DB_SCHEMA_DESIGN.md) - DB設計原則（SSOT）
- [MICROSERVICES_MAP.md](./MICROSERVICES_MAP.md) - サービス境界
- [SERVICE_SPEC_EDUANIMA_PROFESSOR.md](../../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md) - Professor仕様

---

## テーブル一覧

### コアエンティティ
1. **users** - ユーザー（OAuth/OIDC由来、個人情報非収集）【NanoID: 20文字】
2. **subjects** - 科目/コース【NanoID: 20文字】
3. **raw_files** - 原本ファイル（GCS保存、自動取り込み対応）【NanoID: 20文字】
4. **materials** - 資料チャンク（Markdown化済み、ベクトル埋め込み済み）【NanoID: なし】
5. **chats** - 質問・検索セッション（Phase 2〜4の統合）【NanoID: 20文字】

### 非同期処理
6. **jobs** - 非同期処理ジョブ（ファイル取込、バッチ処理等）【NanoID: なし】

---

## ENUM型定義

```sql
-- ユーザーロール
CREATE TYPE user_role AS ENUM (
  'student',
  'instructor',
  'admin'
);

-- 原本ファイルタイプ（Gemini APIがサポートする形式）
CREATE TYPE file_type AS ENUM (
  -- ドキュメント（ネイティブ対応）
  'pdf',
  'text',
  
  -- スクリプト・コード（ネイティブ対応）
  'python',
  'go',
  'javascript',
  'html',
  'css',
  'json',
  'markdown',
  'csv',
  
  -- 画像（ネイティブ対応）
  'png',
  'jpeg',
  'webp',
  'heic',
  'heif',
  
  -- MS Office（Drive API経由で変換が必要）
  'docx',
  'xlsx',
  'pptx',
  
  -- Google Workspace（Drive API経由で変換が必要）
  'google_docs',
  'google_sheets',
  'google_slides',
  
  -- その他
  'other'
);

-- 原本ファイルステータス
CREATE TYPE file_status AS ENUM (
  'uploading',      -- アップロード中
  'uploaded',       -- アップロード完了
  'processing',     -- 処理中（Vision Reasoning実行中）
  'completed',      -- 処理完了
  'failed',         -- 処理失敗
  'archived'        -- アーカイブ済み
);

-- Jobタイプ
CREATE TYPE job_type AS ENUM (
  'file_ingestion',      -- ファイル取込（GCS → Vision Reasoning → チャンク化）
  'ai_batch_processing', -- AIバッチ処理（再Embedding等）
  'search_optimization', -- 検索最適化（インデックス再構築等）
  'maintenance'          -- その他メンテナンス
);

-- Jobステータス
CREATE TYPE job_status AS ENUM (
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

-- ユーザーフィードバック
CREATE TYPE chat_feedback AS ENUM (
  'good',
  'bad'
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
  
  -- 外部公開ID（短縮ID、20文字）
  nanoid TEXT NOT NULL UNIQUE CHECK (length(nanoid) = 20),
  
  -- OAuth/OIDC識別子（例: Google sub、学内IdP uid）
  provider TEXT NOT NULL,
  provider_user_id TEXT NOT NULL,
  
  -- ユーザーロール
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
CREATE INDEX idx_users_nanoid ON users(nanoid);
```

**設計メモ**:
- `uuidv7()` で時系列順のUUID生成
- `nanoid` は外部公開ID（URL/ログ/問い合わせ）
- `provider / provider_user_id` で複数のIdP対応可能
- **個人情報非収集**: `email`, `display_name` を削除
- `is_active` + `deleted_at` でソフトデリート

---

## 2. subjects

```sql
CREATE TABLE subjects (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  nanoid TEXT NOT NULL UNIQUE CHECK (length(nanoid) = 20),
  
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

## 3. raw_files

```sql
CREATE TABLE raw_files (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  nanoid TEXT NOT NULL UNIQUE CHECK (length(nanoid) = 20),
  
  -- 物理制約（MUST）
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  
  -- ファイル情報
  original_filename TEXT NOT NULL,
  file_type file_type NOT NULL,
  file_size_bytes BIGINT NOT NULL,
  
  -- 自動取り込み用のソースURL（将来機能）
  source_url TEXT,
  
  -- GCS保存先（原本）
  gcs_bucket TEXT NOT NULL,
  gcs_object_path TEXT NOT NULL,
  gcs_signed_url_expires_at TIMESTAMPTZ,
  
  -- ステータス
  status file_status NOT NULL DEFAULT 'uploading',
  
  -- メタデータ（ドキュメント・画像・コードのみサポート）
  total_pages INTEGER,           -- PDF/PowerPointのページ数
  mime_type TEXT,
  
  -- 処理情報
  processed_at TIMESTAMPTZ,
  
  -- 監査
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  
  -- ソフトデリート
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at TIMESTAMPTZ,
  
  UNIQUE (gcs_bucket, gcs_object_path)
);

-- 物理制約を強制するための複合インデックス
CREATE INDEX idx_raw_files_subject_user ON raw_files(subject_id, user_id) WHERE is_active;
CREATE INDEX idx_raw_files_nanoid ON raw_files(nanoid);
CREATE INDEX idx_raw_files_status ON raw_files(status) WHERE is_active;
CREATE INDEX idx_raw_files_processed ON raw_files(processed_at DESC) WHERE is_active;
CREATE INDEX idx_raw_files_source_url ON raw_files(source_url) WHERE source_url IS NOT NULL;
```

**設計メモ**:
- **user_id + subject_id は必須**（物理制約の境界）
- `title` を削除（ファイル名から自動生成またはユーザー指定）
- `source_url` を追加（将来の自動取り込み機能用）
- `duration_seconds` を削除（動画は非サポート）
- `ingestion_version` を削除（materials側で管理）
- `gcs_bucket/gcs_object_path` で原本参照（Professor独占）

---

## 4. materials

```sql
CREATE TABLE materials (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  
  -- 親ファイル（物理制約継承）
  raw_file_id UUID NOT NULL REFERENCES raw_files(id) ON DELETE CASCADE,
  
  -- チャンク情報
  sequence_in_file INTEGER NOT NULL,  -- ファイル内の順序
  page_start INTEGER,                 -- 元ページ範囲（開始）
  page_end INTEGER,                   -- 元ページ範囲（終了）
  
  -- チャンク内容（Markdown形式）
  content_markdown TEXT NOT NULL,
  char_count INTEGER NOT NULL,
  
  -- ベクトル表現（pgvector: 768次元）
  embedding vector(768) NOT NULL,
  
  -- Embeddingモデル情報（固定: text-embedding-004）
  embedding_model TEXT NOT NULL DEFAULT 'text-embedding-004',
  
  -- 監査
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  
  -- ソフトデリート
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at TIMESTAMPTZ
);

-- ファイル単位でのチャンク取得（順序保証）
CREATE INDEX idx_materials_file_seq ON materials(raw_file_id, sequence_in_file) WHERE is_active;

-- 全文検索インデックス（PostgreSQL組み込み）
CREATE INDEX idx_materials_content_fts ON materials USING GIN (to_tsvector('english', content_markdown)) WHERE is_active;

-- HNSW ベクトル検索インデックス（高速近似検索）
CREATE INDEX idx_materials_embedding_vector ON materials 
  USING hnsw (embedding vector_cosine_ops) 
  WITH (m = 16, ef_construction = 64)
  WHERE is_active;
```

**設計メモ**:
- **意味単位のチャンク**（高速推論モデルで分割）
- `sequence_in_file` でファイル内の順序を保持（前後文脈の取得に使用）
- `embedding_version` を削除（モデル固定のため）
- `generation` を削除（シンプル化）
- `embedding_model` をデフォルト値付きに変更（text-embedding-004固定）
- **全文検索はPostgreSQL標準のGINインデックス**
- **pgvector 0.8.1のHNSWインデックス**採用

---

## 5. chats

```sql
CREATE TABLE chats (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  nanoid TEXT NOT NULL UNIQUE CHECK (length(nanoid) = 20),
  
  -- 物理制約（MUST）
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  
  -- 会話の親子関係
  parent_chat_id UUID REFERENCES chats(id) ON DELETE SET NULL,
  
  -- 質問内容
  question TEXT NOT NULL,
  
  -- Phase 2（Plan）結果（Structured Outputs JSON）
  -- 構造: {investigation_items, termination_conditions, search_strategy}
  plan_json JSONB,
  
  -- 検索終了理由（実際の終了理由を記録）
  termination_reason TEXT,
  
  -- Phase 4（Answer）結果
  final_answer_markdown TEXT,
  
  -- ユーザーフィードバック
  feedback chat_feedback,
  feedback_at TIMESTAMPTZ,
  
  -- Librarian連携情報
  actual_search_steps INTEGER NOT NULL DEFAULT 0,
  
  -- 検索詳細（各ステップでヒットしたmaterial_id配列）
  search_step_1_hit_material_ids UUID[],
  search_step_2_hit_material_ids UUID[],
  search_step_3_hit_material_ids UUID[],
  search_step_4_hit_material_ids UUID[],
  search_step_5_hit_material_ids UUID[],
  
  -- Pythonサーバから返された根拠（チャンク単位の詳細記録）
  -- 構造: [{material_id, snippet, task_id, relevance_score, page_start, page_end, search_step, selected_for_answer}]
  evidence_snippets JSONB,
  
  -- 最終回答に使用したファイル一覧（ファイル単位の重複排除リスト）
  used_raw_file_ids UUID[] NOT NULL DEFAULT '{}',
  
  -- 監査
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  
  -- ソフトデリート
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at TIMESTAMPTZ
);

-- 物理制約を強制するための複合インデックス
CREATE INDEX idx_chats_subject_user ON chats(subject_id, user_id) WHERE is_active;
CREATE INDEX idx_chats_nanoid ON chats(nanoid);
CREATE INDEX idx_chats_created ON chats(created_at DESC) WHERE is_active;
CREATE INDEX idx_chats_feedback ON chats(feedback) WHERE is_active AND feedback IS NOT NULL;

-- 使用ファイルの検索（GINインデックス）
CREATE INDEX idx_chats_used_raw_files ON chats USING GIN (used_raw_file_ids) WHERE is_active;

-- 根拠スニペットの検索（GINインデックス）
CREATE INDEX idx_chats_evidence_snippets ON chats USING GIN (evidence_snippets);

-- 終了理由での分析用インデックス
CREATE INDEX idx_chats_termination ON chats(termination_reason) WHERE is_active AND termination_reason IS NOT NULL;

-- 会話ツリーの取得
CREATE INDEX idx_chats_parent ON chats(parent_chat_id) WHERE is_active;
```

**設計メモ**:
- **質問・検索セッション**(Phase 2〜4の統合テーブル)
- `parent_chat_id`で会話の親子関係を管理:
  - NULL: 独立した新規質問
  - 値あり: フォローアップ質問、または選択肢選択後の質問
- 選択肢提示チャットの特徴:
  - `plan_json`: NULL (Phase 2未実行)
  - `actual_search_steps`: 0 (Librarianループなし)
  - `used_raw_file_ids`: [] (資料未使用)
  - `evidence_snippets`: NULL
  - `final_answer_markdown`: LLMが生成した選択肢テキスト
- `plan_json`にPhase 2の計画(調査項目、終了条件、検索戦略)を保存

---

## 6. jobs

```sql
CREATE TABLE jobs (
  -- 主キー
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  
  -- ジョブタイプ
  job_type job_type NOT NULL,
  
  -- 処理対象（任意）
  target_raw_file_id UUID REFERENCES raw_files(id) ON DELETE CASCADE,
  
  -- 冪等性キー（Kafka再送対応）
  idempotency_key TEXT NOT NULL UNIQUE,
  
  -- ジョブ状態
  status job_status NOT NULL DEFAULT 'pending',
  
  -- 使用モデル（環境変数から取得）
  gemini_model TEXT,
  gemini_phase gemini_phase,
  
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

CREATE INDEX idx_jobs_target_file ON jobs(target_raw_file_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_idempotency ON jobs(idempotency_key);
CREATE INDEX idx_jobs_created ON jobs(created_at DESC);
CREATE INDEX idx_jobs_type ON jobs(job_type);
```

**設計メモ**:
- **非同期処理ジョブ**を管理（ファイル取込、バッチ処理等）
- `idempotency_key` でKafkaの再送を吸収（冪等性保証）
- `retry_count` でリトライ制御
- `job_type` で様々なジョブタイプに対応

---

## ID戦略の詳細

### NanoID の文字数と適用基準
- **20文字**: 外部公開・URL共有・長期保存が必要なエンティティ
  - users, subjects, raw_files, chats
  - 衝突確率: ~10億年に1回（1時間に1000ID生成想定）
- **NanoID不要**: 外部露出がない内部処理エンティティ
  - materials（UUIDで十分、Librarianとの連携はUUIDで実施）
  - jobs（idempotency_keyで代用）

### 設計根拠
- **materials**: チャンク単位での外部参照は稀（通常はraw_fileレベルで引用表示）
- **jobs**: 内部処理のみ、Kafka再送は `idempotency_key` で管理
- **20文字の選定理由**: NanoIDのデフォルト21文字より1文字短縮し、衝突確率を維持しつつコンパクト化

---

## Go実装例: NanoID生成

```go
import (
    "fmt"
    gonanoid "github.com/matoous/go-nanoid/v2"
)

// 20文字のNanoIDを生成（users, subjects, raw_files, chats用）
func generateNanoID20() (string, error) {
    alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"
    return gonanoid.Generate(alphabet, 20)
}

// 使用例
func createUser(ctx context.Context, db *sql.DB, provider, providerUserID string) error {
    userID := uuid.Must(uuid.NewV7())
    nanoID, err := generateNanoID20()
    if err != nil {
        return fmt.Errorf("failed to generate nanoid: %w", err)
    }
    
    _, err = db.ExecContext(ctx, `
        INSERT INTO users (id, nanoid, provider, provider_user_id, role)
        VALUES ($1, $2, $3, $4, 'student')
    `, userID, nanoID, provider, providerUserID)
    
    return err
}
```

---

## evidence_snippets のJSONB構造

`chats.evidence_snippets` カラムに保存される構造：

```json
[
  {
    "material_id": "uuid-of-materials-table",
    "snippet": "決定係数はR^2 = 1 - (RSS/TSS)で定義される。",
    "task_id": "T1",
    "relevance_score": 0.95,
    "page_start": 15,
    "page_end": 15,
    "search_step": 2,
    "selected_for_answer": true
  }
]
```

**フィールド説明**:
- `material_id`: materialsテーブルのUUID
- `snippet`: 抽出されたテキストスニペット
- `task_id`: Phase 2で定義された調査タスクID
- `relevance_score`: Librarianが計算した関連度スコア（0.0〜1.0）
- `page_start`, `page_end`: 元ドキュメントのページ範囲
- `search_step`: どの検索ステップで取得されたか（1〜5）
- `selected_for_answer`: 最終回答生成に使用されたか

---

## Go型定義

### Pythonサーバからの返却データ

```go
// Pythonサーバ（Librarian）からの返却データ
type LibrarianEvidence struct {
    ChunkID        string  `json:"chunk_id"`         // materials.id
    TaskID         string  `json:"task_id"`          // Phase 2のタスクID
    Snippet        string  `json:"snippet"`          // 抽出されたスニペット
    RelevanceScore float64 `json:"relevance_score"`  // 関連度スコア
}
```

### DBに保存するevidence_snippets

```go
// DBに保存するevidence_snippets
type EvidenceSnippet struct {
    MaterialID       uuid.UUID `json:"material_id"`        // materials.id
    Snippet          string    `json:"snippet"`            // 抽出されたスニペット
    TaskID           string    `json:"task_id"`            // Phase 2のタスクID
    RelevanceScore   float64   `json:"relevance_score"`    // 関連度スコア
    PageStart        *int      `json:"page_start"`         // ページ範囲（開始）
    PageEnd          *int      `json:"page_end"`           // ページ範囲（終了）
    SearchStep       int       `json:"search_step"`        // 検索ステップ番号
    SelectedForAnswer bool     `json:"selected_for_answer"` // 最終回答に使用したか
}
```

### Phase 2の計画全体

```go
// Phase 2の計画全体
type SearchPlan struct {
    InvestigationItems      []InvestigationItem      `json:"investigation_items"`
    TerminationConditions   TerminationConditions    `json:"termination_conditions"`
    SearchStrategy          SearchStrategy           `json:"search_strategy"`
}

// 終了条件
type TerminationConditions struct {
    MaxSearchSteps       int      `json:"max_search_steps"`
    MinEvidenceCount     int      `json:"min_evidence_count"`
    ConfidenceThreshold  float64  `json:"confidence_threshold"`
    StopReasons          []string `json:"stop_reasons"`
}
```

---

## Go実装例

### 1. enrichEvidenceWithPageInfo()

Pythonサーバのレスポンスにページ情報を付加する処理：

```go
func enrichEvidenceWithPageInfo(
    ctx context.Context,
    db *sql.DB,
    librarianEvidence []LibrarianEvidence,
    searchStep int,
) ([]EvidenceSnippet, error) {
    enriched := make([]EvidenceSnippet, 0, len(librarianEvidence))
    
    for _, ev := range librarianEvidence {
        materialID, err := uuid.Parse(ev.ChunkID)
        if err != nil {
            return nil, fmt.Errorf("invalid chunk_id: %w", err)
        }
        
        // materialsテーブルからページ情報を取得
        var pageStart, pageEnd *int
        err = db.QueryRowContext(ctx,
            "SELECT page_start, page_end FROM materials WHERE id = $1",
            materialID,
        ).Scan(&pageStart, &pageEnd)
        if err != nil {
            return nil, fmt.Errorf("failed to get page info: %w", err)
        }
        
        enriched = append(enriched, EvidenceSnippet{
            MaterialID:       materialID,
            Snippet:          ev.Snippet,
            TaskID:           ev.TaskID,
            RelevanceScore:   ev.RelevanceScore,
            PageStart:        pageStart,
            PageEnd:          pageEnd,
            SearchStep:       searchStep,
            SelectedForAnswer: false, // 後で更新
        })
    }
    
    return enriched, nil
}
```

### 2. extractUsedFileIDs()

material_id → raw_file_id の重複排除：

```go
func extractUsedFileIDs(
    ctx context.Context,
    db *sql.DB,
    evidenceSnippets []EvidenceSnippet,
) ([]uuid.UUID, error) {
    if len(evidenceSnippets) == 0 {
        return []uuid.UUID{}, nil
    }
    
    // material_idを抽出
    materialIDs := make([]uuid.UUID, 0, len(evidenceSnippets))
    seen := make(map[uuid.UUID]bool)
    for _, ev := range evidenceSnippets {
        if !seen[ev.MaterialID] {
            materialIDs = append(materialIDs, ev.MaterialID)
            seen[ev.MaterialID] = true
        }
    }
    
    // raw_file_idを取得（重複排除）
    query := `
        SELECT DISTINCT raw_file_id 
        FROM materials 
        WHERE id = ANY($1) AND is_active = TRUE
    `
    rows, err := db.QueryContext(ctx, query, pq.Array(materialIDs))
    if err != nil {
        return nil, fmt.Errorf("failed to extract file ids: %w", err)
    }
    defer rows.Close()
    
    fileIDs := make([]uuid.UUID, 0)
    for rows.Next() {
        var fileID uuid.UUID
        if err := rows.Scan(&fileID); err != nil {
            return nil, err
        }
        fileIDs = append(fileIDs, fileID)
    }
    
    return fileIDs, rows.Err()
}
```

### 3. markEvidenceUsedInAnswer()

selected_for_answerフラグの更新：

```go
func markEvidenceUsedInAnswer(
    evidenceSnippets []EvidenceSnippet,
    usedMaterialIDs []uuid.UUID,
) []EvidenceSnippet {
    usedSet := make(map[uuid.UUID]bool)
    for _, id := range usedMaterialIDs {
        usedSet[id] = true
    }
    
    for i := range evidenceSnippets {
        if usedSet[evidenceSnippets[i].MaterialID] {
            evidenceSnippets[i].SelectedForAnswer = true
        }
    }
    
    return evidenceSnippets
}
```

### 4. saveChatResult()

DB保存処理：

```go
func saveChatResult(
    ctx context.Context,
    db *sql.DB,
    chatID uuid.UUID,
    finalAnswer string,
    evidenceSnippets []EvidenceSnippet,
    usedFileIDs []uuid.UUID,
    terminationReason string,
) error {
    evidenceJSON, err := json.Marshal(evidenceSnippets)
    if err != nil {
        return fmt.Errorf("failed to marshal evidence: %w", err)
    }
    
    query := `
        UPDATE chats
        SET final_answer_markdown = $1,
            evidence_snippets = $2,
            used_raw_file_ids = $3,
            termination_reason = $4,
            completed_at = NOW(),
            updated_at = NOW()
        WHERE id = $5
    `
    
    _, err = db.ExecContext(ctx, query,
        finalAnswer,
        evidenceJSON,
        pq.Array(usedFileIDs),
        terminationReason,
        chatID,
    )
    
    return err
}
```

### 5. generateCitationMarkdown()

ユーザー向け引用表示生成：

```go
func generateCitationMarkdown(
    ctx context.Context,
    db *sql.DB,
    usedFileIDs []uuid.UUID,
) (string, error) {
    if len(usedFileIDs) == 0 {
        return "", nil
    }
    
    query := `
        SELECT rf.original_filename, rf.nanoid
        FROM raw_files rf
        WHERE rf.id = ANY($1) AND rf.is_active = TRUE
        ORDER BY rf.created_at
    `
    
    rows, err := db.QueryContext(ctx, query, pq.Array(usedFileIDs))
    if err != nil {
        return "", err
    }
    defer rows.Close()
    
    var citations strings.Builder
    citations.WriteString("\n\n---\n\n**参考資料:**\n\n")
    
    index := 1
    for rows.Next() {
        var filename, nanoid string
        if err := rows.Scan(&filename, &nanoid); err != nil {
            return "", err
        }
        citations.WriteString(fmt.Sprintf("%d. %s (ID: %s)\n", index, filename, nanoid))
        index++
    }
    
    return citations.String(), rows.Err()
}
```

---

## 設計根拠

### なぜevidence_snippets（チャンク単位）とused_raw_file_ids（ファイル単位）の両方が必要か

1. **evidence_snippets（チャンク単位）**:
   - **用途**: デバッグ、分析、透明性確保
   - Librarianがどのチャンクを検索し、どの程度関連度が高かったかを記録
   - 各ステップでの検索品質を後から評価可能
   - ユーザーに詳細な根拠を示す際に使用

2. **used_raw_file_ids（ファイル単位）**:
   - **用途**: ユーザー向け引用表示、ファイルアクセス制御
   - 「この回答はどのファイルを参照したか」をファイル単位で明示
   - 重複排除されたファイルリストで効率的な表示が可能
   - 将来的なアクセス制御（ユーザーがファイルを削除した場合の対応等）

### file_type ENUMの設計根拠

Gemini API（File API）の仕様に基づく：

1. **ネイティブ対応形式**:
   - PDF, テキスト, 画像（PNG, JPEG等）, スクリプト（Python, Go等）
   - これらはGemini APIが直接処理可能

2. **変換が必要な形式**:
   - MS Office（docx, xlsx, pptx）→ Drive API経由でPDFに変換
   - Google Workspace → Drive API経由で変換

3. **将来拡張**:
   - ENUMに新しいタイプを追加することで、新しいファイル形式に対応可能
   - `other` タイプで未知の形式を一時的に受け入れ

### JSONB型を採用する理由

1. **柔軟性**:
   - `plan_json`, `evidence_snippets` は構造が進化する可能性が高い
   - 新しいフィールド追加時にスキーマ変更不要

2. **検索性能**:
   - GINインデックスで効率的な検索が可能
   - JSONBはバイナリ形式で保存され、パース不要

3. **型安全性**:
   - Goコードでは構造体にマッピング
   - スキーマ検証はアプリケーション層で実施

---

## マイグレーション方針

### Atlas運用
- スキーマ定義は `schema.hcl` が唯一の正
- 手動 `ALTER TABLE` は禁止
- expand/contract パターンで破壊的変更を回避

### 初回セットアップ手順
1. ENUM型を定義
2. コアエンティティ（users, subjects）を作成
3. 原本ファイル・チャンクテーブル（raw_files, materials）を作成
4. インデックス（B-tree, GIN, HNSW）を作成
5. チャット・非同期ジョブテーブル（chats, jobs）を作成

### 推奨拡張（MVP後）
- `subject_users`: 科目の共有・ロール管理
- `user_preferences`: ユーザー設定
- `audit_logs`: 詳細監査ログ
- `file_summaries`: 大量ファイル高速絞り込み用の要約（必要時のみ）

---

## パフォーマンス最適化指針

### 検索の主経路

```sql
-- keyword検索（全文検索 + 物理制約）
SELECT m.* FROM materials m
JOIN raw_files rf ON m.raw_file_id = rf.id
WHERE rf.subject_id = :subject_id
  AND rf.user_id = :user_id
  AND rf.is_active = TRUE
  AND m.is_active = TRUE
  AND to_tsvector('english', m.content_markdown) @@ plainto_tsquery('english', :query)
ORDER BY ts_rank(to_tsvector('english', m.content_markdown), plainto_tsquery('english', :query)) DESC
LIMIT :top_k;

-- vector検索（ベクトル類似度 + 物理制約）
SELECT m.*, m.embedding <=> :query_vector AS distance
FROM materials m
JOIN raw_files rf ON m.raw_file_id = rf.id
WHERE rf.subject_id = :subject_id
  AND rf.user_id = :user_id
  AND rf.is_active = TRUE
  AND m.is_active = TRUE
ORDER BY m.embedding <=> :query_vector
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
- [ ] 外部公開が必要なテーブルには `nanoid` (20文字) でユニーク制約が付いているか
- [ ] materials/jobsテーブルには `nanoid` が存在しないことを確認したか
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
