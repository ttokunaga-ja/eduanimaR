# codingAgent_FeatureSlicedDesign_template

このフォルダは、Feature-Sliced Design (FSD) と MUI v6 (Pigment CSS) を採用したフロントエンド開発において、AI (CodingAgent) に迷わせないための Agent-First ドキュメントテンプレートです。

## 確定版：技術スタック（2026年時点）
- Next.js（App Router）
- TypeScript
- FSD（Feature-Sliced Design）
- MUI v6 + Pigment CSS（Zero-runtime CSS）
- TanStack Query v5/v6（サーバー状態管理）
- Orval（OpenAPI から TS 型 & クライアント生成）
- Zod（バリデーション）
- React Hook Form（フォーム）
- Vitest + Playwright（Testing）
- ESLint + eslint-plugin-boundaries（FSD 境界強制）
- Turbopack（Next.js 標準）
- Node.js（Docker / Next.js standalone）

詳細は `docs/02_tech_stack/STACK.md` を参照。

## 目的
- AI への「コンテキスト注入」と「制約強制」を最優先
- 少人数でもメンテナンス可能な分量・粒度

## まず読む場所（AI / 人間共通）
- `.cursorrules`（AI 常時制約）
- `docs/README.md`（Docs Portal：読む順の入口）
- `docs/01_architecture/FSD_OVERVIEW.md`（FSD 概要と運用原則）
- `docs/01_architecture/FSD_LAYERS.md`（FSD レイヤールール）
- `docs/01_architecture/DATA_ACCESS_LAYER.md`（DAL：認可/DTO最小化/取得の置き場所）
- `docs/01_architecture/CACHING_STRATEGY.md`（Next キャッシュ/再検証の契約）
- `docs/02_tech_stack/STACK.md`（確定版スタックと鉄則）
- `docs/02_tech_stack/MUI_PIGMENT.md`（Pigment CSS の DO / DON'T）
- `docs/02_tech_stack/SSR_HYDRATION.md`（SSR/Hydration：概念と例外判断）
- `docs/03_integration/API_CONTRACT_WORKFLOW.md`（OpenAPI 契約運用：変更/レビュー/CI差分検知）
- `docs/03_integration/ERROR_HANDLING.md`（失敗の標準：RSC/Route Handler/Client）
- `docs/03_integration/ERROR_CODES.md`（エラーコード→UI挙動のマッピング）
- `docs/03_integration/SECURITY_CSP.md`（CSP/セキュリティヘッダー方針）
- `docs/05_operations/OBSERVABILITY.md`（ログ/エラー/計測）
- `docs/05_operations/RELEASE.md`（環境/リリース/ロールバック）
- `docs/05_operations/PERFORMANCE.md`（性能チェックと運用）
- `docs/05_operations/CI_CD.md`（CI/CD：最低限のゲート）
- `docs/05_operations/SLO_ALERTING.md`（SLO/アラート：最小運用）
- `docs/skills/README.md`（Skills：Agent向けの短い実務ルール集）

## 使い方（テンプレ導入手順）
1. 本テンプレの `docs/` と `.cursorrules` をプロジェクトに持ち込む
2. 「プロジェクト固有で必ず埋める」項目を埋める（下記）
3. 新しい slice を作る前に `docs/01_architecture/SLICES_MAP.md` を更新する
4. 実装で迷ったら、まずドキュメントを修正して“契約”を更新する（コードより先に）

### Quickstart（最短で開発開始）
- `docs/00_quickstart/QUICKSTART.md`
- `docs/00_quickstart/PROJECT_DECISIONS.md`

## プロジェクト固有で必ず埋めるもの
- `docs/03_integration/DOCKER_ENV.md`：local/staging の baseURL、認証方式、proxy方針
- `docs/03_integration/API_GEN.md`：OpenAPI の取得方法、生成コマンド、生成物の置き場
- `docs/03_integration/API_VERSIONING_DEPRECATION.md`：互換性/段階移行/廃止の方針（プロジェクトで確定）
- `docs/02_tech_stack/STATE_QUERY.md`：RSC/Client の取得方針、エラー分類、mutation後の整合方針

---

## 本番前チェック（最小）

- SSR/Hydration が崩れていない（初期表示/操作、主要ページ）
- キャッシュ方針（tag/path/revalidate）がドキュメント通りになっている
- セキュリティヘッダー/CSP の方針が確定している
- Web Vitals / エラー観測が最低限入っている（運用で気づける）
- `next build` が通り、production 相当の動作確認ができている

## 注意（Pigment CSS）
Pigment CSS は仕様/実装の更新が続く可能性があります。
- 禁止事項（Emotion 等）を守る
- アップグレード時は `docs/02_tech_stack/MUI_PIGMENT.md` を先に更新する

## ディレクトリ
- `docs/`：ルールと契約（AI の判断材料）
- `src/`：FSD に基づくソースコード配置（実装は各プロジェクトで作成）

補足：このリポジトリの `src/` には **最小の雛形（FSD + Public API 前提）** が入っています。実プロジェクトでは `docs/01_architecture/FSD_LAYERS.md` の責務に従って、必要な slice を追加・拡張してください。
