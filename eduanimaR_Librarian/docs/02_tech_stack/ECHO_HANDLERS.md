# ECHO_HANDLERS

## 目的
Echo v5.0.1 のハンドラー実装を、OpenAPI駆動（Contract First）で統一する。

## 原則
- ルーティング/入出力スキーマは OpenAPI が正
- 生成コードの手編集は禁止

## ハンドラーの責務
- 入力: バリデーション、認証/認可、リクエストDTO → domain/usecase への変換
- 出力: usecase 結果 → レスポンスDTO、エラーマッピング
- 禁止: ビジネスロジックの実装、直接DBアクセス

## エラーハンドリング
- handler は domain/usecase のエラーを受け取り、共通レスポンス形式に変換する
- 例: `ERROR_HANDLING.md` のマッピング表に従う

## タイムアウト/キャンセル
- `context.Context` を必ず usecase/repository に伝播する
- 外部I/O（DB/ES/HTTP）は deadline を尊重する
