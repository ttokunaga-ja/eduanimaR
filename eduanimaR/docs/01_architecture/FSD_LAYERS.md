# FSD Layers（運用ルール）

このドキュメントは、FSD（Feature-Sliced Design）における**レイヤーの責務**と、実装時に破りやすい**依存/配置ルール**を固定するための「契約」です。

- FSDの概要と判断基準： [FSD_OVERVIEW.md](./FSD_OVERVIEW.md)
- sliceの一覧（追加前に更新）： [SLICES_MAP.md](./SLICES_MAP.md)

---

## eduanimaR固有の責務境界（Must）

### フロントエンド ↔ バックエンドの責務分離

eduanimaRでは、以下の3層構成でシステムを構成します：

- **Frontend（Next.js + FSD）**: 
  - **責務**: 質問受付 + SSE受信 + エビデンス表示のみ
  - **禁止事項**: 検索戦略判断、Librarian直接通信、エビデンス選定ロジック
  
- **Professor（Go）**: 
  - **責務**: 認証・DB/GCS管理・検索戦略決定・最終回答生成
  - **権限**: PostgreSQL（pgvector含む）、GCS、Kafka への唯一の直接アクセス権限
  
- **Librarian（Python）**: 
  - **責務**: 検索クエリ生成・推論ループ制御（ステートレス）
  - **制約**: DB/GCS直接アクセス禁止（すべてProfessor経由）

### フロントエンドが実装してはならないこと（禁止事項）

以下はバックエンドの責務であり、フロントエンドで実装してはなりません：

- ❌ **Phase 2（大戦略/Planning）の実装**（Professor/Goの責務）
  - タスク分割（調査項目のリスト）の生成
  - 停止条件（Stop Conditions）の定義
  - Librarianへの初期パラメータの整理

- ❌ **検索戦略の判断ロジック**（Professor Phase 2の責務）
  - 「検索すべきか」「ヒアリングすべきか」の判断
  - 検索戦略（広範囲/精密/根拠探索）の決定

- ❌ **Phase 3（小戦略）の再試行制御**（Librarian/Pythonの責務）
  - 最大5回の再検索ループ（1回目: 直球、2回目: 補完、3回目: 類義語、4〜5回目: フォールバック）
  - 停止条件の満足判定（充足性・明確性・視覚情報の言語化）
  - 「収集完了」または「不足を宣言して終了」の判断
  
- ❌ **LibrarianとのgRPC通信**（Professorが仲介）
  - Librarianへのクエリリクエスト送信
  - Librarianからの検索結果受信
  
- ❌ **エビデンス選定ロジック**（Librarian Phase 3の責務）
  - `keyword_list` / `semantic_query` の生成
  - `evidence_snippets` の抽出・評価
  
- ❌ **ファイルアップロードUI**（Phase 1/2は拡張機能の自動アップロードのみ）
  - Phase 1: API直接呼び出し（curl/Postman） + 拡張機能実装
  - Phase 2: 拡張機能の自動アップロードのみ（Web版にUIを実装してはならない）

### フロントエンドが実装すべきこと（Must）

- ✅ **質問の受付と送信**: ユーザー入力を Professor API (`POST /v1/qa/ask`) へ送信
- ✅ **SSE受信と状態表示**: `thinking` / `searching` / `evidence` / `answer` イベントをリアルタイム表示
- ✅ **エビデンスの表示**: 
  - 根拠（資料名・ページ・抜粋）を主役に配置
  - クリッカブルなGCS署名付きURL + ページ番号
  - `why_relevant`（なぜこの箇所が選ばれたか）を明示
- ✅ **Chrome拡張機能の自動アップロード**: LMS資料の自動検知・アップロード（Phase 1で実装、Phase 2で本番適用）

### SSEイベントの種類と表示対応

| イベント | 意味 | フロントエンドのUI表示 |
|:---|:---|:---|
| `thinking` | Phase 2（大戦略）実行中 | 「検索戦略を立案中...」プログレスバー |
| `searching` | Phase 3（小戦略）実行中 | 「資料を検索中...（試行 X/5）」プログレスバー |
| `evidence` | エビデンス発見 | 資料カード表示（資料名・ページ・抜粋・why_relevant） |
| `answer` | Phase 4（最終回答）生成中 | 回答ストリーミング表示 |
| `complete` | 回答完了 | Good/Badフィードバックボタン表示 |
| `error` | エラー発生 | エラーコード別UI表示（`ERROR_CODES.md`参照） |

**重要**: 
- `error` イベントで `reason: insufficient_evidence` を受信した場合、自動リトライしない（ユーザー判断）。
- `complete` イベント受信後、回答の最後にGood/Badのフィードバックボタンを表示する。

### Professor/Librarianから受け取るデータ形式

**SSE `evidence` イベント**:
```json
{
  "event": "evidence",
  "data": {
    "file_id": "uuid",
    "subject_id": "uuid",
    "title": "資料名",
    "page_number": 12,
    "section": "第3章 統計的推定",
    "excerpt": "信頼区間の定義は...",
    "why_relevant": "なぜこの箇所が選ばれたか",
    "url": "gs://bucket/path/to/file.pdf"
  }
}
```

