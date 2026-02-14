# Project Decisions（SSOT）

このファイルは「プロジェクトごとに選択が必要」な決定事項の SSOT。
AI/人間が推測で穴埋めしないために、まずここを埋めてから実装する。

## 基本
- サービス名（固定）: `eduanima-librarian`
- 環境: local / staging / production
- Professor 側のサービス名/エンドポイント: （例: `eduanima-professor` / base URL）

## サービス境界（Must）
- Librarian の責務（何をする/しない）:
	- する: 検索戦略、検索ループ制御、停止判断、エビデンス選定
	- しない: DB/インデックス管理、バッチ処理、最終回答文生成
- Professor の責務（Librarian 観点での境界）:
	- 検索の物理実行、ドキュメント管理、永続化、最終回答生成
- 依存関係（禁止依存を含む）:
	- Librarian → Professor（検索ツール呼び出し）
	- Professor → Librarian（推論委譲）
	- Librarian → DB/Index（禁止）

## 契約（Must）
- Professor ↔ Librarian（HTTP/JSON）:
	- OpenAPI の SSOT 置き場: `03_integration/`（方針は `API_CONTRACT_WORKFLOW.md` を正）
	- エンドポイント: `POST /v1/librarian/search-agent`
- 互換性ポリシー（`API_VERSIONING_DEPRECATION.md` と整合）:
	- 追加は原則互換、削除/必須化/型変更は破壊的

## データ（Must）
- Librarian は DB を持たない（DB-less）
- 出力データ（Evidence）の最小フィールド: `temp_index`（Professor が安定参照へ変換できること）
- PII/機密情報の扱い（ログ含む）: （方針をここに記載）

## 運用（Must）
- 観測性（ログ/メトリクス/トレース）導入範囲：
- SLO（対象導線/指標/アラート閾値）：
- Secrets 管理（どこで管理し、どう配るか）：

## 配信（Must）
- デプロイ先（例: Cloud Run 等）:
- ロールバック方針（Librarian は DB 変更を伴わない前提での運用方針）:
