# CLEAN_ARCHITECTURE（eduanima-professor）

## 目的
Professor（Go）を「インフラ・実行部隊・最終回答者（実務） + データの守護者」として成長させても破綻しないように、ディレクトリ構成と依存方向（境界）を固定する。

> 補足: 思考（検索戦略の立案・充足判定・LangGraphのループ）は Python（Librarian）が担い、Professor はその要求を **安全に実行**（検索/取得/権限強制）し、最後に回答を合成する。

> 修正（責務の2段階）: 検索戦略は **大戦略（Phase 2: WHAT/停止条件/タスク分割）** と **小戦略（Phase 3: HOW/クエリ生成/再試行/終了判定）** に分かれる。
> - Phase 2（大戦略/プランニング）は Professor（Go）が担う（高速推論モデル で「調査項目」と「停止条件」を作る）
> - Phase 3（小戦略/タクティクス）は Librarian（Python）が担う（高速推論モデル でクエリ生成・ツール選択・反省/再試行・停止条件の満足判定を行う）
> - Professor（Go）は **検索ツールの実行（DB検索の物理実行 + 制約/権限強制）** と **最終回答生成（高精度推論モデル）** を担う

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
  - `gemini/`（高速推論モデル / 高精度推論モデル の呼び出し実装。モデルは環境変数で切替）
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

### 4) Phase別の責務分担

#### Phase 1: バックエンド完成（ローカル開発）
- **Professor責務**:
  - Professor APIが完全に動作（OpenAPI定義完備）
  - Librarian推論ループとの統合完了（gRPC双方向ストリーミング）
  - 認証不要でcurlリクエストによる資料アップロードが可能（開発用エンドポイント）
  - OCR + 構造化処理（高速推論モデル）
  - pgvector埋め込み生成・保存（HNSW検索）
  - **Web版固有機能のAPI提供**: 科目一覧・資料一覧・会話履歴取得
  - **QA機能のSSEストリーミング**: thinking/searching/evidence/answer/completeイベント配信
  - **フィードバック受信**: Good/Badフィードバックの保存
- **Librarian責務**:
  - Phase 3の小戦略実行: クエリ生成（最大5回試行）
  - 停止条件の満足判定
  - Professor経由での検索実行（DB/GCS直接アクセス禁止）

#### Phase 2: SSO認証 + 本番環境デプロイ
- **Professor責務追加**:
  - SSO認証基盤（OAuth/OIDC）実装
  - 本番環境デプロイ（Google Cloud Run）
  - 拡張機能からの資料自動アップロード本番適用
  - 未登録ユーザーへの適切なエラーレスポンス（`AUTH_USER_NOT_REGISTERED`）
- **Librarian責務**: Phase 1と同じ

#### Phase 3: Chrome Web Store公開
- **Professor責務**: Phase 2から変更なし
- **Librarian責務**: Phase 1と同じ

#### Phase 4: 閲覧中画面の解説機能追加
- **Professor責務追加**:
  - HTML・画像を受け取るエンドポイント追加
  - Gemini Vision APIでの画像解析
  - 資料との関連付けロジック追加
- **Librarian責務**: Phase 1と同じ

#### Phase 5: 学習計画立案機能（構想段階）
- **Professor責務追加**（未確定）:
  - 小テスト結果の保存・分析
  - 学習計画生成API
- **Librarian責務**: Phase 1と同じ

## 禁止事項
- transport から直接DBクエリを実行しない
- domain が pgx/sqlc/transport/SDK に依存しない
- Librarian へ DB/GCS 直接アクセス経路を作らない
- **Web版からのファイルアップロードUIを持つエンドポイントを作らない**（拡張機能の自動アップロードのみ）
- **Web版からの新規ユーザー登録エンドポイントを作らない**（拡張機能のSSO登録のみ）
