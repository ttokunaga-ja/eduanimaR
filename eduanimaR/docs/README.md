# Docs Portal（Frontend / FSD Template）

この `docs/` 配下は、Next.js（App Router）+ FSD（Feature-Sliced Design）での開発を「契約（運用ルール）」として固定するためのドキュメント集です。

目的：
- 判断のぶれ（人間/AI）を減らす
- 依存境界・契約駆動・運用の事故を先に潰す
- “本番だけ壊れる” を再現可能な手順に落とす

---

## Quickstart（最短で開発開始）
0. `00_quickstart/QUICKSTART.md`（30分で着手できる状態にする）
1. `00_quickstart/PROJECT_DECISIONS.md`（プロジェクト固有の決定事項SSOT）

## まず読む（最短ルート）
1. 技術スタック（SSOT）：`02_tech_stack/STACK.md`
2. FSDの概要：`01_architecture/FSD_OVERVIEW.md`
3. レイヤー境界：`01_architecture/FSD_LAYERS.md`
4. データ取得の契約（DAL）：`01_architecture/DATA_ACCESS_LAYER.md`
5. キャッシュ/再検証：`01_architecture/CACHING_STRATEGY.md`
6. API 生成：`03_integration/API_GEN.md`
7. API 契約運用：`03_integration/API_CONTRACT_WORKFLOW.md`
8. 失敗の標準：
   - `03_integration/ERROR_HANDLING.md`
   - `03_integration/ERROR_CODES.md`
9. セキュリティ（CSP/ヘッダー）：`03_integration/SECURITY_CSP.md`
10. 運用（最小）：
   - `05_operations/OBSERVABILITY.md`
   - `05_operations/RELEASE.md`
   - `05_operations/PERFORMANCE.md`

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
- `02_tech_stack/MUI_PIGMENT.md`
- `02_tech_stack/SSR_HYDRATION.md`
- `02_tech_stack/STATE_QUERY.md`
- `02_tech_stack/SERVER_ACTIONS.md`
- `02_tech_stack/ROUTING_UX_CONVENTIONS.md`

---

## Integration（契約/境界）
- API 生成：`03_integration/API_GEN.md`
- API 契約ワークフロー：`03_integration/API_CONTRACT_WORKFLOW.md`
- バージョニング/廃止：`03_integration/API_VERSIONING_DEPRECATION.md`
- エラー形式/扱い：`03_integration/ERROR_HANDLING.md`
- エラーコード：`03_integration/ERROR_CODES.md`
- CSP/ヘッダー：`03_integration/SECURITY_CSP.md`
- Auth/Session：`03_integration/AUTH_SESSION.md`
- i18n/Locale（必要な場合）：`03_integration/I18N_LOCALE.md`
- Docker 環境：`03_integration/DOCKER_ENV.md`

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
