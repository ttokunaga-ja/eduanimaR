# Docs Portal

この `docs/` 配下は **Professor（Go）側**の設計/運用/契約（SSOT）ドキュメント集です。

他コンポーネント（例: フロントエンド / Librarian 推論サービス）については、このリポジトリでは **通信（契約）と役割（責務境界）** のみを扱い、実装詳細は扱いません。

特に、検索戦略は **大戦略（Phase 2: GoがWHAT/停止条件）** と **小戦略（Phase 3: PythonがHOW/終了判定）** に分担し、Professor（Go）が検索の物理実行と最終回答生成を担います。Professor ↔ Librarian間の通信は **gRPC（双方向ストリーミング）** で行い、契約は `proto/librarian/v1/librarian.proto` で定義されています。責務分離の正は `01_architecture/MICROSERVICES_MAP.md` を参照してください。

## Quickstart（最短で開発開始）
0. `00_quickstart/QUICKSTART.md`
1. `00_quickstart/PROJECT_DECISIONS.md`（プロジェクト固有の決定事項SSOT）

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
