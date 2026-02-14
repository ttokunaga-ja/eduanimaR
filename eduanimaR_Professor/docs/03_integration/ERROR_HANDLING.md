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

## gRPC（内部）との整合
- 内部 gRPC は status code を必ず返す（成功=OK、失敗=適切な code）
- 外部（HTTP/OpenAPI）に返す `error.code`（アプリケーションコード）は **安定ID** として扱い、gRPC/HTTP の両方で同一のコード体系を用いる
- Gateway は gRPC の status code / details を、外部の HTTP ステータス + 共通レスポンス形式へ変換する
  - 例: gRPC `NOT_FOUND` → HTTP 404
  - 例: gRPC `INVALID_ARGUMENT` → HTTP 400

> 補足: gRPC status とアプリケーション `error.code` は別物。
> status は「通信/一般分類」、`error.code` は「業務的な安定識別子」。

## 運用ルール
- `code` は安定IDとして扱い、破壊的変更を避ける
- `message` は利用者向け（内部情報を漏らさない）
- `request_id` は必ず付与する
