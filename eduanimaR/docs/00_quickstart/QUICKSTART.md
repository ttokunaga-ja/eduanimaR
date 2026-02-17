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
   - **汎用質問対応の検証**: 明確な質問/曖昧な質問/資料収集の3パターンをテスト（詳細は「7) 汎用質問対応の動作確認」を参照）

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

### Phase 3: Chrome Web Store公開

**目的**: 拡張機能をChrome Web Storeで公開し、より広範なユーザーへ提供する。

**実装完了条件**:
- Chrome Web Storeで公開（非公開配布 or 限定公開）
- ストア審査対応（プライバシーポリシー・スクリーンショット等）
- Web版・バックエンドはPhase 2から変更なし

---

### Phase 4: 閲覧中画面の解説機能追加

**目的**: 小テストなどで間違った場合に、間違った原因を資料をもとに考える支援機能を追加する。

**実装完了条件**:

1. **拡張機能版**
   - 現在閲覧中の画面のHTML取得
   - 画面内に表示されている画像ファイル取得（図・グラフ等）
   - 取得したHTML・画像をProfessor APIへ送信
   - LLMによる解説生成（資料を根拠に表示）

2. **バックエンド（Professor）**
   - HTML・画像を受け取るエンドポイント追加
   - Gemini Vision APIでの画像解析
   - 資料との関連付けロジック追加

3. **ユースケース**
   - 小テストで間違った問題の画面を開く
   - 拡張機能で「この画面を解説」ボタンをクリック
   - LLMが資料を根拠に解説を生成

4. **注意事項**
   - 取得したHTML・画像は「その場の解析のみ（保存しない/短期）」とする

---

### Phase 5: 学習計画立案機能（構想段階）

**目的**: 過去の小テストや学習履歴をもとに、既存資料のどこを確認すべきかを提案する。

**備考（構想レベル）**:

1. **学習計画チャット**
   - 過去の小テスト結果を取得
   - 間違いの傾向分析
   - 既存資料の「どこを確認すべきか」「どの順序で学ぶべきか」をチャットで提案

2. **実装方針（未確定）**
   - 小テスト結果の保存方式（短期/長期）
   - 学習履歴の匿名化
   - プライバシー配慮（被験者データの扱い）

**注**: Phase 5は構想段階であり、Phase 1-4の完了後に詳細を検討する。

詳細は [`../../eduanimaRHandbook/04_product/ROADMAP.md`](../../eduanimaRHandbook/04_product/ROADMAP.md) を参照。

---

## Phase 1開発開始前の必須作業

### 1. OpenAPI契約の確認（MUST）
1. `eduanimaR_Professor/docs/openapi.yaml`が存在することを確認
2. 以下のエンドポイントが定義されていることを確認:
   - `POST /v1/auth/dev-login`
   - `POST /v1/qa/stream`
   - `GET /v1/subjects`
   - `GET /v1/subjects/{subject_id}/materials`

### 2. Orval設定の確認（MUST）
1. `orval.config.ts`が`eduanimaR_Professor/docs/openapi.yaml`を参照していることを確認
2. 生成コマンドを実行:
   ```bash
   npm run api:generate
   ```
3. `src/shared/api/generated/`に型定義・クライアントが生成されることを確認

### 3. 開発サーバー起動（Phase 1）
1. バックエンド（Professor）を起動:
   ```bash
   cd eduanimaR_Professor
   docker compose up -d
   ```

2. フロントエンドを起動:
   ```bash
   cd eduanimaR
   npm install
   npm run dev
   ```

3. `http://localhost:3000`にアクセスし、`dev-user`で自動認証されることを確認

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

### 技術スタック（確定）

本プロジェクトのChrome拡張機能は、以下の技術スタックで実装します:

| 技術要素 | 詳細 |
|---------|------|
| **Framework** | **Plasmo Framework**（Manifest V3対応、Reactベース） |
| **UI System** | MUI v6 + Pigment CSS（Shadow DOM隔離戦略） |
| **DOM検知** | **MutationObserver**（LMS資料の自動検知） |
| **拡張内通信** | **Plasmo Messaging**（Content Scripts ⇔ Background/Service Worker） |
| **外部通信** | Background/Service Workerから Professor API へHTTP（CORS制約なし） |
| **Service Worker前提** | 常駐しない設計（起動/停止の揺らぎを許容） |
| **認証** | Phase 1: dev-user固定、Phase 2: SSO（OAuth/OIDC）トークンをChrome Storageへ保存 |

