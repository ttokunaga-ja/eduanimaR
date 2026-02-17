# Project Decisions（SSOT）

このファイルは「プロジェクトごとに選択が必要」な決定事項の SSOT。
AI/人間が推測で穴埋めしないために、まずここを埋めてから実装する。

## 基本
- サービス名（固定）: `eduanima-librarian`
- 環境: local / staging / production
- Professor 側のサービス名/エンドポイント:
  - サービス名: `eduanima-professor`
  - Phase 1（ローカル）: `localhost:50051`（gRPC）
  - Phase 2以降: `professor.internal:50051`（Cloud Run / VPC内部通信）

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
- **Professor ↔ Librarian（gRPC / Protocol Buffers）**:
  - **SSOT（契約の正）**: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`
  - RPC: `LibrarianService.Reason`（双方向ストリーミング）
  - HTTP/JSON（`openapi.librarian.yaml`）は参考定義。実装では proto を使うこと
- 互換性ポリシー（`API_VERSIONING_DEPRECATION.md` と整合）:
  - フィールド追加は互換、フィールド削除/番号変更は破壊的（proto のルールを遵守）

## データ（Must）
- Librarian は DB を持たない（DB-less）
- 出力データ（Evidence）の最小フィールド: `temp_index`（Professor が安定参照へ変換できること）
- PII/機密情報の扱い（ログ含む）:
  - ユーザーの質問文（`user_query`）は推論時に LLM へ渡すが、Librarian 内で永続化しない
  - 構造化ログでは `user_query` / `analyzed_info` フィールドを除外（またはマスク）する
  - Librarian から外部 LLM API へのリクエストには質問テキストが含まれるため、API プロバイダーのデータ処理方針に準拠する

## 運用（Must）
- 観測性（ログ/メトリクス/トレース）導入範囲:
  - **ログ**: 構造化 JSON ログ（OpenTelemetry Log SDK）。`user_query` はマスク必須
  - **メトリクス**: Prometheus scrape 対応（`/metrics`）。推論ループ回数・レイテンシ・エラー率を計測
  - **トレース**: OTEL Trace（`request_id` を全スパンに伝播）。Professor ↔ Librarian gRPC スパンを結合
- SLO（対象導線/指標/アラート閾値）:
  - 推論ループ応答レイテンシ p95 ≤ 3秒（Phase 1 ローカル計測）
  - エラー率（gRPC INTERNAL / DEADLINE_EXCEEDED） ≤ 1%
  - `max_retries` 未達による PARTIAL_RESULT 率 ≤ 10%
- Secrets 管理（どこで管理し、どう配るか）:
  - Phase 1: ローカル `.env` ファイル（`.gitignore` 必須）。`GEMINI_API_KEY` 等
  - Phase 2以降: GCP Secret Manager 経由（Cloud Run のサービスアカウントに参照権限付与）

## 配信（Must）
- デプロイ先:
  - Phase 1: Docker Compose（`docker-compose.yml` 内 `librarian` サービスとして定義）
  - Phase 2以降: Google Cloud Run（`us-central1`、min-instances=0、max-instances=3）
- ロールバック方針（Librarian は DB 変更を伴わない前提での運用方針）:
  - DB 変更なし → 前バージョンのコンテナイメージへ即時ロールバック可能（Cloud Run の traffic split で 100% 切り替え）
  - ロールバック手順: `gcloud run services update-traffic librarian --to-revisions PREV_REVISION=100`
  - Phase 1: Docker Compose を前イメージで `docker-compose up -d` するだけで完了
