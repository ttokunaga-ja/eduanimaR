# Quickstart（Frontend / FSD Template）

目的：eduanimaR フロントエンドを30分で起動・開発開始できる状態にする。

## 0) 前提
- **Node.js**: LTS 推奨（v20以上）
- **パッケージマネージャ**: `npm`（統一）
- **バックエンド**: Professor（Go）がローカルまたはCloud Runで稼働していること

## 1) ローカル起動（最短）
```bash
npm install
npm run dev
```

→ http://localhost:3000 でWebアプリが起動

## 2) 環境変数（Must）
`.env.example` をベースに `.env.local` を作成：

```env
# Professor（Go）のベースURL
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
API_BASE_URL=http://localhost:8080

# Phase 2以降（SSO）
# NEXT_PUBLIC_OAUTH_CLIENT_ID=your-client-id
# OAUTH_CLIENT_SECRET=your-secret
```

SSOT：`03_integration/DOCKER_ENV.md` と `05_operations/RELEASE.md`

## 3) API 生成（契約駆動の入口）
1. Professor の OpenAPI を取得：
   ```bash
   curl http://localhost:8080/swagger/openapi.yaml > openapi/openapi.yaml
   ```
2. 型・クライアント生成：
   ```bash
   npm run api:generate
   ```
   → `src/shared/api/generated/` に TypeScript コードが生成される

SSOT：`03_integration/API_GEN.md`

## 4) 最低限の品質ゲート（CIの骨格）
```bash
npm run lint          # ESLint + eslint-plugin-boundaries
npm run typecheck     # TypeScript
npm test              # Vitest
npm run build         # Next.js build
```

SSOT：`05_operations/CI_CD.md`

## 5) Chrome拡張機能（開発時）
```bash
npm run build:extension
```

→ `dist/extension/` を Chrome の「拡張機能を読み込む」で追加

## 6) 次に埋める（プロジェクト固有）
- `00_quickstart/PROJECT_DECISIONS.md`（本プロジェクトの決定事項）
- `01_architecture/SLICES_MAP.md`（新規機能追加時に slice を追記）
- `03_integration/ERROR_CODES.md`（Professor と同期）
