# CLEAN_ARCHITECTURE（eduanima-professor）

## 目的
Professor（Go）を「司令塔 + データの守護者」として成長させても破綻しないように、ディレクトリ構成と依存方向（境界）を固定する。

## 前提
- 外向き: HTTP/JSON（OpenAPI） + SSE
- 内向き: Librarian とは gRPC（Proto）
- DB/GCSへ直接アクセスできるのは Professor のみ

## 推奨レイアウト（Standard Go Layout + Clean Architecture）
- `cmd/eduanima-professor/`
  - エントリポイント（main）。DI（依存注入）の組み立てのみ
- `internal/transport/`
  - `internal/transport/http/`: HTTP(OpenAPI) + SSE の handler
  - `internal/transport/worker/`: Kafka consumer / worker の起動・制御
- `internal/usecase/`
  - `ingest/`: 受領 → GCS → Kafka 投入
  - `ingestworker/`: consume → OCR/構造化 → DB 永続化
  - `orchestration/`: 質問受付 → Librarian呼び出し → 進捗統合
  - `search/`: Librarian検索要求の受理 → DB検索（物理制約強制）
  - `synthesis/`: 収集済み資料から最終回答を合成
- `internal/domain/`
  - エンティティ/値オブジェクト/ドメインエラー
- `internal/ports/`
  - usecase が依存する抽象（DB/GCS/Kafka/LLM/Librarian）
- `internal/adapters/`
  - `postgres/`（pgx + sqlc + pgvector）
  - `gcs/`
  - `kafka/`（producer/consumer）
  - `librariangrpc/`（gRPC client）
  - `gemini/`（2.0 Flash / 2.5 Flash-Lite / 3.0 Pro の呼び出し実装）
- `pkg/`
  - 横断共有してよい（かつ安定）なライブラリのみ（乱用禁止）

## 依存方向（MUST）
- `transport` → `usecase` → `domain`
- `adapters` → `ports` → `domain`
- `usecase` は `ports`（interface）にのみ依存し、`adapters` の実装に依存しない

## Professor 固有の不変条件（MUST）
### 1) DB/GCS への直接アクセスの独占
- Postgres/GCS の認証情報は Professor のみに付与する
- Librarian は DB/GCS の認証情報を持たない（ネットワーク的にも閉じる）

### 2) 検索の物理制約（Physical Constraint Enforcement）
- Librarian から渡されるのは「検索意図」であり、SQLは Professor が確定する
- MUST: `subject_id`, `user_id`, `is_active` 等の強制条件は Repository 層で必ず付与する
- MUST NOT: Librarian から渡されたフィルタをそのまま WHERE に反映して制約を回避させない

### 3) 契約の境界
- OpenAPI（`docs/openapi.yaml`）と Proto（`proto/`）が契約の正
- sqlc / OpenAPI / Proto などの生成物を手で編集しない

## 禁止事項
- transport から直接DBクエリを実行しない
- domain が pgx/sqlc/transport/SDK に依存しない
- Librarian へ DB/GCS 直接アクセス経路を作らない
