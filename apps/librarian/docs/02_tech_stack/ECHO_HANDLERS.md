# ECHO_HANDLERS（Not Applicable）

本サービスは Python（Litestar）で実装するため、Echo（Go）ハンドラー規約は適用外。

ハンドラーの責務（Librarian）:
- 入力: msgspec による検証、リクエストDTO → usecase 変換
- 出力: usecase 結果 → レスポンスDTO、エラーマッピング
- 禁止: ビジネスロジックの混入、DB接続
