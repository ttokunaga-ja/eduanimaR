# ERROR_HANDLING

## 目的
サービス共通のエラー型とHTTPレスポンス形式を定義し、クライアント実装と運用（監視/調査）を容易にする。

## 共通レスポンス形式（例）
```json
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "User not found",
    "details": {
      "field": "user_id"
    },
    "request_id": "..."
  }
}
```

## マッピング方針
- domain/usecase は「意味」を持つ型付きエラーを返す
- handler はエラーを以下にマッピングする
  - 4xx: クライアント起因（入力不正/権限/リソース無し）
  - 5xx: サーバ起因（依存障害/予期せぬ例外）

## 依存障害の扱い（Librarian特有）
- Gemini API の失敗/タイムアウトは `DEPENDENCY_UNAVAILABLE`（503）を基本とする
- Professor 側検索ツールの失敗も同様に 503 を基本とする

> ただし、入力が原因の失敗（例: 必須フィールド欠落）は 400（`VALIDATION_FAILED`）。

## 運用ルール
- `code` は安定IDとして扱い、破壊的変更を避ける
- `message` は利用者向け（内部情報を漏らさない）
- `request_id` は必ず付与する
