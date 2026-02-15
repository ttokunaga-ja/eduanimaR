# Slices Map (機能一覧と配置)

Last-updated: 2026-02-15

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

- **`user`**: ユーザー（SSO/OAuth で取得したプロフィール）
  - **責務**: ユーザー名・アイコン表示
  - **依存**: `shared/api`（Professor の `/me` エンドポイント）
  - **バックエンド境界**: Professor の `user` テーブル

- **`session`**: セッション（ログイン状態の表現）
  - **責務**: 認証状態の管理、トークンリフレッシュ
  - **依存**: `shared/api`（Professor の `/auth/refresh` エンドポイント）
  - **バックエンド境界**: Cookie ベースのセッション（Phase 2以降）

### features
- **`qa-chat`**: Q&A（Professor の SSE + Librarian Agent の推論結果）
  - **責務**: 質問入力、リアルタイム回答表示、ソース（参照箇所）のクリッカブルリンク
  - **依存**: `shared/api`（Professor の `/qa/stream` SSE）、`entities/file`
  - **バックエンド境界**: Professor（SSE配信）↔ Librarian（gRPC、検索戦略立案）

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

## slice 追記テンプレ
追記するときは、最低限これだけ埋めます。

- `layers/<slice>`：何を表すか（1行）
- 依存（使うentity/shared）：
- 外部公開（`index.ts` でexportするもの）：

## 追加・変更のチェックリスト
- 既存 slice で表現できない理由は明確か
- 依存方向（layers）に違反していないか
- Public API（`index.ts`）経由の import になっているか
