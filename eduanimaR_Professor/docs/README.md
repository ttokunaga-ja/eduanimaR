# Docs Portal

この `docs/` 配下は **Professor（Go）側**の設計/運用/契約（SSOT）ドキュメント集です。

他コンポーネント（例: フロントエンド / Librarian 推論サービス）については、このリポジトリでは **通信（契約）と役割（責務境界）** のみを扱い、実装詳細は扱いません。

特に、検索戦略は **大戦略（Phase 2: GoがWHAT/停止条件）** と **小戦略（Phase 3: PythonがHOW/終了判定）** に分担し、Professor（Go）が検索の物理実行と最終回答生成を担います。Professor ↔ Librarian間の通信は **gRPC（双方向ストリーミング）** で行い、契約は `proto/librarian/v1/librarian.proto` で定義されています。責務分離の正は `01_architecture/MICROSERVICES_MAP.md` を参照してください。

## Quickstart（最短で開発開始）
0. `00_quickstart/QUICKSTART.md`
1. `00_quickstart/PROJECT_DECISIONS.md`（プロジェクト固有の決定事項SSOT）

---

## Phase 1-5の役割（完全版定義）

### Phase 1: バックエンド完成 + Web版完全動作（ローカル開発）

**目的**: バックエンド機能を完全に完成させ、Web版で全機能を検証可能にする。

**実装完了条件**:
- Professor APIが完全に動作（OpenAPI定義完備）
- Librarian推論ループとの統合完了（gRPC双方向ストリーミング）
- **認証不要でcurlリクエストによる資料アップロードが可能**（開発用エンドポイント）
- OCR + 構造化処理（高速推論モデル）
- pgvector埋め込み生成・保存（HNSW検索）

**デプロイ**: ローカル環境のみ

---

### Phase 2: 拡張機能版作成 + 本番環境デプロイ

**目的**: 拡張機能版をZIPファイルで配布可能な状態にし、SSO認証を実装する。

**実装完了条件**:
- SSO認証基盤（OAuth/OIDC）
- 本番環境デプロイ（Google Cloud Run）
- 拡張機能からの資料自動アップロード本番適用

---

### Phase 3: Chrome Web Store公開

**目的**: Phase 2から変更なし（拡張機能のストア公開のみ）

---

### Phase 4: 閲覧中画面の解説機能追加

**目的**: 小テストなどで間違った場合に、間違った原因を資料をもとに考える支援機能を追加する。

**実装完了条件**:
- HTML・画像を受け取るエンドポイント追加
- Gemini Vision APIでの画像解析
- 資料との関連付けロジック追加

---

### Phase 5: 学習計画立案機能（構想段階）

**目的**: 過去の小テストや学習履歴をもとに、既存資料のどこを確認すべきかを提案する（構想段階）。

---

## Phase 1 OpenAPI仕様（最小版）

Professor の外向きAPI契約は、以下の最小セットから開始する。

### 必須エンドポイント

#### 1. 資料アップロード
```
POST /v1/subjects/{subjectId}/materials
Content-Type: multipart/form-data
Authorization: Bearer {token} （Phase 2以降）
X-Dev-User: dev-user （Phase 1のみ）

Request:
  - file: binary（PDF/PowerPoint）

Response (202 Accepted):
{
  "material_id": "uuid",
  "status": "processing"
}
```

#### 2. 質問応答（SSEストリーミング）
```
POST /v1/qa/stream
Content-Type: application/json
Accept: text/event-stream
Authorization: Bearer {token} （Phase 2以降）
X-Dev-User: dev-user （Phase 1のみ）

Request:
{
  "subject_id": "uuid",
  "question": "string"
}

Response (200 OK, SSE):
event: thinking
data: {"message": "検索戦略を立案中..."}

event: searching
data: {"message": "資料を検索中...（試行 1/5）"}

event: evidence
data: {
  "material_id": "uuid",
  "page_number": 12,
  "excerpt": "...",
  "why_relevant": "..."
}

event: answer
data: {"chunk": "回答の一部..."}

event: complete
data: {"message": "回答完了"}
```

#### 3. フィードバック送信
```
POST /v1/qa/feedback
Content-Type: application/json
Authorization: Bearer {token} （Phase 2以降）
X-Dev-User: dev-user （Phase 1のみ）

Request:
{
  "qa_session_id": "uuid",
  "feedback": "good" | "bad",
  "comment": "string (optional)"
}

Response (200 OK):
{
  "success": true
}
```

#### 4. 処理状態確認
```
GET /v1/materials/{materialId}/status

Response (200 OK):
{
  "material_id": "uuid",
  "status": "ready" | "processing" | "failed",
  "progress": integer (0-100)
}
```

#### 5. 科目一覧取得（Web版固有機能）
```
GET /v1/subjects
Authorization: Bearer {token} （Phase 2以降）
X-Dev-User: dev-user （Phase 1のみ）

Response (200 OK):
{
  "subjects": [
    {
      "subject_id": "uuid",
      "name": "科目名",
      "material_count": integer
    }
  ]
}
```

#### 6. 資料一覧取得（Web版固有機能）
```
GET /v1/subjects/{subjectId}/materials
Authorization: Bearer {token} （Phase 2以降）
X-Dev-User: dev-user （Phase 1のみ）

Response (200 OK):
{
  "materials": [
    {
      "material_id": "uuid",
      "title": "資料名",
      "upload_date": "ISO8601",
      "page_count": integer
    }
  ]
}
```

