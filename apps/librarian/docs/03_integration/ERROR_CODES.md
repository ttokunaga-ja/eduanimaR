# ERROR_CODES

## 目的
Professor（Go）を含む呼び出し元が分岐できる、安定したエラーコード体系を定義する。

## ルール
- `code` は機械可読な安定ID（破壊的変更を避ける）
- `message` は人間向け（内部情報は含めない）
- ドメイン別にプレフィックスを揃える（例: `USER_`, `ORDER_`）
- 呼び出し元で `code` による分岐が必要な場合、必ず OpenAPI の `responses` に明記する

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
| code | HTTP | 意味 | 呼び出し元の対応例 |
| --- | --- | --- | --- |
| `VALIDATION_FAILED` | 400 | 入力が不正 | ログ/アラート（契約逸脱の可能性） |
| `UNAUTHORIZED` | 401 | 認証なし/無効 | 認証設定の再確認（内部通信） |
| `FORBIDDEN` | 403 | 権限なし | 呼び出し側の許可設定を確認 |
| `NOT_FOUND` | 404 | リソース無し | 依存する参照IDの誤りを疑う |
| `CONFLICT` | 409 | 競合（状態不整合） | リトライではなく状態確認 |
| `RATE_LIMITED` | 429 | レート制限 | バックオフ/キュー制御 |
| `INTERNAL` | 500 | 想定外エラー | 障害対応（request_id で追跡） |
| `DEPENDENCY_UNAVAILABLE` | 503 | 依存障害（Gemini/Professor側の一時障害等） | リトライ/フォールバック |

## 呼び出し元との同期
- `error.code` の追加/変更は破壊的になりやすい。原則として新規追加で進化させる。
- 変更時は `API_VERSIONING_DEPRECATION.md` に従う。

