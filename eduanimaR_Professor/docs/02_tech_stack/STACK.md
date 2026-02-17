# STACK

Last-updated: 2026-02-17

## 技術スタック（2026年2月最新版）

| 項目 | バージョン | リリース日 | 備考・新機能要約 |
| :--- | :--- | :--- | :--- |
| **Go** | **1.25.7** | 2026/02/04 | 最新安定版。セキュリティ修正およびコンパイラ最適化を含む。 |
| **gRPC (google.golang.org/grpc)** | **latest** | - | **Professor ↔ Librarian** の内部通信。Protocol Buffers(.proto)で型安全な契約。双方向ストリーミングに対応。契約: `proto/librarian/v1/librarian.proto` |
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
| Ingestion（Professor / 前処理） | PDF/画像→Markdown化・意味単位チャンク分割（`chunks[]` をJSON出力）を **バッチ処理** で実行 | **高速推論モデル** |
| Phase 2（Plan / Professor） | タスク分割（調査項目）・停止条件（Stop Conditions）・コンテキスト定義（大戦略: WHAT） | **高速推論モデル** |
| Phase 3（Search / Librarian） | クエリ生成・ツール選択・反省/再試行・停止条件の満足判定（小戦略: HOW） | **高速推論モデル** |
| Phase 4（Answer / Professor） | 選定資料の全文Markdownを読み込み最終回答生成 | **高精度推論モデル** |

### 要件（運用ポリシー）
- Summary（要約）は **原則生成しない**（検索精度は詳細Chunkを正とする）。大量ファイルからの高速選別が必要になった場合のみ「ファイル単位Summary」を追加する。
- Phase 3（検索）中は **チャンク＋前後**を使い、Phase 4（回答）で初めて **選定資料の全文Markdown** を読み込む。

### LangGraph ループ設定（推奨 / Librarian）
- `MaxRetry`（検索ステップ上限）: **5回（3回 + 2回リカバリ）**
- 5回で停止条件に達しない場合: 「現時点の根拠で回答へ進む」または「不足を明記して終了」を許可する（無限ループ回避）

### thinking_level（推奨）
- Ingestion: `Minimal`（定型変換なので推論を最小化）
- Phase 2: `Medium`（調査項目/停止条件のミスが全体コストに直結するため）
- Phase 3: `Low`（速度優先。最終回のみ `Medium` に上げて再検討してよい）

### モデル設定（環境変数）
2モデル戦略を採用:

- `PROFESSOR_MODEL_FAST`（default: 高速推論モデル） - Ingestion/Planning用
- `LIBRARIAN_MODEL_FAST`（default: 高速推論モデル） - Search用
- `PROFESSOR_MODEL_ACCURATE`（default: 高精度推論モデル） - Answer用

**注意:** Gemini 2.0 Flash提供終了により、OCR/構造化処理も高速推論モデルで実行します。

## 通信スタック（SSOT）
- Frontend ↔ Professor: **HTTP/JSON（OpenAPI）** + **SSE**
- Professor ↔ Librarian: **gRPC（Proto、双方向ストリーミング）**、契約: `proto/librarian/v1/librarian.proto`

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

---

## Phase 1で明示的に使わない技術（Phase 2以降に延期）

| 技術 | 延期理由 |
|:---|:---|
| **Kafka** | Phase 1は同期処理のみ（非同期ワーカー不要） |
| **Elasticsearch** | Phase 1はpgvectorのみで検証（Hybrid検索はPhase 3） |
| **Debezium CDC** | Phase 1はリアルタイム同期不要 |
| **SSO認証** | Phase 1は固定dev-userで動作確認 |

Phase 1の技術スタック（確定版）:
- Go 1.25.7
- PostgreSQL 18.1 + pgvector 0.8.1
- 高速推論モデル（OCR/埋め込み）
- Echo v5.0.1（HTTP API）
- gRPC（Professor ↔ Librarian内部通信）

