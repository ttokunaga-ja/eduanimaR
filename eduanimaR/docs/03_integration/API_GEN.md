---
Title: API Integration & Code Generation
Description: eduanimaRのAPI生成とOrval設定
Owner: @ttokunaga-ja
Status: Published
Last-updated: 2026-02-16
Tags: frontend, eduanimaR, api, orval, code-generation
---

# API Integration & Code Generation

Last-updated: 2026-02-16

バックエンド（Goマイクロサービス）との通信コードは、OpenAPI 定義から自動生成します。

目的：
- 型定義の手書きを禁止し、契約ズレを根絶する
- API 呼び出しの入口を `shared/api` に集約し、実装のばらつきを防ぐ

---

## 0) バックエンド（Professor）側の責務

### MUST（バックエンドが保証すること）

バックエンド（Professor）は以下を **OpenAPI 定義に明記**する責任があります：

1. **Breaking Changes / Compatible Changes の明記**
   - OpenAPI の `description` や `x-breaking-change` などのフィールドで明示
   - 互換性がない変更は事前に廃止手順（deprecation）を経る

2. **enum の意味とマッピング**
   - 可能な箇所で DB ENUM（PostgreSQL ENUM 等）を採用
   - API / DB / アプリケーション間で enum の意味が一致していること

3. **エラー形式とエラーコードの標準化**
   - エラーレスポンスの形式を統一（`ERROR_HANDLING.md` 参照）
   - エラーコードとUI挙動のマッピング（`ERROR_CODES.md` 参照）

4. **フロントエンドの責務範囲**:
   - フロントエンドはファイルアップロードUIを持たない（Chrome拡張機能またはAPI直接呼び出し）
   - フロントエンドは開発専用の認証UIを持たない（Phase 1はdev-user自動設定、Phase 2以降は拡張機能経由SSO）
   - Web版からの新規ユーザー登録は無効化（拡張機能のSSO登録のみ許可）

### 参照先（バックエンドドキュメント）

- Professor の API 生成運用：`../../../eduanimaR_Professor/docs/03_integration/API_GEN.md`
- Professor の API 契約ワークフロー：`../../../eduanimaR_Professor/docs/03_integration/API_CONTRACT_WORKFLOW.md`

---

## 1) 契約の配置場所（SSOT）

### バックエンド（Professor）側
- **OpenAPI定義**: `eduanimaR_Professor/docs/openapi.yaml`
  - Phase 1必須エンドポイント:
    - `POST /v1/auth/dev-login` (開発認証)
    - `POST /v1/qa/stream` (SSE応答)
    - `GET /v1/subjects` (科目一覧)
    - `GET /v1/subjects/{subject_id}/materials` (資料一覧)

### フロントエンド側
- **生成先**: `src/shared/api/generated/`
- **生成ツール**: Orval（設定: `orval.config.ts`）

### 生成コマンド
```bash
npm run api:generate
```

### 生成物の確認
- 型定義: `src/shared/api/generated/api.ts`
- クライアント関数: `src/shared/api/generated/client.ts`

### CI での検証
- `contract-codegen-check`で差分を検出（`../05_operations/CI_CD.md`参照）

---

## 2) ワークフロー（固定）

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

## 3) ディレクトリ規約（推奨）

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

## 4) baseURL / 認証（ポリシー）

### baseURL
- baseURL は環境変数で切り替える（ハードコード禁止）
- 例：
  - ブラウザ向け：`NEXT_PUBLIC_API_BASE_URL`
  - サーバー向け：`API_BASE_URL`（server-only）

### 認証
- Cookie 認証：BFF（Next）経由の同一オリジンに寄せる（CORS/SameSite 地獄を避ける）
- Bearer：`client.ts` で header を一元付与（各featureで勝手に付けない）

### Chrome拡張機能からのAPI通信

Chrome拡張機能からProfessor APIへの通信は、以下の方針で実装します。

#### 通信方式（Plasmo Framework前提）

1. **Content Scripts**:
   - LMS DOMを監視（MutationObserver）、資料を検知
   - Plasmo Messaging経由でBackground/Service Workerへファイルデータを送信
   - **HTTP通信は行わない**（CSP/CORS制約により直接通信は不可）

