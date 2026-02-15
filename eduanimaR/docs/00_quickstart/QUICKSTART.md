# Quickstart（Frontend / FSD Template）

Last-updated: 2026-02-15

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

# Phase 2（SSO）
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

## 5) 開発環境でのファイルアップロード
フロントエンドはファイルアップロードUIを持ちません。
開発時のファイルアップロードは、外部ツール（curl, Postman等）でProfessor APIへ直接実行します。

```bash
# 例: curlでファイルアップロード
curl -X POST http://localhost:8080/api/materials/upload \
  -H "Content-Type: multipart/form-data" \
  -F "file=@/path/to/document.pdf" \
  -F "subject_id=xxx-xxx-xxx"
```

## 6) Chrome拡張機能（Phase 1から実装）
```bash
npm run build:extension
```

→ `dist/extension/` を Chrome の「拡張機能を読み込む」で追加

### Phase 1での拡張機能開発
- **自動アップロード機能**: LMSページから資料を自動検知・アップロードする機能を実装
- **ローカルテスト**: Chrome拡張機能をローカルで読み込み、Moodleテストサイトで動作確認
- **認証なし**: dev-user固定で動作（Professor APIが開発モードで自動設定）

### Phase 2での拡張機能公開
- **SSO認証**: LMS上でのGoogle/Meta/Microsoft/LINE認証
- **ユーザー登録**: 拡張機能からのみユーザー登録可能
- **Chrome Web Store公開**: 非公開配布として公開

**重要**: 本番環境（Phase 2）では、ファイルアップロード・ユーザー登録はChrome拡張機能からのみ実行可能。
Web版は拡張機能で登録したユーザーの閲覧専用チャネルとして機能。

## 7) 次に埋める（プロジェクト固有）
- `00_quickstart/PROJECT_DECISIONS.md`（本プロジェクトの決定事項）
- `01_architecture/SLICES_MAP.md`（新規機能追加時に slice を追記）
- `03_integration/AUTH_SESSION.md`（SSO認証・セッション管理の詳細）
- `03_integration/ERROR_CODES.md`（Professor と同期）