#### 7. 会話履歴取得（Web版固有機能）
```
GET /v1/subjects/{subjectId}/conversations
Authorization: Bearer {token} （Phase 2以降）
X-Dev-User: dev-user （Phase 1のみ）

Response (200 OK):
{
  "conversations": [
    {
      "conversation_id": "uuid",
      "question": "質問文",
      "created_at": "ISO8601"
    }
  ]
}
```

### OpenAPI 3.1.0完全版
詳細な仕様は `docs/openapi.yaml` を参照してください。
フロントエンド（eduanimaR）のOrval設定で自動生成するため、手書きクライアントは禁止。

---

## まず読む（最短ルート）
1. 技術スタック: `02_tech_stack/STACK.md`
2. 全体構成: `01_architecture/MICROSERVICES_MAP.md`
3. 依存方向: `01_architecture/CLEAN_ARCHITECTURE.md`
4. 通信/契約:
   - `03_integration/INTER_SERVICE_COMM.md`
   - `03_integration/API_CONTRACT_WORKFLOW.md`
   - `03_integration/PROTOBUF_GRPC_STANDARDS.md`
5. 同期（DB↔検索）: `01_architecture/SYNC_STRATEGY.md`

## 実装・統合（契約まわり）
- OpenAPI: `03_integration/API_CONTRACT_WORKFLOW.md`
- バージョニング/廃止: `03_integration/API_VERSIONING_DEPRECATION.md`
- 契約テスト: `03_integration/CONTRACT_TESTING.md`
- エラー形式/コード:
  - `03_integration/ERROR_HANDLING.md`
  - `03_integration/ERROR_CODES.md`
- gRPC/Proto標準: `03_integration/PROTOBUF_GRPC_STANDARDS.md`
- イベント契約（Kafka/DLQ/再処理）: `03_integration/EVENT_CONTRACTS.md`

## アーキテクチャ（詳細）
- Clean Architecture: `01_architecture/CLEAN_ARCHITECTURE.md`
- DB設計: `01_architecture/DB_SCHEMA_DESIGN.md`
- レジリエンス（timeout/retry/idempotency）: `01_architecture/RESILIENCY.md`

## テスト
- 戦略: `04_testing/TEST_STRATEGY.md`
- ピラミッド: `04_testing/TEST_PYRAMID.md`
- 性能/負荷: `04_testing/PERFORMANCE_LOAD_TESTING.md`

## 運用（本番で回す）
- 観測性: `05_operations/OBSERVABILITY.md`
- SLO/アラート/Runbookの最小: `05_operations/SLO_ALERTING.md`
- CI/CD: `05_operations/CI_CD.md`
- リリース/デプロイ: `05_operations/RELEASE_DEPLOY.md`
- 段階的リリース: `05_operations/PROGRESSIVE_DELIVERY.md`
- マイグレーション: `05_operations/MIGRATION_FLOW.md`
- APIセキュリティ: `05_operations/API_SECURITY.md`
- 監査ログ: `05_operations/AUDIT_LOGGING.md`
- Identity/Zero Trust: `05_operations/IDENTITY_ZERO_TRUST.md`
- Secrets/Key管理: `05_operations/SECRETS_KEY_MANAGEMENT.md`
- 脆弱性運用: `05_operations/VULNERABILITY_MANAGEMENT.md`
- サプライチェーンセキュリティ: `05_operations/SUPPLY_CHAIN_SECURITY.md`
- データ保護/DR: `05_operations/DATA_PROTECTION_DR.md`
- インシデント/ポストモーテム: `05_operations/INCIDENT_POSTMORTEM.md`

## フロントエンド（同居/分離どちらでも）
- FSD層: `01_architecture/FSD_LAYERS.md`
- TSの位置づけ: `02_tech_stack/TS_GUIDE.md`

> 注: フロントエンド実装詳細（コンポーネント設計など）は本リポジトリの対象外です。
> 本リポジトリでは、Professor（Go）側で必要な「契約（OpenAPI）と責務境界」のみを扱います。

## Skills（Agent向けの実務ルール集）
このテンプレートの前提（SSOT/禁止事項/安全なデフォルト/チェックリスト）を短くまとめた Skill ドキュメントです。

- ポータル: `skills/README.md`
- 一覧:
  - `skills/SKILL_STACK_SSOT.md`
  - `skills/SKILL_GO_1_25_BACKEND.md`
  - `skills/SKILL_DB_ATLAS_SQLC_PGX.md`
  - `skills/SKILL_DEPLOY_GCP_CLOUD_RUN.md`
  - `skills/SKILL_CONTRACTS_PROTO_GRPC_BUF.md`
  - `skills/SKILL_CONTRACTS_OPENAPI_ORVAL.md`
  - `skills/SKILL_RESILIENCY_TIMEOUTS_RETRIES_IDEMPOTENCY.md`
  - `skills/SKILL_OBSERVABILITY_OTEL_SLO.md`
  - `skills/SKILL_API_SECURITY_OWASP.md`
  - `skills/SKILL_SUPPLY_CHAIN_SLSA_SBOM.md`
  - `skills/SKILL_SEARCH_ELASTICSEARCH.md`
  - `skills/SKILL_EVENTS_CDC_KAFKA.md`
