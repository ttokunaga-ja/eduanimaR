# ERROR_CODES

## 目的
consumer（例: フロントエンド）が分岐できる安定したエラーコード体系を定義する。

## ルール
- `code` は機械可読な安定ID（破壊的変更を避ける）
- `message` は人間向け（内部情報は含めない）
- ドメイン別にプレフィックスを揃える（例: `USER_`, `ORDER_`）
- **フロントエンド側で `code` による分岐が必要** な場合、必ず OpenAPI の `responses` に明記する

## 共通レスポンス形式（確定）
```json
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "User not found",
    "details": {
      "field": "user_id"
    },
    "request_id": "abc-123-def"
  }
}
```

## 標準エラーコード一覧
| code | HTTP | 意味 | フロントエンド対応例 |
| --- | --- | --- | --- |
| `VALIDATION_FAILED` | 400 | 入力が不正 | フォームエラーを表示（Zod） |
| `UNAUTHORIZED` | 401 | 認証なし/無効 | ログイン画面へリダイレクト |
| `FORBIDDEN` | 403 | 権限なし | 権限不足のメッセージ表示 |
| `NOT_FOUND` | 404 | リソース無し | 404ページ表示 |
| `CONFLICT` | 409 | 競合（重複/状態不整合） | 再試行を促す |
| `RATE_LIMITED` | 429 | レート制限 | リトライ待機時間を表示 |
| `INTERNAL` | 500 | 想定外エラー | 汎用エラーページ |
| `DEPENDENCY_UNAVAILABLE` | 503 | 依存サービス障害 | メンテナンス中表示 |

## フロントエンドとの同期
- Orval で生成された型には、OpenAPI で定義されたエラーレスポンスの型も含まれる
- フロントエンド側は生成された型で `error.code` を判定し、適切な UI を出す

