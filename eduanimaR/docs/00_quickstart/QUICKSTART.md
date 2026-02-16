# Quickstart（Frontend / FSD Template）

Last-updated: 2026-02-16

目的：eduanimaR フロントエンドを30分で起動・開発開始できる状態にする。

## 読む順序（推奨）

**まず最初に上流ドキュメントを参照してください**:
- [`../../eduanimaRHandbook/README.md`](../../eduanimaRHandbook/README.md) - サービス全体のコンセプトと設計原則（テンプレート説明）
- [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md) - Mission/Vision/North Star（**最重要**: 学習支援の目的と原則）
- [`../../eduanimaRHandbook/04_product/ROADMAP.md`](../../eduanimaRHandbook/04_product/ROADMAP.md) - Phase別のリリース計画（Phase 1〜4の詳細）
- [`../../eduanimaRHandbook/02_strategy/TECHNICAL_STRATEGY.md`](../../eduanimaRHandbook/02_strategy/TECHNICAL_STRATEGY.md) - 技術方針（検索戦略/セキュリティ前提/データ基盤）

サービス全体のコンセプトを理解してからフロントエンド実装に入ることで、設計判断の背景が明確になります。

## 0) 前提

- **Node.js**: LTS 推奨（v20以上、最新は v24.13.1）
- **パッケージマネージャ**: `npm`（統一）
- **バックエンド**: 
  - Professor（Go）がローカルまたはCloud Runで稼働
  - Librarian（Python）がローカルまたはCloud Runで稼働
  - Professor ↔ Librarian 間のHTTP/JSON通信が確立されていること（エンドポイント: `POST /v1/librarian/search-agent`）

**重要**: Librarianはステートレスサービス（会話履歴・キャッシュなし）です。1リクエスト内で推論が完結します。

**バックエンド詳細参照**:
- Professor: [`../../eduanimaR_Professor/docs/README.md`](../../eduanimaR_Professor/docs/README.md)
- Librarian: [`../../eduanimaR_Librarian/docs/README.md`](../../eduanimaR_Librarian/docs/README.md)

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

**契約駆動開発の原則**（参照: [`../03_integration/API_GEN.md`](../03_integration/API_GEN.md)）:
- 手書きの型定義を禁止し、契約ズレを根絶する
- API 呼び出しの入口を `shared/api` に集約し、実装のばらつきを防ぐ

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
- `/v1/qa/ask` (SSE): 質問 → Librarian推論ループ → 回答生成のストリーミング
- `/v1/materials/upload`: ファイルアップロード（Chrome拡張機能・curl使用、**Web版UIなし**）
- `/v1/subjects`: 科目一覧取得

SSOT：[`../03_integration/API_GEN.md`](../03_integration/API_GEN.md)、Professor: [`../../eduanimaR_Professor/docs/03_integration/API_GEN.md`](../../eduanimaR_Professor/docs/03_integration/API_GEN.md)

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

## 7) 汎用質問対応の動作確認（Phase 1必須）

Phase 1では、AI Agentによる質問対応パイプラインの動作確認が必須です。

### SSEエンドポイントのテスト（複数ユースケース）

#### 1. 明確な質問
```bash
curl -N http://localhost:8080/v1/qa/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "決定係数の計算式は？", "subject_id": "xxx-xxx-xxx"}'
```

#### 2. 曖昧な質問
```bash
curl -N http://localhost:8080/v1/qa/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "決定係数って何？", "subject_id": "xxx-xxx-xxx"}'
```

#### 3. 資料収集依頼
```bash
curl -N http://localhost:8080/v1/qa/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "統計学の資料を集めて", "subject_id": "xxx-xxx-xxx"}'
```

**期待されるSSEイベント**（すべてのユースケースで共通）:
- `event: thinking` → Agent が戦略を立案中
- `event: searching` → 資料を検索中
- `event: evidence` → 根拠資料を選定完了
- `event: answer` → 回答を生成中
- `event: done` → 完了

### フロントエンドでの確認
- **単一UI**で すべてのユースケースを処理できること（`features/qa-chat`）
- 推論状態がリアルタイム表示されること
- 参照元資料へのリンクがクリッカブルであること
- エラー時に再試行ボタンが表示されること

SSOT：`03_integration/API_CONTRACT_WORKFLOW.md`
