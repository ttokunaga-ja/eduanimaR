# TEST_PYRAMID

## 目的
品質保証の投資配分を固定し、開発速度と信頼性を両立する。

## ピラミッド
1. Unit（最優先）
- domain/usecase のテスト
- 外部I/Oはinterfaceでモック化

2. Integration（重要）
- outbound client（Professor / Gemini）の呼び出しを含めて検証
	- 成功/失敗/タイムアウト
	- リトライの境界（ネットワーク一時失敗のみ等）
	- エラーマッピング（依存障害を 503 等へ）
- OpenAPI（Professor↔Librarian）の契約整合（破壊的変更の検知）

3. HTTP（必要最小）
- handler をテストクライアントで検証（入力バリデーション / レスポンス形状 / エラー形式）
- usecase はモック注入し、transport 層の責務に集中させる

## 禁止/注意
- 1テストで多数の責務を検証しない（失敗時の原因特定が難しい）
- flake 対策としてタイムアウト/リトライはテスト側で明確に扱う

## 本サービスで扱わないもの
- DB マイグレーション/リポジトリのテスト（Librarian は DB-less）
- HTTP/JSON 以外の内部 RPC 方式の性能/互換性テスト（Librarian の SSOT は HTTP/JSON）
