# API Integration & Code Generation

バックエンド（Goマイクロサービス）との通信コードは、OpenAPI 定義から自動生成します。

目的：
- 型定義の手書きを禁止し、契約ズレを根絶する
- API 呼び出しの入口を `shared/api` に集約し、実装のばらつきを防ぐ

---

## 1) ワークフロー（固定）

1. **OpenAPI の取得**：バックエンドリポジトリ/環境から最新の `openapi.yaml`（または `openapi.json`）を取得
2. **コード生成**：以下のコマンドを実行し、`src/shared/api` を更新
   ```bash
   npm run api:generate
   ```
3. **実装**：生成物（React Query hooks / client）を利用して実装

テンプレのデフォルト：
- OpenAPI 配置：`openapi/openapi.yaml`
- Orval 設定：`orval.config.ts`
- 生成先：`src/shared/api/generated`

補足：Orval は React Query hooks の生成に対応し、MSW handlers も生成できます（テストで有効）。

---

## 2) ディレクトリ規約（推奨）

```text
src/shared/api/
├── generated/          # 自動生成（手動編集禁止）
├── client.ts           # baseURL / 認証 / 共通fetcher（手書き）
├── errors.ts           # エラー分類（手書き・最小）
└── index.ts            # Public API（外部はここからのみimport）
```

### 生成物の扱い
- `generated/` は **100% 機械生成**。手で直したら次回生成で消える
- 手でやりたいこと（baseURL、ヘッダ、cookie、リトライ）は `client.ts` に閉じ込める

---

## 3) baseURL / 認証（ポリシー）

### baseURL
- baseURL は環境変数で切り替える（ハードコード禁止）
- 例：
  - ブラウザ向け：`NEXT_PUBLIC_API_BASE_URL`
  - サーバー向け：`API_BASE_URL`（server-only）

### 認証
- Cookie 認証：BFF（Next）経由の同一オリジンに寄せる（CORS/SameSite 地獄を避ける）
- Bearer：`client.ts` で header を一元付与（各featureで勝手に付けない）

---

## 4) 生成設定（方針）

プロジェクト固有の Orval 設定（`orval.config.*` 等）は、以下を満たすようにする：

- `operationId` を安定させる（生成 hook 名の安定化）
- React Query hooks を生成する（`useXxx`）
- 可能なら MSW handlers も生成する（テストの初期コストが下がる）

---

## 5) 禁止（AI/人間共通）

- コンポーネント内での手書き `fetch` / `axios`（生成物がある前提）
- エンドポイント/型の推測実装（OpenAPI に寄せる）
- `generated/` の手編集

---

## 6) Agent への最重要指示

バックエンドへのリクエストが必要な場合は、必ず `src/shared/api`（Public API）経由の生成物を使用してください。
`fetch` や `axios.get` を直接コンポーネントに書かないでください。

---

## 7) SSR/Hydration（必須）との統合

本テンプレートでは SSR/Hydration を **必須（Must）** とします。
そのため「サーバで取得して HTML に載せる」か「クライアントで hooks が取得する」かを、暗黙で混ぜないことが重要です。

基本：
- **Server（RSC / Route Handler / Server Action）**：生成クライアント（非 hook）を使用
- **Client**：生成 hooks（TanStack Query）を使用

SSR/Hydration で初期表示データを埋めたい場合（推奨）：
- Server（RSC）で `prefetch` → `dehydrate` し、Client で `HydrationBoundary` を使う
- これにより「初回表示に必須のデータを client で取りに行って白画面」が起きにくい

禁止：
- RSC が Route Handler を呼ぶ（サーバ内で余計な HTTP hop を作る）

詳細：
- SSR/Hydration の運用方針： [../02_tech_stack/SSR_HYDRATION.md](../02_tech_stack/SSR_HYDRATION.md)
- TanStack Query の統合： [../02_tech_stack/STATE_QUERY.md](../02_tech_stack/STATE_QUERY.md)

---

## 8) CI（Must）: 生成物ドリフト検出（api:generate の差分を落とす）

目的：
- PR 時点で **OpenAPI / 生成設定 / 生成物** の整合性を強制し、契約ズレを運用で発見しない

要件（Must）：
- CI で `npm run api:generate` を実行する
- 実行後に `src/shared/api`（特に `generated/`）へ差分が出ないこと（差分が出たら失敗）

注意：
- OpenAPI を別リポジトリ/環境から取得する場合は、生成前に `openapi.yaml/json` を取得してワークスペースへ配置する（取得方法はプロジェクトで統一）

例（GitHub Actions）：

```yaml
name: api-generate-check

on:
  pull_request:
    paths:
      - 'openapi/**'
      - 'orval.config.*'
      - 'package.json'
      - 'package-lock.json'
      - 'src/shared/api/**'
      - '.github/workflows/api-generate-check.yml'

jobs:
  api-generate-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: 'npm'

      - run: npm ci

      # OpenAPI を repo 外から取得する場合は、ここに取得ステップを入れる
      # - run: ./scripts/fetch-openapi.sh

      - run: npm run api:generate

      - name: Fail if generated output changed
        run: git diff --exit-code
```
