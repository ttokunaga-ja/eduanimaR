# codingAgent_MicroServicesArchitecture_template

このフォルダは、Microservices + Gateway + Clean Architecture + 契約駆動 + 運用を前提にした **Agent-First ドキュメントテンプレート**です。

## まず読む（最短ルート）
- Docs Portal: [docs/README.md](docs/README.md)
- Stack（SSOT）: [docs/02_tech_stack/STACK.md](docs/02_tech_stack/STACK.md)
- 全体構成: [docs/01_architecture/MICROSERVICES_MAP.md](docs/01_architecture/MICROSERVICES_MAP.md)
- 依存方向: [docs/01_architecture/CLEAN_ARCHITECTURE.md](docs/01_architecture/CLEAN_ARCHITECTURE.md)
- Deploy（Cloud Run）: [docs/skills/SKILL_DEPLOY_GCP_CLOUD_RUN.md](docs/skills/SKILL_DEPLOY_GCP_CLOUD_RUN.md)

## 目的
- AI/人間の判断のぶれを減らす（SSOT・禁止事項・安全なデフォルトを明文化）
- “本番だけ壊れる” を運用手順に落とす（CI/CD、Secrets、観測性、リリース、復旧）

## 使い方（テンプレ導入手順）
1. このテンプレの `docs/` と `.cursorrules` をプロジェクトに持ち込む
2. プロジェクト固有の前提（後述）を埋め、SSOTを確定させる
3. 実装で迷ったら、まずドキュメントを修正して“契約”を更新する

### Quickstart（最短で開発開始）
- `docs/00_quickstart/QUICKSTART.md`
- `docs/00_quickstart/PROJECT_DECISIONS.md`

## プロジェクト固有で必ず埋めるもの（最低限）
- サービス境界/責務: [docs/01_architecture/MICROSERVICES_MAP.md](docs/01_architecture/MICROSERVICES_MAP.md)
- 外向き契約（OpenAPI）と運用: [docs/03_integration/API_CONTRACT_WORKFLOW.md](docs/03_integration/API_CONTRACT_WORKFLOW.md)
- 内向き契約（Proto/gRPC）標準: [docs/03_integration/PROTOBUF_GRPC_STANDARDS.md](docs/03_integration/PROTOBUF_GRPC_STANDARDS.md)
- エラー形式/コード: [docs/03_integration/ERROR_HANDLING.md](docs/03_integration/ERROR_HANDLING.md), [docs/03_integration/ERROR_CODES.md](docs/03_integration/ERROR_CODES.md)
- CI の最低ゲート: [docs/05_operations/CI_CD.md](docs/05_operations/CI_CD.md)
- リリース/ロールバック方針: [docs/05_operations/RELEASE_DEPLOY.md](docs/05_operations/RELEASE_DEPLOY.md)
- Secrets/Key 管理: [docs/05_operations/SECRETS_KEY_MANAGEMENT.md](docs/05_operations/SECRETS_KEY_MANAGEMENT.md)

## 注意
- 本テンプレは「ドキュメントが正（SSOT）」です。コード/構成はプロジェクトで最適化しつつ、SSOTとの整合を優先してください。
