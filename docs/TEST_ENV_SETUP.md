# テスト環境整備計画

**作成日**: 2026-02-18  
**対象**: eduanimaR (Next.js 15), eduanimaR_Professor (Go 1.25), eduanimaR_Librarian (Python 3.12)

---

## 現状サマリー

| コンポーネント | 状態 | 詳細 |
|---|---|---|
| **eduanimaR** (Frontend) | ⚠️ 未整備 | `vitest@3`, `@playwright/test` は package.json に存在するが、`vitest.config.ts` / `playwright.config.ts` / `@testing-library/react` がない |
| **eduanimaR_Professor** (Go) | ⚠️ 最小 | SSOT 契約テスト 1本のみ (`contracttest/ssot_test.go`)。Unit / Integration は未作成 |
| **eduanimaR_Librarian** (Python) | ❌ 未整備 | テストファイルなし。pytest の設定もなし |

---

## Phase 1: フロントエンド（eduanimaR）テスト環境

### 1-1. 不足パッケージのインストール

```bash
cd eduanimaR

# Vitest + React Testing Library（Unit / Component テスト用）
npm install --save-dev \
  @testing-library/react \
  @testing-library/user-event \
  @testing-library/jest-dom \
  @vitejs/plugin-react \
  jsdom \
  msw

# Playwright（E2E テスト用）は既にインストール済み
npx playwright install --with-deps chromium
```

### 1-2. `vitest.config.ts` の作成

```typescript
// eduanimaR/vitest.config.ts
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    exclude: ['node_modules', '.next', 'e2e'],
    coverage: {
      reporter: ['text', 'lcov'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/**/*.d.ts',
        'src/**/index.ts',          // barrel exports
        'src/shared/api/generated', // Orval 生成物
      ],
      thresholds: {
        statements: 60,
        branches: 60,
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
```

### 1-3. Vitest セットアップファイルの作成

```typescript
// eduanimaR/src/test/setup.ts
import '@testing-library/jest-dom'
```

### 1-4. `playwright.config.ts` の作成

```typescript
// eduanimaR/playwright.config.ts
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? 'github' : 'html',
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:3000',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  // E2E 実行前に dev server を起動する（ローカル）
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
  },
})
```

### 1-5. 優先して書くべきテスト（第1弾）

```
src/
  features/
    chat/
      ui/
        ChatInput.test.tsx         # ← 最優先：入力→送信ボタン活性化
        ChatStream.test.tsx        # ← SSE ストリーム表示の単体
      model/
        useChatStream.test.ts      # ← カスタムフックの境界ケース
  shared/
    lib/
      i18n/
        config.test.ts             # ← 純粋関数ユニット
e2e/
  home.spec.ts                     # ← ホームページ表示（最小E2E）
  chat.spec.ts                     # ← チャット導線の主要フロー
```

---

## Phase 2: Professor バックエンド（eduanimaR_Professor）テスト環境

### 2-1. 依存パッケージの追加

```bash
cd eduanimaR_Professor

# Unit テスト用
go get github.com/stretchr/testify@v1.9.0

# モック生成（インターフェース → モック自動生成）
go install github.com/vektra/mockery/v2@latest

# Integration テスト用（Testcontainers: Docker で実 DB/Kafka を起動）
go get github.com/testcontainers/testcontainers-go@v0.36.0
go get github.com/testcontainers/testcontainers-go/modules/postgres@v0.36.0
go get github.com/testcontainers/testcontainers-go/modules/kafka@v0.36.0
```

### 2-2. モック生成設定（`.mockery.yaml`）

```yaml
# eduanimaR_Professor/.mockery.yaml
with-expecter: true
dir: "internal/ports/mocks"
mockname: "Mock{{.InterfaceName}}"
outpkg: "mocks"
packages:
  github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports:
    interfaces:
      QuestionRepository:
      SubjectRepository:
      MaterialRepository:
      LLMClient:
      LibrarianClient:
      EventPublisher:
      StorageClient:
```

モック生成コマンド:
```bash
mockery
```

### 2-3. `Makefile` テストターゲット

```makefile
# eduanimaR_Professor/Makefile

.PHONY: test test-unit test-integration test-contract

## ユニットテスト（モックを使用、DB不要、高速）
test-unit:
	go test ./internal/usecases/... -v -count=1

## Integration テスト（Testcontainers で実 PostgreSQL を起動）
test-integration:
	go test ./internal/adapters/postgres/... -v -count=1 -tags=integration

## 契約テスト（OpenAPI / Proto SSOT 検証）
test-contract:
	go test ./internal/contracttest/... -v

## 全テスト
test: test-unit test-contract test-integration
```

### 2-4. 優先して書くべきテスト（第1弾）

#### ① UseCase ユニットテスト（モック使用）

