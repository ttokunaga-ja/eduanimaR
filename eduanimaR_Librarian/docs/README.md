# Docs Portal（eduanima-librarian）

この `docs/` は `eduanima-librarian`（Python 推論マイクロサービス）の SSOT。
本サービスは **DB-less** で、検索の物理実行・DB/インデックス管理・バッチは Go 側（Professor）が担う。
Professor との通信は **gRPC（双方向ストリーミング）** で行い、契約は `eduanimaR_Professor/proto/librarian/v1/librarian.proto` で定義される。

## Quickstart
0. `00_quickstart/QUICKSTART.md`
1. `00_quickstart/PROJECT_DECISIONS.md`

## まず読む（最短ルート）
1. サービス仕様（SSOT）: `01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md`
2. 責務境界: `01_architecture/MICROSERVICES_MAP.md`
3. 依存方向: `01_architecture/CLEAN_ARCHITECTURE.md`
4. 技術スタック（SSOT）: `02_tech_stack/STACK.md`
5. 通信/契約: `03_integration/INTER_SERVICE_COMM.md`, `03_integration/API_CONTRACT_WORKFLOW.md`
6. レジリエンス: `01_architecture/RESILIENCY.md`

## 契約・統合
- gRPC/Proto（Professor ↔ Librarian）: `03_integration/API_CONTRACT_WORKFLOW.md`、契約SSOT: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`
- バージョニング/廃止: `03_integration/API_VERSIONING_DEPRECATION.md`
- 契約テスト: `03_integration/CONTRACT_TESTING.md`
- エラー形式/コード: `03_integration/ERROR_HANDLING.md`, `03_integration/ERROR_CODES.md`

## 運用
- 観測性: `05_operations/OBSERVABILITY.md`
- SLO/アラート/Runbookの最小: `05_operations/SLO_ALERTING.md`
- CI/CD: `05_operations/CI_CD.md`

> DB/検索基盤/イベント同期の運用は Professor 側の SSOT を参照。
