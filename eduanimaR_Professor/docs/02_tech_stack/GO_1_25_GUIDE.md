# GO_1_25_GUIDE

## 目的
Go 1.25.7 の機能を活かしつつ、読みやすさ・型安全性・運用性を損なわない利用基準を定義する。

## 基本方針
- 可読性が最優先（賢いワンライナーより、意図が伝わるコード）
- 標準ライブラリ優先（`log/slog`, `errors` 等）

## context（MUST）
Professor は HTTP（OpenAPI）/SSE と gRPC（Librarian）をまたぐため、`context.Context` の扱いを統一する。

- handler/transport で受けた `context` を usecase → repository/adapter へ必ず伝播する
- gRPC クライアント呼び出しは deadline 必須（無限待ち禁止）
- SSE はクライアント切断を前提にし、切断後に goroutine が残り続けないようにする

## エラー
- 原則: sentinel error よりも型付きエラー（domain error）を使う
- 複数エラーの合成は `errors.Join` を使う（独自multi-error禁止）

## イテレーション（range-over-func / iterators）
- 採用基準:
  - コレクション生成（中間slice）を避けたい
  - フィルタ/マップ/ストリーム処理の意図が明確
- 非採用基準:
  - チームの理解を超えて読みにくくなる
  - デバッグが著しく難しくなる

## ログ
- 構造化ログは `log/slog` を標準とする
- ルール:
  - request_id / trace_id / user_id（取得できる場合）を必ず付与
  - PII をログに出さない