**SSE `answer` イベント**:
```json
{
  "event": "answer",
  "data": {
    "content": "回答本文（Markdown形式）",
    "citations": [
      {
        "file_id": "uuid",
        "page_number": 12,
        "excerpt": "引用箇所"
      }
    ]
  }
}
```

**SSE `error` イベント**:
```json
{
  "event": "error",
  "data": {
    "code": "4004",
    "reason": "insufficient_evidence",
    "message": "情報が不足しています。質問を具体化するか、資料を追加してください"
  }
}
```

**参照**: 
- [`../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md`](../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md)
- [`../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_LIBRARIAN.md`](../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_LIBRARIAN.md)
- [`../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`](../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md)

### Chrome拡張機能の責務（Phase 1から実装）

Chrome拡張機能は、以下の責務を持ちます：

#### ✅ 実装すべきこと

1. **UI統合（Content Script）**
   - Moodle FABメニュー（PENアイコン）への「AI質問」アイテム追加
   - FABメニュー検出: `document.querySelector('.float-button-menu')`
   - メニューアイテム挿入: DOM操作でリスト項目追加

2. **サイドパネル表示（Plasmo CSUI）**
   - Plasmo CSUIでReactコンポーネントをマウント
   - Shadow DOM隔離戦略でLMS CSSと衝突回避
   - サイドパネルの開閉制御（transform: translateX）

3. **状態永続化（sessionStorage）**
   - パネル開閉状態の保存・復元
   - 会話履歴の保存・復元
   - ページ遷移後の状態維持

4. **資料自動収集（MutationObserver）**
   - LMS資料の自動検知（PDF、スライド等）
   - Professor API (`POST /v1/materials/upload`) への自動送信
   - アップロード状態のUI表示

5. **認証管理（Phase 2）**
   - SSO認証トークンの取得・保存（Chrome Storage API）
   - トークン有効期限の管理
   - 認証エラー時の再認証フロー

#### ❌ 実装してはならないこと

1. **Moodleの既存DOMを破壊的に変更**
   - FABメニューの完全置き換え禁止
   - 既存メニューアイテムの削除禁止
   - Moodleのイベントハンドラを上書き禁止

2. **検索戦略判断・エビデンス選定ロジック**
   - これらはProfessor/Librarianの責務（バックエンド）
   - フロントエンドは「質問を投げてSSEで受け取る」のみ

3. **会話履歴の永続化（localStorage）**
   - Phase 1はsessionStorageのみ使用
   - 永続化はPhase 2以降（Professor側で管理）

#### バックエンドとの責務境界

- **Frontend（Chrome拡張）**: UI統合、SSE受信、エビデンス表示、状態永続化
- **Professor（Go）**: 検索戦略決定、DB/GCS管理、最終回答生成
- **Librarian（Python）**: 検索クエリ生成、推論ループ制御（ステートレス）

**参照**: 
- [`../../eduanimaRHandbook/02_strategy/TECHNICAL_STRATEGY.md`](../../eduanimaRHandbook/02_strategy/TECHNICAL_STRATEGY.md) L128-144
- [`../02_tech_stack/STACK.md`](../02_tech_stack/STACK.md)

---

## 1) 依存ルール（最重要）

### 1.1 レイヤー依存（単方向）
import は必ず上→下のみ：

`app` → `pages` → `widgets` → `features` → `entities` → `shared`

- ✅ 許可：`features/*` → `entities/*` / `shared/*`
- ❌ 禁止：`entities/*` → `features/*`

### 1.2 同一レイヤー内の Isolation
同一レイヤーの別sliceへ直接依存しません。

- ❌ 原則禁止：`features/a` → `features/b`
- 例外が必要なら（優先順）：
	1. 共通化できるものを `shared` へ移す
	2. 上位（`widgets` / `pages`）で合成する
	3. 設計自体（sliceの切り方）を見直す

---

## 2) Public API（`index.ts`）

各sliceはトップレベルに Public API を持ち、外部はそこからのみ import します。

- ✅ `import { UserCard } from '@/entities/user'`
- ❌ `import { UserCard } from '@/entities/user/ui/UserCard'`

Public API に公開するもの（目安）：
- `ui`：画面合成に必要なコンポーネント（例：`UserCard`）
- `model`：外部から利用される hook / actions（必要最小限）
- `api`：外部から叩く必要がある場合のみ（基本は `shared/api` に寄せる）

---

## 3) レイヤー別の責務（何を置くか）

### app（Application / ルート）
- **責務**：アプリ初期化、Provider、グローバル設定、エラー境界、ルーティングの殻
- **Next.js App Router 採用時**：`src/app` は App Router のディレクトリでもあるため、本テンプレではここを *appレイヤー* として扱います
- **置くもの**：
	- `layout.tsx`（Providers / global styles / metadata）
	- `providers/*`（QueryClientProvider、Theme、i18n 等）
	- `error.tsx` / `not-found.tsx`（必要な場合）
- **置かないもの**：画面固有の実装（ビジネスUIの本体）

