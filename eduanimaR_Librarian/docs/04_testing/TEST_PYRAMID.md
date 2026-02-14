# TEST_PYRAMID

## 目的
品質保証の投資配分を固定し、開発速度と信頼性を両立する。

## ピラミッド
1. Unit（最優先）
- domain/usecase のテスト
- 外部I/Oはinterfaceでモック化

2. Integration（重要）
- PostgreSQL を Testcontainers v0.40.1 で起動し、repository/sqlc を実検証
- マイグレーション（Atlas）を適用してからテストする

3. HTTP（必要最小）
- handler は `httptest` で検証
- usecase はモック注入

## 禁止/注意
- 1テストで多数の責務を検証しない（失敗時の原因特定が難しい）
- flake 対策としてタイムアウト/リトライはテスト側で明確に扱う
