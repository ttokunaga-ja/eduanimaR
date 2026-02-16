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

---

## Phase別の実装制約（SSOT）

eduanimaRは段階的リリースを前提とし、Phase 1〜4で機能を積み上げます。各Phaseで「実装すべきこと」「実装してはならないこと」を明確にします。

詳細は [`../../eduanimaRHandbook/04_product/ROADMAP.md`](../../eduanimaRHandbook/04_product/ROADMAP.md) を参照してください。

---

### Phase 1（ローカル開発 + Librarian統合）

**目的**: ローカル環境でのLibrarian推論ループの動作確認

#### ✅ 実装すべきこと

1. **基本的なQA UI**（`features/qa-chat`）
   - ユーザー入力欄（テキスト）
   - SSE受信ロジック（thinking/searching/evidence/answer）
   - エビデンス表示（資料名・ページ・抜粋・クリッカブルURL）
   
2. **Chrome拡張機能の自動アップロード**
   - LMS資料の自動検知
   - Professor API (`POST /v1/files/upload`) への自動送信
   - アップロード状態の表示
   
3. **開発環境での動作確認**
   - Web版: curlやPostmanでAPIテスト + SSE動作確認
   - 拡張機能: Chromeにローカル読み込みで動作確認

#### ❌ 実装してはならないこと

1. **ファイルアップロードUI**
   - Web版にファイル選択・アップロードUIを実装してはならない
   - API直接呼び出し（curl/Postman）で代替する
   
2. **ユーザー登録UI**
   - Phase 2のSSO実装後に対応
   
3. **本番環境へのデプロイ**
   - Phase 1は開発環境のみ

---

### Phase 2（本番環境・同時リリース）

**目的**: Chrome拡張機能とWebアプリを同時に本番リリース

#### ✅ 追加すべきこと

1. **SSO認証**（Google/Meta/Microsoft/LINE）
   - `features/auth-sso` の実装
   - Professor API (`POST /v1/auth/login`) との統合
   
2. **Chrome Web Store公開**
   - 非公開配布（限定公開）
   - 拡張機能でのユーザー登録フロー
   
3. **Web版からの未登録ユーザー誘導UI**
   - SSO認証後、`AUTH_USER_NOT_REGISTERED` を受信した場合
   - 拡張機能ダウンロードページへ誘導
   
4. **Librarian連携の本番適用**
   - Phase 1で実装済みの推論ループを本番環境で稼働

#### ❌ 実装してはならないこと

1. **Web版からの新規登録UI**
   - 新規登録は拡張機能でのみ可能
   - Web版は「既存ユーザーのログイン専用」
   
2. **拡張機能以外のアップロードUI**
   - Web版にファイルアップロードUIを実装してはならない
   - Phase 2でも拡張機能の自動アップロードのみ

---

### Phase 3以降（将来）

**予定機能**:
- 学習ロードマップ生成（Learning Support）
- 小テストHTML解析（Feedback Loop）
- コンテキスト自動認識サポート（Seamless Experience）

詳細は [`../../eduanimaRHandbook/04_product/ROADMAP.md`](../../eduanimaRHandbook/04_product/ROADMAP.md) を参照。

---

## 0) 前提

- **Node.js**: LTS 推奨（v20以上、最新は v24.13.1）
- **パッケージマネージャ**: `npm`（統一）
- **バックエンド**: 
  - Professor（Go）がローカルまたはCloud Runで稼働
  - Librarian（Python）がローカルまたはCloud Runで稼働
  - Professor ↔ Librarian 間の**gRPC通信**が確立されていること

**重要**: 
- **Librarian**: ステートレスサービス（会話履歴・キャッシュなし）、Professorが決定した戦略に基づきクエリ生成
- **Professor**: 検索戦略決定（「検索実行 vs ヒアリング」の判断）、終了条件決定、データ守護者
- **通信**: Professor ↔ Librarian間は **gRPC** (proto定義: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`)、Frontend ↔ Professor間は HTTP/JSON + SSE

**AI Agent質問システムの柔軟性**:
- **曖昧な質問**: Professor Phase 2でヒアリング判断 → Phase 4-Aで意図候補3つ提示 → ユーザー選択後にPhase 2再実行
- **明確な質問**: Phase 2で検索戦略決定 → Phase 3でLibrarian経由検索 → Phase 4-Bで回答生成
- **意図選択フロー**: Chrome拡張の表示範囲制約により、候補は3つ固定で提示

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