2. **Background/Service Worker**:
   - Content Scriptsからのメッセージを受信
   - Orval生成クライアント（`packages/shared-api`）を使用してProfessor APIへHTTPリクエスト
   - **CORS制約がない**ため、直接`http://localhost:8080`（開発）または`https://api.eduanimar.example.com`（本番）へ通信可能

#### 実装パターン（Plasmo Messaging）

**1. Background Message Handler（ファイルアップロード）**:

```typescript
// apps/extension/background/messages/upload-material.ts
import type { PlasmoMessaging } from "@plasmohq/messaging"
import { apiClient } from "@packages/shared-api"

export type UploadMaterialRequest = {
  file: File
  subjectId: string
}

export type UploadMaterialResponse = {
  success: boolean
  materialId?: string
  error?: string
}

const handler: PlasmoMessaging.MessageHandler<
  UploadMaterialRequest,
  UploadMaterialResponse
> = async (req, res) => {
  try {
    const { file, subjectId } = req.body
    const result = await apiClient.uploadMaterial(file, subjectId)
    res.send({ success: true, materialId: result.id })
  } catch (error) {
    res.send({ success: false, error: error.message })
  }
}

export default handler
```

**2. Content Script（資料検知・送信）**:

```typescript
// apps/extension/contents/lms-material-detector.ts
import { sendToBackground } from "@plasmohq/messaging"

// MutationObserverでLMS資料を検知
const observer = new MutationObserver(async (mutations) => {
  for (const mutation of mutations) {
    const materialLinks = detectMaterialLinks(mutation.target)
    for (const link of materialLinks) {
      const file = await fetchFileFromLink(link.href)
      const subjectId = extractSubjectId(link)
      
      // Background/Service Workerへ送信
      const response = await sendToBackground({
        name: "upload-material",
        body: { file, subjectId }
      })
      
      if (response.success) {
        console.log(`Uploaded: ${response.materialId}`)
      }
    }
  }
})

observer.observe(document.body, { childList: true, subtree: true })
```

**3. Sidepanel/Popup（質問送信、SSE受信）**:

```typescript
// apps/extension/sidepanel/components/QAChatPanel.tsx
import { sendToBackground } from "@plasmohq/messaging"
import { useQAStream } from "@packages/shared-api" // 共有Hook

export function QAChatPanel() {
  const { stream, send, isLoading } = useQAStream()
  
  const handleSend = async (question: string) => {
    // Background経由でSSE接続を確立
    const response = await sendToBackground({
      name: "qa-ask",
      body: { question, subjectId: "xxx" }
    })
    // SSEイベントはBackground → Sidepanelへストリーミング
  }
  
  return (
    <div>{/* UI */}</div>
  )
}
```

#### 認証トークン管理（Phase別）

| Phase | 認証方式 | トークン保存 | 実装詳細 |
|------|---------|------------|---------|
| **Phase 1** | `dev-user`固定 | 環境変数 | `PLASMO_PUBLIC_DEV_USER_TOKEN` を Background/Service Worker で読み取り、各リクエストの`Authorization`ヘッダーに付与 |
| **Phase 2** | SSO（OAuth/OIDC） | Chrome Storage API | SSO認証後、Access Tokenを`chrome.storage.local`に保存し、Background/Service Worker で各リクエストに付与 |

**実装例（Phase 2認証）**:

```typescript
// apps/extension/background/messages/auth-login.ts
import type { PlasmoMessaging } from "@plasmohq/messaging"
import { apiClient } from "@packages/shared-api"

const handler: PlasmoMessaging.MessageHandler = async (req, res) => {
  const { idToken } = req.body // SSOプロバイダーからのIDトークン
  
  try {
    const result = await apiClient.login({ idToken })
    
    // Access Tokenを Chrome Storage へ保存
    await chrome.storage.local.set({
      accessToken: result.accessToken,
      refreshToken: result.refreshToken,
      expiresAt: Date.now() + result.expiresIn * 1000
    })
    
    res.send({ success: true })
  } catch (error) {
    res.send({ success: false, error: error.message })
  }
}

export default handler
```

#### baseURL設定（環境別）

Chrome拡張機能では、環境変数を使用してbaseURLを切り替えます。

**Plasmo環境変数**:
- `.env.development`:
  ```
  PLASMO_PUBLIC_API_BASE_URL=http://localhost:8080
  ```
