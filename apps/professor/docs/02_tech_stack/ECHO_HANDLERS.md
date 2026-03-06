# ECHO_HANDLERS

## 目的
Echo v5.0.1 のハンドラー実装を、OpenAPI駆動（Contract First）で統一する。

## 原則
- ルーティング/入出力スキーマは OpenAPI が正
- 生成コードの手編集は禁止

> SSE は OpenAPI で表現しづらい場合があるため、SSE導線は OpenAPI と併せて「ストリーミングのイベント仕様（別ドキュメント）」を SSOT として扱う。

## ハンドラーの責務
- 入力: バリデーション、認証/認可、リクエストDTO → domain/usecase への変換
- 出力: usecase 結果 → レスポンスDTO、エラーマッピング
- 禁止: ビジネスロジックの実装、直接DBアクセス

## エラーハンドリング
- handler は domain/usecase のエラーを受け取り、共通レスポンス形式に変換する
- 例: `ERROR_HANDLING.md` のマッピング表に従う

## タイムアウト/キャンセル
- `context.Context` を必ず usecase/repository に伝播する
- 外部I/O（DB/GCS/Kafka/HTTP/gRPC/Gemini）は deadline を尊重する

## SSE（MUST）
- 進捗/引用/回答を段階的に送る（クライアントは再接続を想定）
- クライアント切断時にサーバ側処理を止められる箇所は止める（context cancel）
- イベントは重複してもUIが破綻しない形式にする
