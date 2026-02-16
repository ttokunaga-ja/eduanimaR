---
Title: Slices Map
Description: eduanimaRのFSD機能一覧と配置マップ
Owner: @ttokunaga-ja
Status: Published
Last-updated: 2026-02-16
Tags: frontend, eduanimaR, fsd, architecture, slices
---

# Slices Map (機能一覧と配置)

Last-updated: 2026-02-16

このドキュメントは「どの機能がどの slice に存在するか」を定義し、AI が勝手に新規 slice を乱立させないためのマップです。

## ルール
- 新しい slice を作る前に、このファイルに追記してから実装する
- slice 名は英小文字のケバブケース（例: `qa-chat`）
- `features` 内の slice 同士は直接 import しない（合成は `widgets` / `pages`）

## 命名・粒度のガイド
- **slice = ドメイン/ユースケースの境界**。UI都合（`components/`）では切らない
- 小さすぎるslice（ボタン1個等）は避ける → `shared/ui` か既存sliceへ
- 大きすぎるslice（アプリ全体の状態など）は避ける → `app/providers` に寄せるか分割

## 追加フロー（必須）
1. このファイルに slice を追記（理由/責務を1行で書く）
2. 依存ルール（layers / isolation / public API）に違反しない構造を先に決める
3. 実装（外部公開は `index.ts` のみ）

## Slices（eduanimaR 固有）

### entities
- **`subject`**: 科目（Professor の `subject_id` に対応）
  - **責務**: 科目名・期間・担当教員の表示
  - **依存**: `shared/api`（Professor の `/subjects` エンドポイント）
  - **バックエンド境界**: Professor の `subject` テーブル

- **`file`**: 資料ファイル（Professor の GCS URL / metadata に対応）
  - **責務**: ファイル一覧・サムネイル・メタデータ表示
  - **依存**: `shared/api`（Professor の `/files` エンドポイント）
  - **バックエンド境界**: Professor の `file` テーブル + GCS

- **`evidence`**: 選定エビデンス（Librarianが選定した根拠箇所）
  - **責務**: Librarianが選定したエビデンス（根拠箇所）の表示・管理
  - **属性**: 
    - `temp_index`: LLM可視の一時参照ID（Professorが`document_id`に変換）
    - `document_id`: 安定したドキュメント識別子
    - `snippets`: Markdown形式の断片
    - `why_relevant`: 選定理由（なぜこのエビデンスが選ばれたか）
  - **依存**: `shared/api`（Professor経由でLibrarian結果を取得）
  - **バックエンド境界**: Librarianの`selected_evidence`出力

- **`user`**: ユーザー（SSO/OAuth で取得したプロフィール）
  - **責務**: ユーザー名・アイコン表示
  - **依存**: `shared/api`（Professor の `/me` エンドポイント）
  - **バックエンド境界**: Professor の `user` テーブル

- **`session`**: セッション（ログイン状態の表現）
  - **責務**: 認証状態の管理、トークンリフレッシュ
  - **依存**: `shared/api`（Professor の `/auth/refresh` エンドポイント）
  - **バックエンド境界**: Cookie ベースのセッション（Phase 2以降）

### features
- **`qa-chat`**: Q&A（Professor の SSE + Librarian推論結果）
  - **責務**: 
    - 質問入力、リアルタイム回答表示
    - Professor SSE経由でのリアルタイム回答配信
    - 選定エビデンス（Librarianが選定した根拠箇所）の表示
    - ソース（参照箇所）のクリッカブルリンク
  - **依存**: `shared/api`（Professor の `/v1/search` エンドポイント、SSE）、`entities/evidence`、`entities/file`
  - **バックエンド境界**: Professor SSE + Librarian推論ループ結果（Professor ↔ Librarian間はgRPC）

- **`auth-by-token`**: トークンでの再認証/更新
  - **責務**: リフレッシュトークンによるセッション延長
  - **依存**: `entities/session`
  - **バックエンド境界**: Professor の `/auth/refresh` エンドポイント

- **`chrome-extension-bridge`**: 拡張機能連携（Phase 3以降）
  - **責務**: Chrome拡張機能からのSSO認証結果受け取り、自動アップロード状態の表示
  - **依存**: `shared/api`（Professor の認証エンドポイント）
  - **バックエンド境界**: Professor（SSO検証、ユーザー登録、科目同期）

### widgets
- **`file-tree`**: 科目別ファイルツリー表示
  - **責務**: `entities/subject` + `entities/file` の合成、折りたたみ UI
  - **依存**: `entities/subject`, `entities/file`

- **`search-loop-status`**: Librarian推論ループ進行状況表示（Phase 3以降）
  - **責務**: Librarian推論ループの進行状況表示（現在の試行回数、停止条件達成状況）
  - **依存**: `features/qa-chat`、`entities/evidence`
  - **バックエンド境界**: Librarianの`status`フィールド（`SEARCHING` / `COMPLETED` / `ERROR`）

