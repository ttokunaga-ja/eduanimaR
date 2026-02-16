# Quickstart（Frontend / FSD Template）

Last-updated: 2026-02-15

目的：eduanimaR フロントエンドを30分で起動・開発開始できる状態にする。

## 0) 前提
- **Node.js**: LTS 推奨（v20以上）
- **パッケージマネージャ**: `npm`（統一）
- **バックエンド**: 
  - Professor（Go）がローカルまたはCloud Runで稼働
  - Librarian（Python）がローカルまたはCloud Runで稼働
  - Professor ↔ Librarian 間のgRPC通信が確立されていること

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

フロントエンドは Professor の OpenAPI 仕様から型・クライアントを自動生成します。

1. Professor の OpenAPI を取得：
   ```bash
   curl http://localhost:8080/swagger/openapi.yaml > openapi/openapi.yaml
   ```

2. 型・クライアント生成：
   ```bash
   npm run api:generate
   ```
   → `src/shared/api/generated/` に TypeScript コードが生成される

**重要なエンドポイント**:
- `/v1/qa/ask` (SSE): 質問 → Librarian推論 → 回答生成のストリーミング
- `/v1/materials/upload`: ファイルアップロード（拡張機能・curl使用）
- `/v1/subjects`: 科目一覧取得

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
- LMS資料の自動検知・アップロード機能を完全実装
- Professor APIへの直接通信（`/v1/materials/upload`）
- ローカル環境での動作検証（Chromeに手動読み込み）
- **Librarian推論ループとの統合確認**（質問機能のテスト）

### Phase 2での拡張機能公開
- **SSO認証**: LMS上でのGoogle/Meta/Microsoft/LINE認証
- **ユーザー登録**: 拡張機能からのみユーザー登録可能
- **Chrome Web Store公開**: 非公開配布として公開

**重要**: 本番環境（Phase 2）では、ファイルアップロード・ユーザー登録はChrome拡張機能からのみ実行可能。
Web版は拡張機能で登録したユーザーの閲覧専用チャネルとして機能。

## 7) Librarian推論ループの動作確認（Phase 1必須）

Phase 1では、Librarian統合が必須要件です。以下を確認してください：

### SSEエンドポイントのテスト
```bash
curl -N http://localhost:8080/v1/qa/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "決定係数とは？", "subject_id": "xxx-xxx-xxx"}'
```

**期待されるSSEイベント**:
- `event: thinking` → Professor が Phase 2（大戦略）を実行中
- `event: searching` → Librarian が検索戦略を立案中
- `event: evidence` → Librarian がエビデンスを選定
- `event: answer` → Professor が最終回答を生成中
- `event: done` → 完了

### フロントエンドでの確認
- チャット画面でリアルタイム推論状態が表示されること
- 参照元資料へのリンクがクリッカブルであること
- 推論失敗時に再試行ボタンが表示されること

SSOT：`03_integration/API_CONTRACT_WORKFLOW.md`
