# CI/CD（Frontend）

このドキュメントは、フロントエンド（Next.js + FSD）の CI/CD で **最低限守るゲート** を固定します。

目的：
- 境界違反（FSD）が本番へ入る
- 生成物ズレ（OpenAPI/Orval）
- `next build` は通るが本番で壊れる
を早期に止める。

関連：
- FSD境界：`../01_architecture/FSD_LAYERS.md`
- API契約：`../03_integration/API_CONTRACT_WORKFLOW.md`
- テスト：`../04_testing/TEST_STRATEGY.md`

---

## 結論（Must）

- `lint`（boundaries含む）/ `typecheck` / `test` / `build` を必須ゲートにする
- 翻訳キーの不足・未使用キーを検出する i18n チェック（例：`i18next-scanner` / 専用スクリプト）を CI に組み込み、UI文言のハードコーディングを防ぐ
- API生成は **差分検知** をCIで行う（生成し忘れ防止）
- main へのマージは “再現可能なビルド” が条件

---

## 標準ジョブ名と配置（SSOT）

このテンプレートで CI を組むときは、ジョブ名と配置を以下で固定する（チーム間で呼び方を揃える）。

- API 生成ドリフト検出（Must）
  - Job 名（推奨）: `api-generate-check`
  - 配置（推奨）: `.github/workflows/api-generate-check.yml`
  - 仕様（SSOT）: `../03_integration/API_GEN.md`

- i18n チェック（採用時の Must）
  - Job 名（推奨）: `i18n-check`
  - 配置（推奨）: `.github/workflows/i18n-check.yml`

- E2E smoke（採用時の Must）
  - Job 名（推奨）: `playwright-smoke`
  - 配置（推奨）: `.github/workflows/playwright-smoke.yml`

注意：SLO/アラートは CI の対象ではなく運用の契約。
- SSOT: `SLO_ALERTING.md` と `OBSERVABILITY.md`

### 契約テスト（Must）

- **契約コードチェック**（必須）
  - Job 名（推奨）: `contract-codegen-check`
  - 配置（推奨）: `.github/workflows/contract-codegen-check.yml`
  - 対象: `openapi/openapi.yaml` / `src/shared/api/generated/`
  - 実行内容: `npm run api:generate` → 差分検出

### セキュリティスキャン（Must）

- **依存関係の脆弱性検査**: `npm audit` または Dependabot
- **Secret scanning**: 誤ってコミットされたシークレットの検出
- **SAST**: 静的アプリケーションセキュリティテスト

### CI実行環境の最小権限化（推奨）

- **OIDC**: 短命クレデンシャルを優先
- **環境変数**: シークレットは環境変数で管理、コードに埋め込まない
- **読み取り専用トークン**: 可能な限り読み取り専用権限を使用

**参照元SSOT**:
- `../../eduanimaR_Professor/docs/05_operations/CI_CD.md`
- `../../eduanimaR_Librarian/docs/05_operations/CI_CD.md`

---

## 最低限のパイプライン（テンプレ）

PR（必須）：
1. `npm ci`
2. `npm run lint`
3. `npm run typecheck`
4. `npm test`
5. `npm run build`
6. `api-generate-check`（`npm run api:generate` → diff チェック）
7. （最小の）Playwright smoke（主要導線だけ）

main（任意）：
- デプロイ（staging → production）
- 監視（エラー率/Vitals）を確認し、ロールバック判断できる状態

推奨：main へのデプロイ後は `SLO_ALERTING.md` の対象導線をベースに、
RUM（Vitals/JSエラー）と BFF（5xx/latency）を確認できるダッシュボードを必須にする。

---

## API生成のCIチェック（推奨）

目的：OpenAPI/OrvalのズレをPRで止める。

- CIで `npm run api:generate` を実行し、生成物に差分が出ないことを確認
- 差分が出たら、生成し忘れ or OpenAPIの更新漏れ

推奨（運用ルール）：
- `.github/workflows/api-generate-check.yml` は **PRで必ず実行**し、差分が出た場合は PR をブロックする
- 実装例は `../03_integration/API_GEN.md` の「CI（Must）: 生成物ドリフト検出」を正とする

---

## i18n / 翻訳ファイルのCIチェック（推奨）

目的：UI文言のハードコーディングや翻訳漏れを早期に検出し、PRでブロックする。

- `npm run i18n:extract` を実行して翻訳キーを抽出（`i18next-scanner` の使用を推奨）
- `npm run i18n:check` を実行して、コードで参照されている翻訳キーが `public/locales/{lang}/*.json` に存在するかを検証する（missing keys は CI でエラーにする）
- `npm run lint` で `react/jsx-no-literals` を有効にして、JSX中のハードコーディング文字列を防ぐ

GitHub Actions のサンプルワークフローは `.github/workflows/i18n-check.yml` を参照してください。
---

## キャッシュとビルド（注意）

- `next build` は production の前提（開発サーバで動く＝本番OKではない）
- 動的化（CSP nonce 等）でコスト/キャッシュが変わるため、
  意図したレンダリング戦略になっているか確認する

関連：`../03_integration/SECURITY_CSP.md`

---

## 禁止（AI/人間共通）

- boundaries 違反を例外設定で握りつぶす
- 生成物を手編集する
- build を通さずにマージする