```
internal/usecases/
  chat_usecase_test.go       # ← 最優先
    - 正常系：問いを投げると SSE イベントが返る
    - 異常系：subject が inactive → ErrSubjectNotActive
    - 境界条件：message が空 → バリデーションエラー
  ingest_usecase_test.go
    - 正常系：PDF アップロード → Kafka イベント発行
    - 冪等性：同一 material_id で重複呼び出し → エラーなし
```

#### ② Repository Integration テスト（Testcontainers + 実DB）

```
internal/adapters/postgres/
  question_repo_test.go      # sqlc 生成物の実動作確認
  subject_repo_test.go
```

#### ③ Handler テスト（Echo の httptest）

```
internal/adapters/http/handlers/
  chat_handler_test.go       # エラー形式（ErrorResponse）が仕様通りか
  material_handler_test.go
```

### 2-5. テスト用ヘルパーパッケージ

```
internal/testhelper/
  postgres.go     # Testcontainers で PostgreSQL を起動するヘルパー
  fixtures.go     # テストデータ（固定 seed）
  minio.go        # Testcontainers で MinIO を起動するヘルパー
```

---

## Phase 3: Librarian（eduanimaR_Librarian）テスト環境

### 3-1. pytest セットアップ

```bash
cd eduanimaR_Librarian

# pyproject.toml または requirements-dev.txt に追加
pip install pytest pytest-asyncio pytest-cov pytest-mock
```

`pyproject.toml` の設定:

```toml
[tool.pytest.ini_options]
asyncio_mode = "auto"
testpaths = ["tests"]
addopts = "--cov=src --cov-report=term-missing --cov-fail-under=50"

[tool.coverage.run]
source = ["src"]
omit = ["src/**/generated/*"]
```

### 3-2. テストディレクトリ構成

```
eduanimaR_Librarian/
  tests/
    unit/
      test_reason_service.py     # Reason ユースケース（LLM クライアントをモック）
      test_chunk_service.py      # チャンク分割ロジック
    integration/
      test_grpc_server.py        # gRPC サーバーの起動〜 RPC 呼び出し
    conftest.py                  # 共通フィクスチャ（mock LLM client など）
```

### 3-3. 優先して書くべきテスト（第1弾）

```python
# tests/unit/test_reason_service.py
async def test_reason_returns_citations_on_success():
    """Reason RPC が成功時に citations 付きで応答することを確認"""

async def test_reason_raises_on_empty_context():
    """コンテキストが空の場合 gRPC INVALID_ARGUMENT を返すことを確認"""

# tests/integration/test_grpc_server.py
async def test_grpc_server_starts_and_responds():
    """gRPC サーバーが起動し Reason RPC に応答することを確認"""
```

---

## Phase 4: CI/CD 統合

### GitHub Actions ワークフロー構成

```
.github/workflows/
  ci-frontend.yml     # lint + typecheck + vitest + playwright
  ci-professor.yml    # go vet + go test (unit + contract + integration)
  ci-librarian.yml    # ruff + mypy + pytest
```

#### `ci-professor.yml` の重要ジョブ

```yaml
jobs:
  test-unit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }
      - run: go test ./internal/usecases/... -v -count=1
        working-directory: eduanimaR_Professor

  test-contract:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }
      - run: go test ./internal/contracttest/... -v
        working-directory: eduanimaR_Professor

  test-integration:
    runs-on: ubuntu-latest
    # Testcontainers が Docker を使うため Docker-in-Docker は不要（runner に Docker あり）
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }
      - run: go test ./internal/adapters/... -v -count=1 -tags=integration
        working-directory: eduanimaR_Professor
```

---

## 実施順序（推奨）

```
Week 1:  Phase 2-①（Professor UseCase Unit テスト）
         → 最も "壊れると痛い" ビジネスロジックを先に守る

Week 2:  Phase 1（Frontend vitest 設定 + ChatInput テスト）
         → UI の主要コンポーネントをカバー

Week 3:  Phase 2-②（Professor Repo Integration テスト）
         → DB スキーマ変更の検知

Week 4:  Phase 3（Librarian pytest 基盤）
         + CI/CD ワークフロー統合（Phase 4）

Month 2: Playwright E2E の主要導線（ホーム → チャット → 回答表示）
```

---

## 現在 CI で動いている検査（既存）

| 検査 | ファイル | 状態 |
|---|---|---|
| OpenAPI SSOT 検証 | `contracttest/ssot_test.go` | ✅ 動作中 |
| Proto SSOT 検証 | `contracttest/ssot_test.go` | ✅ 動作中 |
| ESLint (FSD boundaries) | `eslint.config.mjs` | ✅ 設定済み |
| TypeScript 型チェック | `tsconfig.json` | ✅ `npm run typecheck` |

---

## 参照ドキュメント

- `eduanimaR/docs/04_testing/TEST_STRATEGY.md`
- `eduanimaR_Professor/docs/04_testing/TEST_STRATEGY.md`
- `eduanimaR_Professor/docs/04_testing/TEST_PYRAMID.md`
