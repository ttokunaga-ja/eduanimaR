# DB_SCHEMA_ERD

## 目的
eduanima-professor のデータベーススキーマを視覚的に理解するためのER図（Entity-Relationship Diagram）。

> 本ファイルは [DB_SCHEMA_TABLES.md](./DB_SCHEMA_TABLES.md) の視覚的補完資料。

---

## 全体構成図

```mermaid
erDiagram
    users ||--o{ subjects : "owns"
    users ||--o{ materials : "uploads"
    users ||--o{ chats : "creates"
    
    subjects ||--o{ materials : "contains"
    subjects ||--o{ chats : "scopes"
    
    chats ||--o{ chats : "parent of"
    
    materials ||--o{ material_pages : "has"
    materials ||--o{ chunks : "contains"
    materials ||--o{ ingest_jobs : "processes"
    materials ||--o{ session_evidence : "cited in"
    
    material_pages }o--|| materials : "belongs to"
    
    chunks }o--|| materials : "belongs to"
    chunks ||--o{ chunk_embeddings : "has"
    chunks ||--o{ session_evidence : "selected as"
    
    chunk_embeddings }o--|| chunks : "represents"
    
    chats ||--o{ search_steps : "executes"
    chats ||--o{ session_evidence : "collects"
    
    search_steps }o--|| chats : "belongs to"
    
    session_evidence }o--|| chats : "belongs to"
    session_evidence }o--|| chunks : "references"
    session_evidence }o--|| materials : "references"
    
    ingest_jobs }o--|| materials : "processes"
```

---

## コアエンティティ（ドメインモデル）

```mermaid
erDiagram
    users {
        uuid id PK
        text nanoid UK "外部公開ID"
        text provider "OAuth provider"
        text provider_user_id "IdP識別子"
        text email
        text display_name
        enum user_role "student/instructor/admin"
        timestamptz created_at
        timestamptz updated_at
        boolean is_active
        timestamptz deleted_at
    }
    
    subjects {
        uuid id PK
        text nanoid UK
        uuid owner_user_id FK
        text title
        text description
        text academic_year
        text semester
        text course_code
        timestamptz created_at
        timestamptz updated_at
        boolean is_active
        timestamptz deleted_at
    }
    
    materials {
        uuid id PK
        text nanoid UK
        uuid user_id FK "物理制約"
        uuid subject_id FK "物理制約"
        text title
        text original_filename
        enum material_type "pdf/powerpoint/..."
        bigint file_size_bytes
        text gcs_bucket "原本保存先"
        text gcs_object_path
        integer ingestion_version "LLM世代管理"
        enum status "pending/ingesting/ready/..."
        integer total_pages
        timestamptz created_at
        timestamptz updated_at
        timestamptz ingested_at
        boolean is_active
        timestamptz deleted_at
    }
    
    users ||--o{ subjects : "owner_user_id"
    users ||--o{ materials : "user_id"
    subjects ||--o{ materials : "subject_id"
```

---

## 資料・チャンク・Embedding

```mermaid
erDiagram
    materials {
        uuid id PK
        text nanoid UK
        uuid subject_id FK
        uuid user_id FK
        text title
        integer ingestion_version
        enum status
    }
    
    material_pages {
        uuid id PK
        uuid material_id FK
        integer page_number "1始まり"
        text content_markdown "OCR/Vision結果"
        jsonb metadata "構造化情報"
        timestamptz created_at
        timestamptz updated_at
    }
    
    chunks {
        uuid id PK
        text nanoid UK
        uuid material_id FK
        integer page_start
        integer page_end
        text content_markdown "意味単位チャンク"
        integer sequence_in_material "順序保証"
        integer generation "LLM世代管理"
        integer char_count
        timestamptz created_at
        timestamptz updated_at
        boolean is_active
        timestamptz deleted_at
    }
    
    chunk_embeddings {
        uuid id PK
        uuid chunk_id FK
        vector embedding "pgvector (768次元)"
        text model_name "Embeddingモデル"
        text model_version "世代管理"
        timestamptz created_at
    }
    
    materials ||--o{ material_pages : "material_id"
    materials ||--o{ chunks : "material_id"
    chunks ||--|| chunk_embeddings : "chunk_id"
```

---

## 推論セッション・検索・根拠

