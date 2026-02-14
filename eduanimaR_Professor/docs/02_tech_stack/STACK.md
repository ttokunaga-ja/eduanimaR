# STACK

## 技術スタック（2026年2月最新版）

| 項目 | バージョン | リリース日 | 備考・新機能要約 |
| :--- | :--- | :--- | :--- |
| **Go** | **1.25.7** | 2026/02/04 | 最新安定版。セキュリティ修正およびコンパイラ最適化を含む。 |
| **gRPC (google.golang.org/grpc)** | **latest** | - | **Professor ↔ Librarian** の内部通信。Protocol Buffers(.proto)で型安全な契約。 |
| **PostgreSQL** | **18.1** | 2025/10頃~ | `uuidv7()`、非同期I/O (AIO)、B-tree Skip Scanの正式サポート。 |
| **pgvector** | **0.8.1** | 2025/09/04 | HNSWインデックスの構築・検索パフォーマンス向上。反復インデックススキャン対応。 |
| **Atlas** | **v1.0.0** | 2025/12/24 | メジャーリリース到達。Monitoring as Code、Schema Statistics機能追加。 |
| **sqlc** | **1.30.0** | 2025/09/01 | pgx/v5、ENUM配列の対応強化。MySQL/SQLiteエンジンの改善。 |
| **pgx** | **v5.8.0** | 2025/12/26 | Go 1.24+必須化。パイプライン処理の改善、`pgtype.Numeric`の最適化。 |
| **Echo** | **v5.0.1** | 2026/01/28 | Professor の外向きAPI（HTTP/JSON）と **SSE** に使用。 |
| **Kafka (segmentio/kafka-go)** | **latest** | - | IngestJob の publish/consume。非同期ワーカーで OCR/構造化/Embedding準備を実行。 |
| **Google Cloud Run** | - | - | Professor / Librarian の実行基盤（ステートレス）。 |
| **Cloud SQL for PostgreSQL** | - | - | Professor の永続化ストア（pgvector）。 |
| **Google Cloud Storage (GCS)** | - | - | 講義資料の原本ストレージ（Professor のみが直接アクセス）。 |
| **Google Generative AI SDK for Go** | **latest** | - | Professor が Gemini を呼び出す（OCR/構造化/最終生成）。 |
| **slog** | - | - | Goの標準ライブラリ |
| **Testcontainers** | **v0.40.1** | 2025/11/06 | PostgresSQLにてSSL設定（WithSSLSettings）の簡略化、証明書の自動マウントと設定対応 |

## モデル利用（SSOT）
単一モデルで完結させず、用途ごとに最適モデルを使い分ける。

| フェーズ | タスク | 使用モデル |
| :--- | :--- | :--- |
| ① PDF解析 | OCR・図版認識 | **Gemini 2.0 Flash** |
| ② 構造化 | Markdown整理・要約・Embedding用ドキュメント生成 | **Gemini 2.5 Flash-Lite** |
| ③ 生成・推論 | 最終回答生成（引用元を含める） | **Gemini 3.0 Pro** |

## 通信スタック（SSOT）
- Frontend ↔ Professor: **HTTP/JSON（OpenAPI）** + **SSE**
- Professor ↔ Librarian: **gRPC（Proto）**

## 設計ポリシー
- **UUID + NanoID**: 内部主キーはUUID（推奨: UUIDv7）、外部公開キーはNanoIDを採用し、セキュリティとユーザビリティを両立する。
- **ENUM型の積極採用**: 固定値の管理はPostgreSQL ENUM型を使用し、型安全性・パフォーマンス・可読性を向上させる。
	- **VARCHAR型での固定値管理は禁止**（必ず PostgreSQL ENUM を使う）

## SSOT（Single Source of Truth）
- DBスキーマ: Atlas の `schema.hcl`
- API契約（外向き）: OpenAPI（`docs/openapi.yaml`）
- API契約（内向き）: Proto（`proto/`）
- SQL: `sql/queries/*.sql`（sqlcで生成）

## Post-MVP候補（MVPでは使わない）
- Elasticsearch（検索は Postgres/pgvector を正とする）
- Debezium CDC（Elasticsearch を採用する場合の差分同期手段）

