# Docs Portal（Frontend / Chrome Extension）

この `docs/` 配下は、eduanima+R の **フロントエンド（Web）** と **Chrome 拡張** を実装/運用するための「契約（SSOT）」です。

目的：
- 要件・契約のぶれ（人間/AI）を減らす
- 境界（Frontend ↔ Professor）と失敗の扱いを固定する
- ストリーミング（SSE）/拡張（DOM監視）を安全に運用する

---

## Quickstart（最短で開発開始）
0. `00_quickstart/QUICKSTART.md`
1. `00_quickstart/PROJECT_DECISIONS.md`

## まず読む（上流 → 下流）
1. 総合要件（SSOT）：`06_requirements/EDUANIMA_R_PRD.md`
2. リリースロードマップ（SSOT）：`06_requirements/RELEASE_ROADMAP.md`
3. 技術スタック（SSOT）：`02_tech_stack/STACK.md`
4. API 契約運用：`03_integration/API_CONTRACT_WORKFLOW.md`
5. SSE（必須）：`03_integration/SSE_STREAMING.md`
6. Chrome 拡張連携：`03_integration/CHROME_EXTENSION_BACKEND_INTEGRATION.md`
7. 失敗の標準：`03_integration/ERROR_HANDLING.md` / `03_integration/ERROR_CODES.md`

---

## Architecture
- FSD：
  - `01_architecture/FSD_OVERVIEW.md`
  - `01_architecture/FSD_LAYERS.md`
  - `01_architecture/SLICES_MAP.md`
- Data Access / Cache：
  - `01_architecture/DATA_ACCESS_LAYER.md`
  - `01_architecture/CACHING_STRATEGY.md`
- UI設計：`01_architecture/COMPONENT_ARCHITECTURE.md`
- A11y（最小契約）：`01_architecture/ACCESSIBILITY.md`
- FSD ツール運用：`01_architecture/FSD_TOOLING.md`
- レジリエンス（FE版）：`01_architecture/RESILIENCY.md`

---

## Tech Stack
- `02_tech_stack/STACK.md`
- `02_tech_stack/SSR_HYDRATION.md`
- `02_tech_stack/STATE_QUERY.md`
- `02_tech_stack/SERVER_ACTIONS.md`
- `02_tech_stack/ROUTING_UX_CONVENTIONS.md`

---

## Integration（契約/境界）
- API 生成：`03_integration/API_GEN.md`
- API 契約ワークフロー：`03_integration/API_CONTRACT_WORKFLOW.md`
- バージョニング/廃止：`03_integration/API_VERSIONING_DEPRECATION.md`
- SSE：`03_integration/SSE_STREAMING.md`
- Chrome 拡張 ↔ Backend：`03_integration/CHROME_EXTENSION_BACKEND_INTEGRATION.md`
- エラー形式/扱い：`03_integration/ERROR_HANDLING.md`
- エラーコード：`03_integration/ERROR_CODES.md`
- CSP/ヘッダー：`03_integration/SECURITY_CSP.md`
- Auth/Session：`03_integration/AUTH_SESSION.md`
- i18n/Locale（必要な場合）：`03_integration/I18N_LOCALE.md`
- Docker 環境：`03_integration/DOCKER_ENV.md`

---

## Requirements（SSOT）
- `06_requirements/EDUANIMA_R_PRD.md`
- `06_requirements/RELEASE_ROADMAP.md`
- `06_requirements/README.md`

---

## Testing
- 戦略：`04_testing/TEST_STRATEGY.md`
- ピラミッド：`04_testing/TEST_PYRAMID.md`
- 性能（フロント）：`04_testing/PERFORMANCE_LOAD_TESTING.md`

---

## Operations
- 観測性：`05_operations/OBSERVABILITY.md`
- 性能：`05_operations/PERFORMANCE.md`
- リリース：`05_operations/RELEASE.md`
- CI/CD：`05_operations/CI_CD.md`
- SLO/アラート：`05_operations/SLO_ALERTING.md`
- Secrets/Key：`05_operations/SECRETS_KEY_MANAGEMENT.md`
- Identity/Zero Trust：`05_operations/IDENTITY_ZERO_TRUST.md`
- 脆弱性運用：`05_operations/VULNERABILITY_MANAGEMENT.md`
- サプライチェーン：`05_operations/SUPPLY_CHAIN_SECURITY.md`
- インシデント：`05_operations/INCIDENT_POSTMORTEM.md`

---

## Skills（Agent向け：短い実務ルール）
- `skills/README.md`

運用の基本：
- “迷ったらコードではなくドキュメントを更新して契約を変える”
- “例外は増やさず、境界の切り方を見直す”