```mermaid
erDiagram
    chats {
        uuid id PK
        text nanoid UK
        uuid user_id FK "物理制約"
        uuid subject_id FK "物理制約"
        uuid parent_chat_id FK "会話の親子関係"
        text question "ユーザー質問"
        jsonb plan_json "Phase2: Plan結果"
        text final_answer_markdown "Phase4: Answer結果"
        enum status "active/completed/failed/..."
        integer max_search_steps "Librarianループ上限"
        integer actual_search_steps
        timestamptz created_at
        timestamptz updated_at
        timestamptz completed_at
        boolean is_active
        timestamptz deleted_at
    }
    
    search_steps {
        uuid id PK
        uuid session_id FK
        integer step_number "1,2,3..."
        text step_id "Librarian proto相関ID"
        enum search_mode "keyword/vector/hybrid"
        text query_text
        integer top_k
        text[] keywords
        integer result_count
        timestamptz created_at
    }
    
    session_evidence {
        uuid id PK
        uuid session_id FK
        uuid chunk_id FK "選定チャンク"
        uuid material_id FK "引用元資料"
        integer page_start
        integer page_end
        text snippet_markdown "表示スニペット"
        text selection_reason "Librarian選定理由"
        float search_score "参考値"
        timestamptz created_at
    }
    
    chats ||--o{ search_steps : "session_id"
    chats ||--o{ session_evidence : "session_id"
    session_evidence }o--|| chunks : "chunk_id"
    session_evidence }o--|| materials : "material_id"
```

---

## 非同期処理（Ingest Jobs）

```mermaid
erDiagram
    materials {
        uuid id PK
        text nanoid UK
        enum status
        integer ingestion_version
    }
    
    ingest_jobs {
        uuid id PK
        text nanoid UK
        uuid material_id FK
        text idempotency_key UK "Kafka冪等性"
        enum status "pending/processing/completed/failed/..."
        text gemini_model "使用モデル"
        enum gemini_phase "ingestion"
        text error_message
        integer retry_count
        integer max_retries
        timestamptz created_at
        timestamptz updated_at
        timestamptz started_at
        timestamptz completed_at
    }
    
    materials ||--o{ ingest_jobs : "material_id"
```

---

## 物理制約の強制（Security Boundary）

### 検索クエリでの強制パターン

```mermaid
graph TD
    A[Librarian: 検索要求] -->|gRPC: SearchRequest| B[Professor: 検索実行]
    B --> C{物理制約チェック}
    C -->|user_id| D[users.id]
    C -->|subject_id| E[subjects.id]
    C -->|is_active=true| F[ソフトデリート除外]
    D --> G[WHERE句に強制追加]
    E --> G
    F --> G
    G --> H[PostgreSQL検索実行]
    H --> I[結果セット]
    I -->|EvidenceChunk[]| J[Librarian: 評価]
    
    style C fill:#ff9999
    style G fill:#ff9999
    style H fill:#99ccff
```

### SQLでの実装例

```sql
-- ❌ 禁止: 物理制約なし（全データアクセス可能）
SELECT * FROM chunks WHERE content_markdown @@ :query;

-- ✅ 必須: 物理制約を強制
SELECT c.* 
FROM chunks c
JOIN materials m ON c.material_id = m.id
WHERE m.subject_id = :subject_id     -- 科目スコープ
  AND m.user_id = :user_id           -- ユーザースコープ
  AND m.is_active = TRUE             -- アクティブのみ
  AND c.is_active = TRUE
  AND c.content_markdown @@ :query;
```

---

## データフロー: Ingestion Loop

```mermaid
sequenceDiagram
    participant F as Frontend
    participant P as Professor
    participant G as GCS
    participant K as Kafka
    participant W as Professor Worker
    participant DB as PostgreSQL
    participant LLM as Gemini API

    F->>P: POST /api/materials/upload<br/>(file + subject_id)
    P->>P: 認証: user_id確定
    P->>DB: INSERT materials<br/>(status='uploading')
    P->>G: 原本アップロード
    G-->>P: GCS URI
    P->>DB: UPDATE materials<br/>(status='pending_ingestion', gcs_path)
    P->>DB: INSERT ingest_jobs<br/>(idempotency_key)
    P->>K: Produce IngestJob
    P-->>F: 202 Accepted {material_id}
    
    K->>W: Consume IngestJob
    W->>DB: SELECT material (冪等性チェック)
    W->>G: 原本ダウンロード
    W->>LLM: Vision Reasoning<br/>(Gemini 3 Flash Batch)
    LLM-->>W: Markdown + Structured Outputs
    W->>DB: INSERT material_pages
    W->>DB: INSERT chunks (sequence順)
    W->>LLM: Embedding生成
    LLM-->>W: vectors[]
    W->>DB: INSERT chunk_embeddings
    W->>DB: UPDATE materials<br/>(status='ready')
    W->>DB: UPDATE ingest_jobs<br/>(status='completed')
```

---

## データフロー: Reasoning Loop

