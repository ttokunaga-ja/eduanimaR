# STACK

## 技術スタック（2026年2月最新版）

| 項目 | バージョン | リリース日 | 備考・新機能要約 |
| :--- | :--- | :--- | :--- |
| **Go** | **1.25.7** | 2026/02/04 | 最新安定版。セキュリティ修正およびコンパイラ最適化を含む。 |
| **gRPC (google.golang.org/grpc)** | **latest** | - | 内部マイクロサービス間通信の標準プロトコル。Protocol Buffers(.proto)で型安全な契約。 |
| **grpc-gateway** | **v2.x** | - | gRPCサービスをHTTP/JSON APIとして公開。Gateway層でgRPC→OpenAPI変換を実行。 |
| **PostgreSQL** | **18.1** | 2025/10頃~ | `uuidv7()`、非同期I/O (AIO)、B-tree Skip Scanの正式サポート。 |
| **pgvector** | **0.8.1** | 2025/09/04 | HNSWインデックスの構築・検索パフォーマンス向上。反復インデックススキャン対応。 |
| **Atlas** | **v1.0.0** | 2025/12/24 | メジャーリリース到達。Monitoring as Code、Schema Statistics機能追加。 |
| **sqlc** | **1.30.0** | 2025/09/01 | pgx/v5、ENUM配列の対応強化。MySQL/SQLiteエンジンの改善。 |
| **pgx** | **v5.8.0** | 2025/12/26 | Go 1.24+必須化。パイプライン処理の改善、`pgtype.Numeric`の最適化。 |
| **Echo** | **v5.0.1** | 2026/01/28 | v5が正式リリース。Gateway層のHTTP処理に使用。エラーハンドリングの刷新、ルーターの最適化。 |
| **Elasticsearch** | **9.2.4** | - | ベクトル検索統合（dense_vector）、Qdrantを完全置換。 |
| **Debezium CDC** | - | - | PostgreSQL論理レプリケーションから移行、Kafka経由のリアルタイム差分同期。 |
| **slog** | - | - | Goの標準ライブラリ |
| **Testcontainers** | **v0.40.1** | 2025/11/06 | PostgresSQLにてSSL設定（WithSSLSettings）の簡略化、証明書の自動マウントと設定対応 |

## 設計ポリシー
- **UUID + NanoID**: 内部主キーはUUID（推奨: UUIDv7）、外部公開キーはNanoIDを採用し、セキュリティとユーザビリティを両立する。
- **ENUM型の積極採用**: 固定値の管理はPostgreSQL ENUM型を使用し、型安全性・パフォーマンス・可読性を向上させる。
	- **VARCHAR型での固定値管理は禁止**（必ず PostgreSQL ENUM を使う）

## SSOT（Single Source of Truth）
- DBスキーマ: Atlas の `schema.hcl`
- API契約: OpenAPI（`docs/openapi.yaml` を置く場合はそれを正にする）
- SQL: `sql/queries/*.sql`（sqlcで生成）

