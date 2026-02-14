# MICROSERVICES_MAP（eduanima+R / Professor）

## 目的
サービス境界（責務）・依存関係・公開IF（OpenAPI / gRPC）を SSOT として固定し、機能追加時に「どこを触るべきか」を最短で判断できるようにする。

## 結論：サービス境界（MUST）
- 本システムは **2サービス構成** とする。
	- **eduanima-professor（Go）**: 司令塔 / データの守護者。外向きAPI、認証・認可、GCS/Kafka/DBの管理、検索の物理制約強制、最終回答合成。
	- **eduanima-librarian（Python）**: 探索（Agentic Search）と評価ループ。DB/GCSへ直接アクセスしない。
- **DB/GCSへの直接アクセス権限は Professor のみに付与**する（最重要不変条件）。

## 契約（SSOT）
- 外向き契約（Frontend ↔ Professor）: OpenAPI（`docs/openapi.yaml`）
- 内向き契約（Professor ↔ Librarian）: gRPC/Proto（`proto/`）

> 注: 現時点でファイルが未作成の場合でも、SSOT の置き場は上記に固定する。

## サービス一覧
| Service | Responsibility | Owning Data | Exposed APIs | Depends On | Runtime |
| --- | --- | --- | --- | --- | --- |
| **eduanima-professor (Go)** | Gateway/Orchestration、認証・認可、資料インジェスト統治、検索の物理制約強制、最終回答合成 | **Cloud SQL(PostgreSQL+pgvector)**、**GCS(原本)**、Kafka(ジョブ) | **HTTP/JSON(OpenAPI)** + **SSE**（外向き） / **gRPC client**（Librarian向け） | OIDC Provider、GCS、Kafka、Gemini API、Librarian(gRPC) | Cloud Run |
| **eduanima-librarian (Python)** | **Phase 3（検索）**: 再検索ループ（LangGraph）、情報十分性判断、Professorへのツール要求（検索実行） | なし（ステートレス） | **gRPC server**（Professorから呼び出し） | Gemini API、Professor（検索結果） | Cloud Run |

## 依存関係（通信方向）
- Frontend → Professor（HTTP/OpenAPI）
- Frontend ← Professor（SSE: 進捗/引用/回答ストリーム）
- Professor → Librarian（gRPC: 探索開始/評価ループ）
- Librarian → Professor（gRPC応答: 検索要求/探索完了通知/資料ID候補）
- Professor → DB/GCS/Kafka（直接アクセス）

## 主要フロー

### Ingestion Loop（資料解析）
1. **Receive**: Frontend（Chrome拡張含む）からファイル受信（user_id, subject_id を確定）
2. **Upload**: Professor が原本を GCS に保存
3. **Produce**: Professor が Kafka に `IngestJob` を publish（冪等キーを含む）
4. **Consume**: Professor ワーカーがジョブを consume
5. **Ingestion（Vision→Chunks）**: **Gemini 3 Flash** で原本（PDF/画像）を **Markdown化/意味単位チャンク分割**し、Structured Outputs（JSON）で `chunks[]` のみを生成
6. **Store**: Professor が Postgres（pgvector）へ永続化（UUIDv7、subject_id/user_id による物理制約を前提）

> 注: 要約（Summary）は **原則生成しない**。大量ファイルからの高速な候補絞り込みが必要になった場合のみ「ファイル単位の短いSummary」を追加で生成する。

### Reasoning Loop（検索・回答）
1. **Start**: Frontend が質問を Professor に送信
2. **Phase 2（Plan）**: Professor が **Gemini 3 Flash** で「意図解釈・検索戦略・停止条件（終了条件）・MaxRetry」を JSON で生成
3. **Phase 3（Search: Python/LangGraph）**: Professor が gRPC で Librarian を起動（質問 + Plan + user_id + subject_id + 制約）
4. **Execute Search（Professor）**: Professor が DB を検索し、**subject_id/user_id/is_active 等の WHERE を強制**して結果（Chunk＋前後）を返却
5. **Loop**: Librarian が **Gemini 3 Flash** で結果を評価し、停止条件に満たなければ再検索（MaxRetry/時間は Professor が制御。推奨: 最大5回 = 3回 + 2回リカバリ）
6. **Finalize**: Librarian が「収集完了」または「不足を宣言して終了」として資料ID/根拠候補を返す
7. **Phase 4（Answer/RAG）**: Professor が **選定された資料のみ全文Markdown** をDBから取得し、**Gemini 3 Pro** で最終回答を生成
8. **Stream**: Professor が SSE で回答・引用・進捗をFrontendへストリーミング

## Phase 3（検索ループ）の設定指針（SSOT）
LangGraph を用いた循環型エージェントで重要なのは **「何回回すか」と「どう止めるか」**。

### MaxRetry（推奨: 最大 5 回）
- 1回目: 直球検索（Phase 2の計画で最も確度が高いクエリ）
- 2回目: 補完・修正（不足要素の狙い撃ち）
- 3回目: 類義語・広域（専門→一般、正規表現を緩める等）
- 4〜5回目: フォールバック（目次/前後関係/周辺章からの材料集め）

> 5回で足りない場合は、それ以上回してもハルシネーション（嘘の検索）やコスト増のリスクが上がる。
> **「現時点で見つかった根拠だけでPhase 4へ進む」または「不足を報告して終了」**を脱出口として必ず用意する。

## Phase 2（Plan JSON）の役割（合格基準/ルーブリック）
Phase 2は Phase 3 のための **合格基準（Definition of Done）** を作る工程。

### 分割（Decomposition）
質問を「意味の最小単位」に分解し、探索のチェックリストにする。

### 停止条件（Stop Conditions）
- **充足性（Sufficiency）**: 必須項目（式/定義/ケース等）が根拠と紐付いて揃っている
- **明確性（Unambiguity）**: 近似概念（相関係数など）と混同していない
- **視覚情報の言語化（Visual Check）**: 図表の凡例/線種/注記など“指しているもの”が確保できている

### thinking_level（推奨）
- Ingestion: `Minimal`
- Phase 2: `Medium`
- Phase 3: `Low`（ただし最終回のみ `Medium` に上げて再検討してよい）

## モデル設定（環境変数）（SSOT）
モデルID（例: `gemini-3-flash` / `gemini-3-pro`）は環境変数で上書きできる。

- `PROFESSOR_GEMINI_MODEL_INGESTION`（default: Gemini 3 Flash）
- `PROFESSOR_GEMINI_MODEL_PLANNING`（default: Gemini 3 Flash）
- `LIBRARIAN_GEMINI_MODEL_SEARCH`（default: Gemini 3 Flash）
- `PROFESSOR_GEMINI_MODEL_ANSWER`（default: Gemini 3 Pro）

## 不変条件（MUST）
- Librarian は DB/GCS に直接アクセスしない（資格情報を持たない）
- 検索の物理制約（subject_id/user_id/is_active 等）は Professor が必ず強制する
- MVP では Elasticsearch は使用しない（検索は Postgres を正とする）

## 変更時のチェックリスト
- “新しいデータ” を誰が所有するか：**原則 Professor** になっているか
- Frontend が Librarian を直接呼べる経路が生えていないか
- gRPC/Proto と OpenAPI の SSOT の場所が変わっていないか（`proto/`, `docs/openapi.yaml`）
- 検索クエリに subject_id/user_id が必ず入る設計・実装になっているか
