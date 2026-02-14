# Slices Map (機能一覧と配置)

このドキュメントは「どの機能がどの slice に存在するか」を定義し、AI が勝手に新規 slice を乱立させないためのマップです。

## ルール
- 新しい slice を作る前に、このファイルに追記してから実装する
- slice 名は英小文字のケバブケース（例: `auth-by-email`）
- `features` 内の slice 同士は直接 import しない（合成は `widgets` / `pages`）

## 命名・粒度のガイド
- **slice = ドメイン/ユースケースの境界**。UI都合（`components/`）では切らない
- 小さすぎるslice（ボタン1個等）は避ける → `shared/ui` か既存sliceへ
- 大きすぎるslice（アプリ全体の状態など）は避ける → `app/providers` に寄せるか分割

## 追加フロー（必須）
1. このファイルに slice を追記（理由/責務を1行で書く）
2. 依存ルール（layers / isolation / public API）に違反しない構造を先に決める
3. 実装（外部公開は `index.ts` のみ）

## Slices

### pages
- (例) `chat-workspace`：チャット（アプリシェル）
- (例) `file-management-center`：資料管理センター
- (例) `history-archive`：履歴アーカイブ

### widgets
- (例) `app-header`：ヘッダー（複数feature/entityを合成）
- (例) `app-sidebar`：ナビゲーション
- (例) `post-card`：投稿カード（entity + feature の合成点）

### features
- (例) `auth-by-email`：ログイン
- (例) `auth-by-token`：トークンでの再認証/更新
- (例) `article-rating`：評価操作

### entities
- (例) `user`：ユーザー表示/モデル
- (例) `session`：セッション（ログイン状態の表現）
- (例) `article`：記事表示/モデル

### shared
- `shared/ui`：原子UI（Button, TextField 等）
- `shared/api`：Orval 生成物と API 設定
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