- `.env.production`:
  ```
  PLASMO_PUBLIC_API_BASE_URL=https://api.eduanimar.example.com
  ```

**API Client設定**:
```typescript
// packages/shared-api/src/client.ts
export const baseURL = 
  process.env.PLASMO_PUBLIC_API_BASE_URL || // Chrome拡張機能
  process.env.NEXT_PUBLIC_API_BASE_URL ||   // Next.js Web
  'http://localhost:8080'                   // Fallback
```

**参照**:
- Plasmo Messaging: https://docs.plasmo.com/framework/messaging
- [`../../eduanimaRHandbook/02_strategy/TECHNICAL_STRATEGY.md`](../../eduanimaRHandbook/02_strategy/TECHNICAL_STRATEGY.md) L128-144
- [`../00_quickstart/QUICKSTART.md`](../00_quickstart/QUICKSTART.md)（Chrome拡張機能セクション）

---

## 5) 生成設定（方針）

プロジェクト固有の Orval 設定（`orval.config.*` 等）は、以下を満たすようにする：

- `operationId` を安定させる（生成 hook 名の安定化）
- React Query hooks を生成する（`useXxx`）
- 可能なら MSW handlers も生成する（テストの初期コストが下がる）

---

## 6) 禁止（AI/人間共通）

- コンポーネント内での手書き `fetch` / `axios`（生成物がある前提）
- エンドポイント/型の推測実装（OpenAPI に寄せる）
- `generated/` の手編集

---

## 7) Agent への最重要指示

バックエンドへのリクエストが必要な場合は、必ず `src/shared/api`（Public API）経由の生成物を使用してください。
`fetch` や `axios.get` を直接コンポーネントに書かないでください。

---

## 8) SSR/Hydration（必須）との統合

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

## 9) CI（Must）: 生成物ドリフト検出（api:generate の差分を落とす）

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

---

## Breaking Changesの検出（Must）

### CI/CDでのOpenAPI契約変更検出

契約駆動開発を徹底するため、OpenAPI契約の変更を自動検出する仕組みを明記:

- **契約コードの配置**: 生成コードは `src/shared/api/generated/` に配置し、FSDの上位層から参照
- **Breaking Changes検出**: CI/CDで以下を検出
  - 既存エンドポイントの削除
  - 必須パラメータの追加
  - レスポンス型の変更
  - HTTPメソッドの変更

### Breaking Changesが検出された場合の対応

1. **Professor側の対応**: 
   - APIバージョニング (`/v1/`, `/v2/`)
   - 廃止予定の明示 (deprecated flag)
   - 移行期間の設定

2. **フロントエンド側の対応**:
   - 生成コードの更新 (`npm run api:generate`)
   - 影響範囲の特定 (TypeScriptエラーから判断)
   - 段階的な移行 (Feature Flagなど)

**参照元SSOT**:
- `../../eduanimaR_Professor/docs/03_integration/API_GEN.md`
- `../../eduanimaR_Professor/docs/03_integration/API_CONTRACT_WORKFLOW.md`

---

## Orval設定

### 設定ファイル
```typescript
// orval.config.ts
export default {
  professor: {
    input: './openapi.yaml',
    output: {
      mode: 'single',
      target: './src/shared/api/generated/professor.ts',
      client: 'react-query',
      mock: true,
    },
    hooks: {
      afterAllFilesWrite: 'prettier --write',
    },
  },
};
```

### 生成コマンド
```bash
npm run api:generate  # Orval実行 + Prettier
```

### 生成物の配置
- **ディレクトリ**: `src/shared/api/generated/`
- **ファイル**: `professor.ts` (型定義 + TanStack Queryフック)
- **コミット方針**: 生成物をコミットする（差分レビューのため）

### エラー型定義の自動生成

```yaml
# openapi.yaml
components:
  schemas:
    ErrorResponse:
      type: object
      required:
        - code
        - message
      properties:
        code:
          type: string
          enum: [MATERIAL_NOT_FOUND, SUBJECT_ACCESS_DENIED, ...]
        message:
          type: string
```

生成結果:
```typescript
export type ErrorResponse = {
  code: 'MATERIAL_NOT_FOUND' | 'SUBJECT_ACCESS_DENIED' | ...;
  message: string;
};
```