### pages（画面の実体）
- **責務**：ルート（URL）に対応する画面を、widgets/features/entitiesで組み立てる
- **置くもの**：`ui/Page.tsx`（画面の合成）、ページ専用の薄い整形
- **置かないもの**：再利用前提の部品（再利用したいなら `widgets`/`features`/`entities`）

### widgets（独立したUIブロック）
- **責務**：複数feature/entityを合成する"塊"（例：ヘッダー、検索結果パネル）
- **置くもの**：レイアウトを含む UI ブロック、複合コンポーネント

### features（ユーザー価値の単位）
- **責務**：ユーザー操作 + ユースケース（例：ログイン、カート追加）
- **置くもの**：フォーム/操作UI、mutation、入力検証、成功/失敗の分岐
- **置かないもの**：アプリ全体の状態管理のハブ（features間依存を作りがち）

### entities（ビジネス実体）
- **責務**：ドメインオブジェクトの表現（表示・最小限の操作）
- **置くもの**：`UserCard` 等の表示、id→表示に必要な最小ロジック
- **置かないもの**：複数entity/featureをまたぐユースケース

### shared（共通基盤）
- **責務**：横断的に再利用される基盤（ビジネスルール禁止）
- **置くもの**：
	- `shared/ui`：UI primitives / wrappers
	- `shared/api`：OpenAPI生成物 + API共通設定
	- `shared/lib`：汎用関数
	- `shared/config`：環境変数、定数

---

## 4) Next.js での実装パターン（薄い adapter）

`src/app/**/page.tsx` はルーティングの"入口"で、原則 `src/pages/**/ui/Page` を import して描画するだけにします。

- 目的：FSDのページ実装を `pages` レイヤーへ集約し、ルーティング都合で構造が崩れるのを防ぐ

---

## 5) バックエンド（Professor）との責務対応

**通信プロトコル**:
- **Frontend ↔ Professor**: HTTP/JSON + SSE（OpenAPI契約）
- **Professor ↔ Librarian（内部）**: gRPC（双方向ストリーミング、フロントエンドからは不可視）

| Frontend (FSD) | Backend (Professor/Clean Arch) | 通信方法 |
| --- | --- | --- |
| `app` (routing) | `cmd/` (entry point) | - |
| `pages` (page composition) | `transport/http` (OpenAPI) | HTTP/JSON (Next.js BFF経由) |
| `features` (use cases) | `usecase` (business logic) | API契約（OpenAPI） |
| `entities` (business entities) | `domain` (entities) | 型生成（Orval） |
| `shared/api` (generated) | `repository` interface | - |
| `shared/ui` (MUI components) | （該当なし） | - |

### Professor OpenAPI 契約の具体例

#### SSE ストリーミング
- **エンドポイント**: `/v1/questions/{request_id}/events`
- **用途**: リアルタイム回答配信と進捗通知
- **実装要件**:
  - EventSource を使用したクライアント側接続管理
  - ストリーミング中のエラーイベント処理
  - 接続断時の再接続戦略（指数バックオフ）

#### エビデンス表示の要件
- **必須要素**: 回答には必ずソースを表示
  - クリッカブルな path/url（GCS 署名付き URL または内部 ID）
  - ページ番号（PDF の場合）またはセクション識別子
- **目的**: 参照元資料への即座の到達を可能にする
- **UI 要件**: ユーザーがワンクリックで原典の該当箇所を開けるリンク

### Frontend固有の注意（MUST）
- Frontend が直接呼ぶバックエンドは **Professor のみ**（OpenAPI）
- **Professor が検索の物理実行と最終回答生成を担当**
- **Librarian との通信は Professor 側で完結**（Frontend は関与しない）
  - Frontend → Professor のみ
  - Professor ↔ Librarian は内部通信（gRPC、Frontend 関与不可）
- 生成回答には必ず **Source を表示する**（クリック可能な path/url + ページ番号等）

### バックエンド詳細の参照先
- Professor（Go）の責務全体：`../../eduanimaR_Professor/docs/README.md`
- Professor の Clean Architecture：`../../eduanimaR_Professor/docs/01_architecture/CLEAN_ARCHITECTURE.md`
- バックエンド側の FSD 対応表：`../../eduanimaR_Professor/docs/01_architecture/FSD_LAYERS.md`

---

## 6) レビュー観点（チェックリスト）

- import が単方向（`app→...→shared`）になっている
- 同一レイヤー別sliceへの依存がない（特に `features→features`）
- deep import していない（Public API 経由）
- 置き場所が妥当（再利用するものを pages に閉じ込めていない）
- **バックエンドとの責務境界が明確**（Professor → Frontend の役割分担）

---

## 7) ルールの強制（ツール）

人手レビューだけでは破綻しやすいため、境界ルールはツールで強制します。

- ESLint：`eslint-plugin-boundaries` で layers / slices の境界違反を検知
- import パス：`@/*` を `src/*` に割り当て、import を正規化（相対パス地獄を避ける）

注意：ここはプロジェクトの ESLint/tsconfig に依存するため、導入時に "実際のディレクトリ構造" に合わせて設定を確定させる。