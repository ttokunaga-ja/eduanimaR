# PROJECT_DECISIONS（eduanimaR_Professor固有の決定事項）

Owner: @ttokunaga-ja  
Status: Published  
Last-updated: 2026-02-17  
Tags: professor, backend, decisions

---

## 1. プロジェクトの性質
eduanimaR_Professor（Go バックエンド）は、**学習効果検証のための研究プロジェクト**として位置づける。

- 収益化は研究完了後の検討事項（Phase 1〜4では非営利）
- Phase 1の主目的は「技術的実現可能性の検証」と「学習効果の測定」

---

## 2. Phase 1のスコープ固定（SSOT）

### 提供する機能
- 資料アップロード（PDF/PowerPoint → GCS保存）
- OCR + 構造化（高速推論モデル使用）
- pgvector埋め込み生成・保存
- Q&A API（単一科目内検索 + 根拠提示）

### スコープ外
- SSO認証（dev-user固定）
- 複数ユーザー対応
- Kafka非同期処理（Phase 1は同期処理のみ）
- Elasticsearch（Phase 1はpgvectorのみ）

---

## 3. 技術的決定事項

### データベース
- PostgreSQL 18.1 + pgvector 0.8.1
- マイグレーション管理: Atlas v1.0.0
- クエリ生成: sqlc 1.30.0
- ドライバ: pgx v5.8.0

### 外部API
- OCR/構造化: 高速推論モデル
- 埋め込み生成: Gemini Embedding（768次元）

### デプロイ
- Phase 1: ローカル実行のみ（Docker Compose）
- Phase 2以降: Google Cloud Run

---

## 4. OpenAPI契約（Phase 1版）

### 必須エンドポイント
1. `POST /subjects/{subjectId}/materials` - 資料アップロード
2. `POST /qa` - 質問応答
3. `GET /materials/{materialId}/status` - 処理状態確認

### 契約の配置
- SSOT: `eduanimaR_Professor/docs/openapi.yaml`
- 生成先（Frontend）: `eduanimaR/src/shared/api/` （Orval自動生成）

---

## 5. 研究データ収集方針

### 取得するデータ
- OCR精度（文字認識率、処理時間）
- 検索応答時間（p50/p95/p99）
- ユーザーフィードバック（根拠の有用性5段階評価）

### 倫理的配慮
- 個人を特定可能なデータは取得しない
- 学習行動データは匿名化して研究利用
- 被験者への事前説明と書面同意を必須化

---

## 6. Phase 1の完了条件

1. 検索成功率70%以上（10件の検証質問で7件成功）
2. 検索応答時間p95で5秒以内
3. ハルシネーション率20%以下
4. 5名以上の被験者から肯定的評価

上記を達成した場合のみ、Phase 2（SSO認証+複数ユーザー）へ移行する。

---

## 7. Phase 1: ローカル開発・固定ユーザー（詳細契約）

Last-updated: 2026-02-17  
Status: Published  
Owner: @ttokunaga-ja

### 認証・認可

- **認証方式**: Phase 1では認証をスキップ
- **固定ユーザー**: 
  - user_id: `dev-user-001` (UUID: `00000000-0000-0000-0000-000000000001`)
  - email: `dev@example.com`
- **実装方針**: 
  - リクエストヘッダー `X-Dev-User: dev-user-001` で固定ユーザーを識別
  - Professor側のミドルウェアで `context` に `user_id` を注入
  - Phase 2でSSO（OAuth/OIDC）実装時に差し替え

### API契約（Professor: OpenAPI）

#### 契約SSOT
`eduanimaR_Professor/docs/openapi.yaml`

#### 最小エンドポイント（Phase 1）

1. **POST /v1/materials**
   - 目的: ファイルアップロード（Chrome拡張機能→Professor）
   - Request: `multipart/form-data` (file, subject_id)
   - Response: `{ material_id: string, job_id: string, status: "pending" }`

2. **POST /v1/questions** + **GET /v1/questions/{request_id}/events**
   - 目的: 質問送信と応答受信（拡張機能/Web→Professor）
   - Request: `{ subject_id: string, question: string }`
   - Response (202): `{ request_id: string }`
   - SSE Stream: `event: progress|answer|done, data: { type, content, ... }`

3. **GET /v1/subjects/{subject_id}/materials**
   - 目的: 科目に紐づくファイル一覧取得
   - Response: `[{ id, filename, uploaded_at }]`

4. **POST /v1/auth/dev-login** (Phase 1専用)
   - 目的: 開発用固定ユーザー認証
   - Response: `{ user_id: "dev-user", authenticated: true }`
   - 注意: Phase 2でSSO実装時に削除

### gRPC契約（Professor ↔ Librarian）

**契約SSOT**: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`

現在の契約は Phase 3 のフル機能（Reason RPC）を定義していますが、Phase 1では以下のシンプルな使い方を想定:

```proto
service LibrarianService {
  rpc Reason(stream ReasoningInput) returns (stream ReasoningOutput);
}

message ReasoningInput {
  string request_id = 1;
  oneof payload {
    Start start = 10;           // 質問開始
    SearchResult search_result = 11; // Professor から検索結果を返す
    Cancel cancel = 12;         // キャンセル
  }
}

message ReasoningOutput {
  string request_id = 1;
  oneof payload {
    Progress progress = 10;           // 進捗通知
    SearchRequest search_request = 11; // Librarian が検索をリクエスト
    Final final = 12;                  // 最終回答
  }
}
```

**Phase 1での使い方**:
- Professor が Start メッセージを送信（question, user_id, subject_id を含む）
- Librarian が SearchRequest で検索を要求
- Professor が物理制約（user_id/subject_id）を強制した検索結果を返す
- Librarian が Final で回答を返す

### データフロー（Phase 1）

1. **ファイルアップロード**:
   Chrome拡張 → Professor (POST /v1/materials) → GCS → Kafka (IngestJob) → Worker (OCR/Chunk/Embed) → PostgreSQL (chunks)

2. **質問応答**:
   拡張/Web → Professor (POST /v1/questions) → Librarian (gRPC Reason) → Professor (Vector Search) → Librarian (Plan/Evaluate) → Professor (SSE via /v1/questions/{request_id}/events) → 拡張/Web

### エラーハンドリング方針

- **ファイルサイズ制限**: 10MB（超過時は `FILE_TOO_LARGE`）
- **処理タイムアウト**: 質問応答は60秒（超過時は `REQUEST_TIMEOUT`）
- **検索結果なし**: `NO_SEARCH_RESULTS` を返し、UI側で「関連資料が見つかりませんでした」と表示

### Phase 2への移行方針

- 認証: 固定ユーザーミドルウェアを削除し、OAuth/OIDCミドルウェアに差し替え
- データベース: `users` テーブルに実ユーザーを追加（Phase 1の固定ユーザーは削除）
- API契約: 後方互換を維持（追加のみ、削除は非推奨期間を設ける）