- **`qa-panel`**: Q&A パネル（履歴 + 入力 + 回答表示）
  - **責務**: `features/qa-chat` + 質問履歴の合成
  - **依存**: `features/qa-chat`, `entities/user`

- **`app-header`**: ヘッダー（ナビゲーション + ユーザーメニュー）
  - **責務**: `entities/user` + `features/auth` の合成
  - **依存**: `entities/user`, `features/auth-by-token`

### pages
- **`dashboard`**: ダッシュボード（科目一覧、最近の質問、アップロード状況）
  - **責務**: `widgets/file-tree` + `widgets/qa-panel` の配置
  - **依存**: `widgets/*`, `features/*`

- **`settings`**: 設定画面（通知設定、アカウント情報）
  - **責務**: ユーザー設定の CRUD
  - **依存**: `entities/user`, `shared/api`

### shared
- `shared/ui`：原子UI（Button, TextField 等）、MUI v6 + Pigment CSS のラッパー
- `shared/api`：Orval 生成物と API 設定（Professor の OpenAPI から生成）
- `shared/lib`：汎用ユーティリティ（ビジネスロジック禁止）

---

## eduanimaR 固有のSlices定義

### `qa` (Q&A機能)
- **責務**: 質問と回答、チャット形式UI
- **Professor API**:
  - `POST /v1/qa/ask` - 質問送信
  - `GET /v1/qa/stream` - SSE回答ストリーミング
  - `GET /v1/qa/history` - 質問履歴取得
- **依存**: `entities/question`, `entities/answer`, `shared/api`
- **UI**: チャットメッセージリスト、入力フォーム、参照資料リンク

### `materials` (資料管理)
- **責務**: 科目・資料の一覧、検索、ツリー表示
- **Professor API**:
  - `GET /v1/materials` - 資料一覧取得
  - `GET /v1/materials/:id` - 資料詳細取得
  - `POST /v1/materials/ingest` - 資料取り込み（Chrome拡張）
- **依存**: `entities/material`, `entities/subject`, `shared/api`
- **UI**: 科目別ツリー表示、検索フィルタ、お気に入り登録

### `study-plan` (学習計画・履歴)
- **責務**: 学習計画作成、進捗管理、履歴表示
- **Professor API**:
  - `GET /v1/study-plan` - 学習計画取得
  - `POST /v1/study-plan` - 学習計画作成
  - `PATCH /v1/study-plan/:id` - 進捗更新
- **依存**: `entities/study-plan`, `shared/api`
- **UI**: カレンダー、タスクリスト、進捗グラフ

### `auth` (認証フロー)
- **`auth`**: 認証・セッション管理・**拡張機能誘導**
  - **責務**: SSO認証、セッション取得、ログアウト、未登録ユーザーの拡張機能誘導
  - **依存**: `shared/api`（Professor `/auth/*` エンドポイント）
  - **バックエンド境界**: Professor `auth` ドメイン
  - **Phase 2追加機能**:
    - 未登録ユーザーの検知（`AUTH_USER_NOT_REGISTERED`）
    - 拡張機能誘導画面（`ExtensionInstallPrompt`）
    - 誘導先URL管理（`shared/config/extension-urls`）
  - **主要コンポーネント**:
    - `features/auth/ui/LoginForm.tsx`
    - `features/auth/ui/ExtensionInstallPrompt.tsx`（Phase 2追加）
    - `features/auth/api/useLogin.ts`
    - `app/auth/register-redirect/page.tsx`（Phase 2追加）
- **Professor API**:
  - `POST /v1/auth/verify` - トークン検証
  - `POST /v1/auth/refresh` - トークン更新
  - `POST /v1/auth/logout` - ログアウト
- **依存**: `shared/api`, NextAuth.js
- **UI**: ログインボタン、プロバイダー選択画面

## Slices間の依存方向（FSD原則）

```
app (pages/layouts)
  ↓
features (qa, materials, study-plan, auth)
  ↓
entities (question, answer, material, subject, study-plan)
  ↓
shared (api, ui, lib)
```

- **禁止**: 下位層から上位層への依存
- **推奨**: Slices間は直接依存せず、shared/apiを経由

---

## slice 追記テンプレ
追記するときは、最低限これだけ埋めます。

- `layers/<slice>`：何を表すか（1行）
- 依存（使うentity/shared）：
- 外部公開（`index.ts` でexportするもの）：

## 追加・変更のチェックリスト
- 既存 slice で表現できない理由は明確か
- 依存方向（layers）に違反していないか
- Public API（`index.ts`）経由の import になっているか