```mermaid
sequenceDiagram
    participant F as Frontend
    participant P as Professor
    participant L as Librarian
    participant DB as PostgreSQL
    participant LLM as Gemini API

    F->>P: POST /api/questions<br/>(question + subject_id)
    P->>P: 認証: user_id確定
    
    Note over P,LLM: Phase 2: Planning
    P->>LLM: Plan生成<br/>(Gemini 3 Flash)
    LLM-->>P: plan_json<br/>(調査項目/停止条件)
    P->>DB: INSERT reasoning_sessions<br/>(plan_json)
    
    Note over P,L: Phase 3: Search Loop
    P->>L: gRPC: Reason(Start)<br/>(question + plan_json)
    
    loop 最大5回
        L->>L: LLM評価 (Gemini 3 Flash)
        L->>P: gRPC: SearchRequest<br/>(query + mode)
        P->>DB: 物理制約付き検索<br/>(subject_id + user_id)
        DB-->>P: chunks[] + metadata
        P->>L: gRPC: SearchResult<br/>(EvidenceChunk[])
        P->>DB: INSERT search_steps
    end
    
    L->>L: 停止条件判定
    L->>P: gRPC: Final<br/>(selected_evidence[])
    
    Note over P,LLM: Phase 4: Answer
    P->>DB: SELECT chunks<br/>(selected_evidence)
    DB-->>P: 全文Markdown
    P->>LLM: 最終回答生成<br/>(Gemini 3 Pro)
    LLM-->>P: final_answer_markdown
    P->>DB: INSERT session_evidence
    P->>DB: UPDATE reasoning_sessions<br/>(final_answer, status='completed')
    P-->>F: SSE: Stream Answer + Citations
```

---

## インデックス戦略

### 検索パフォーマンスの最適化

```mermaid
graph TD
    A[検索要求] --> B{検索モード}
    
    B -->|keyword| C[全文検索<br/>GIN Index]
    B -->|vector| D[ベクトル検索<br/>HNSW Index]
    B -->|hybrid| E[両方実行<br/>スコア統合]
    
    C --> F[物理制約フィルタ<br/>B-tree Index]
    D --> F
    E --> F
    
    F --> G[結果セット]
    
    style C fill:#99ccff
    style D fill:#99ccff
    style F fill:#ff9999
```

### インデックス一覧

| テーブル | インデックス | タイプ | 用途 |
|---------|------------|-------|------|
| users | `idx_users_provider` | B-tree | OAuth認証 |
| users | `idx_users_email` | B-tree | メール検索 |
| materials | `idx_materials_subject_user` | B-tree | **物理制約（主経路）** |
| chunks | `idx_chunks_material_seq` | B-tree | 順序保証取得 |
| chunks | `idx_chunks_content_fts` | GIN | **全文検索** |
| chunk_embeddings | `idx_chunk_embeddings_vector` | HNSW | **ベクトル検索** |

---

## スケーラビリティ考慮事項

### データ規模の見積もり（単一科目）

```mermaid
graph LR
    A[1科目] --> B[1000資料]
    B --> C[平均100ページ/資料]
    C --> D[10万ページ]
    D --> E[平均3チャンク/ページ]
    E --> F[30万チャンク]
    F --> G[30万 Embeddings<br/>768次元]
    
    G --> H[ストレージ試算]
    H --> I[30万 × 768 × 4 bytes<br/>≈ 900 MB]
```

### パーティショニング戦略（将来拡張）

```sql
-- 科目単位でのパーティショニング（大規模化時）
CREATE TABLE chunks_partitioned (
    LIKE chunks INCLUDING ALL
) PARTITION BY HASH (subject_id);

-- 月次パーティショニング（reasoning_sessions）
CREATE TABLE reasoning_sessions_partitioned (
    LIKE reasoning_sessions INCLUDING ALL
) PARTITION BY RANGE (created_at);
```

---

## セキュリティ境界の可視化

```mermaid
graph TB
    subgraph "外部境界"
        A[Frontend<br/>Chrome拡張/Web]
    end
    
    subgraph "Professor境界（DB/GCS独占）"
        B[Professor API]
        C[PostgreSQL]
        D[GCS]
        E[Kafka]
    end
    
    subgraph "Librarian境界（DBアクセス不可）"
        F[Librarian<br/>Python/LangGraph]
    end
    
    A -->|HTTP/JSON<br/>OAuth token| B
    B -->|gRPC<br/>user_id + subject_id| F
    F -->|SearchRequest| B
    B -->|物理制約強制| C
    B --> D
    B --> E
    
    style B fill:#99ccff
    style C fill:#ff9999
    style D fill:#ff9999
    style F fill:#99ff99
```

---

## 関連ドキュメント
- [DB_SCHEMA_TABLES.md](./DB_SCHEMA_TABLES.md) - テーブル定義（SSOT）
- [DB_SCHEMA_DISCUSSION.md](./DB_SCHEMA_DISCUSSION.md) - 議論ポイント
- [DB_SCHEMA_DESIGN.md](./DB_SCHEMA_DESIGN.md) - 設計原則
- [MICROSERVICES_MAP.md](./MICROSERVICES_MAP.md) - サービス境界
