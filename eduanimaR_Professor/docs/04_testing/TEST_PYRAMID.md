# TEST_PYRAMID

## 目的
品質保証の投資配分を固定し、開発速度と信頼性を両立する。

## ピラミッド
1. Unit（最優先）
- domain/usecase のテスト
- 外部I/Oはinterfaceでモック化

重要:
- Librarian（gRPC）や Kafka/DB/GCS などの外部I/Oは、usecase の境界で mock/stub 化する
- キャンセル（context）と deadline の伝播を unit でも検証する（漏れやすい）

2. Integration（重要）
- PostgreSQL を Testcontainers v0.40.1 で起動し、repository/sqlc を実検証
- マイグレーション（Atlas）を適用してからテストする

追加で重要:
- Kafka worker のハンドラ（consume→永続化）の冪等性を検証する（重複イベント/再実行）
- 依存（gRPC/Kafka/DB）障害時のエラー分類が期待通りか確認する

3. Transport（必要最小）
- HTTP handler は `httptest` で検証する
- SSE は「接続→イベント受信→切断/キャンセル」の最小シナリオを検証する
- gRPC は Professor 側クライアントの deadline/エラーマッピングを重点的に検証する（Librarian実装の詳細は対象外）

## 禁止/注意
- 1テストで多数の責務を検証しない（失敗時の原因特定が難しい）
- flake 対策としてタイムアウト/リトライはテスト側で明確に扱う

## 契約（補足）
- OpenAPI / Proto の互換性検査は `03_integration/CONTRACT_TESTING.md` を正とする（lint / breaking / 差分検知）