**Shadow DOM隔離戦略**:
- LMSサイトのCSSと拡張機能のCSSが衝突しないよう、Shadow DOMで隔離
- MUI Pigment CSS をShadow Root内で適用

**MutationObserver設計**:
- LMS資料のDOM変更を監視し、新しい資料を自動検知
- 検知後、Professor API (`/v1/materials/upload`) へ自動送信

**参照**: 
- [`../../eduanimaRHandbook/02_strategy/TECHNICAL_STRATEGY.md`](../../eduanimaRHandbook/02_strategy/TECHNICAL_STRATEGY.md) L128-144
- [`../02_tech_stack/STACK.md`](../02_tech_stack/STACK.md)

### Phase 1実装範囲（詳細）

Phase 1では、以下を**完全実装**します:

1. **FABメニュー統合**:
   - MoodleのFABメニュー（PENアイコン）を検出
   - メニュー内に「AI質問」アイテムを追加（DOM操作）
   - クリック時にサイドパネルをトグル表示
   - **完了条件**: FABメニューから「AI質問」を起動でき、トグル動作が機能する

2. **サイドパネル実装**:
   - Plasmo CSUIでReactコンポーネントをマウント
   - 画面右端に固定（position: fixed、幅400px、高さ100vh）
   - 開閉アニメーション（transform: translateX、0.3秒）
   - 閉じるボタン（「>」ボタン、パネル左端）
   - **完了条件**: サイドパネルが正常に表示され、開閉アニメーションが動作する

3. **状態永続化**:
   - sessionStorageに以下を保存
     - パネル開閉状態（isOpen）
     - スクロール位置（scrollPosition）
     - 会話履歴（conversationHistory）
   - ページリロード時に状態復元
   - **完了条件**: ページ遷移後も状態が維持される

4. **ページ遷移対応**:
   - 通常遷移（ページ全体リロード）: Content Script再実行 → 状態復元
   - SPAナビゲーション（Turbo等）: `turbo:load`イベントリスナー → 状態維持
   - DOM再構築: MutationObserverでFABメニュー再検出 → アイテム再挿入
   - **完了条件**: すべての遷移パターンで状態が維持される

5. **LMS資料の自動検知**:
   - MutationObserverでLMS DOMを監視
   - 資料ダウンロードリンク・PDFファイル名を抽出
   - 検知した資料をローカルストレージに一時保存
   - **完了条件**: LMSページで新しい資料が追加されたときに自動検知される

6. **自動アップロード**:
   - Plasmo Messagingで Content Scripts → Background/Service Worker へファイル送信
   - Background/Service Worker から Professor API (`POST /v1/materials/upload`) へ送信
   - アップロード状態（成功/失敗/進行中）をUIに表示
   - **完了条件**: 資料が自動アップロードされ、状態がUI表示される

7. **質問機能（QAチャット）**:
   - サイドパネル内で質問入力欄を表示
   - Professor API (`POST /v1/qa/ask`、SSE) へ質問送信
   - SSEイベント（thinking/searching/evidence/answer）をリアルタイム表示
   - エビデンスカード表示（クリッカブルURL、why_relevant、snippets）
   - **完了条件**: 質問を送信でき、SSEイベントがリアルタイム表示される

8. **ローカル動作検証**:
   - `npm run build:extension` でビルド
   - Chromeに手動読み込み（`chrome://extensions/` → 「デベロッパーモード」→「パッケージ化されていない拡張機能を読み込む」）
   - ローカルProfessor API（`http://localhost:8080`）に接続して動作確認
   - **検証項目**:
     - FABメニューから「AI質問」を起動できる
     - サイドパネルが正常に表示される
     - ページ遷移後も状態が維持される
     - 質問を送信でき、SSEイベントが表示される

**Phase 1で実装しないこと**（Phase 2以降）:
- ❌ SSO認証（Phase 1は`dev-user`固定）
- ❌ Chrome Web Storeへの公開
- ❌ 本番環境へのデプロイ
- ❌ パネルのリサイズ機能
- ❌ フォールバック（独立ボタン等）

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
