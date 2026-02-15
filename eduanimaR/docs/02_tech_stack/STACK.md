# 技術スタック（SSOT：Web / Chrome Extension）

このドキュメントは、eduanima+R のフロントエンド（Web）および Chrome 拡張の技術スタックを SSOT として固定します。

要件の正：`../06_requirements/EDUANIMA_R_PRD.md`

---

## Executive Summary（Must）

- Web：**Next.js（React, App Router）+ TypeScript**
- Chrome 拡張：**Plasmo（React）**
- Styling：**Tailwind CSS**（拡張/UIの共通化とスコープ制御）
- Server State：**TanStack Query**
- リアルタイム表示：**SSE（Server-Sent Events）**（契約：`../03_integration/SSE_STREAMING.md`）
- API 契約：Professor の **OpenAPI を SSOT**（`../../eduanimaR_Professor/docs/openapi.yaml`）

## 開発フェーズと技術スタック

### Phase 1: ローカル開発（認証なし）
- Next.js（App Router）
- TanStack Query
- Tailwind CSS
- 認証: スキップ（固定のdev-user使用）
- 目的: API開発・検証、ベンチマーク実施

### Phase 2: 認証・セキュリティ
- 上記 + OAuth 2.0 / OpenID Connect
- SSO連携（Google / Meta / Microsoft / LINE）
- 拡張機能専用エンドポイントの追加
- Web版の機能制限（新規登録・科目登録・ファイルアップロード無効化）

### Phase 3: Chrome拡張機能
- Plasmo（React）
- Moodle DOM解析（MutationObserver）
- LMS SSO連携
- 既存APIとの統合
- **最重要機能**: Moodle資料の自動検知・アップロード

### Phase 4: 本番リリース
- 拡張機能: Chrome Web Store（非公開配布）
- Web版: 一般公開（拡張機能で登録したユーザーのみログイン可能）
- 拡張機能 + Web版の同時提供

---

## 1) Web（Phase 1〜）

### Framework

- Next.js（App Router）
- TypeScript

### UI/Styling

- Tailwind CSS

方針：
- 学習を妨げないクリーンな UI
- ソース（根拠URL）提示を UI のデフォルトとして設計

### Data Fetch / Cache

- TanStack Query（主に Client Component でのサーバー状態管理）

備考：
- 認証必須データのキャッシュ方針は `../03_integration/AUTH_SESSION.md` と整合させる

### Streaming

- SSE を採用（回答/進捗の即時性）
- 契約は `../03_integration/SSE_STREAMING.md` を正とする

---

## 2) Chrome 拡張（Phase 2〜）

### Framework

- Plasmo（Popup / Sidepanel / Content Script / Background を扱いやすい）

### UI/Styling

- Tailwind CSS

要点：
- LMS への干渉を避ける（スタイルの衝突を起こさない）
- Sidepanel を主戦場にしてチャットUXを提供

### 通信

- Professor（Go）へ HTTPS
- Q&A は SSE（`fetch` ストリームパースを推奨）

拡張の境界契約：`../03_integration/CHROME_EXTENSION_BACKEND_INTEGRATION.md`

---

## 3) API・型・契約（Must）

- Professor 外向き API は OpenAPI を SSOT とする
  - `../../eduanimaR_Professor/docs/openapi.yaml`

推奨：
- OpenAPI から TypeScript client / types を生成（Orval 等）し、手書きの型ズレを防ぐ

関連：
- `../03_integration/API_CONTRACT_WORKFLOW.md`
- `../03_integration/API_GEN.md`

---

## 4) Forms / Validation（推奨）

- React Hook Form
- Zod（スキーマ/バリデーション）

理由：
- 問い合わせフォーム、不具合申告、アップロード等で入力の失敗を UI で一貫して扱える

---

## 5) Testing（推奨）

- Unit/Component：Vitest
- E2E：Playwright

---

## 6) Lint / Quality（推奨）

- ESLint
-（採用しているなら）FSD 境界チェック（boundaries）

---

## 明確に「やらない」こと

- Web でアクセストークンを LocalStorage に保存する（`AUTH_SESSION.md` に反する）
- SSE のイベント名/形を UI 都合で壊す（契約破壊）
- 拡張で不要な権限/収集を行う（プライバシー/セキュリティ違反）
