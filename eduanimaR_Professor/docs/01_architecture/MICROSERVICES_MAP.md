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
| **eduanima-librarian (Python)** | 検索戦略立案、再検索ループ、評価、Professorへのツール要求（検索実行） | なし（ステートレス） | **gRPC server**（Professorから呼び出し） | Gemini API、Professor（検索結果） | Cloud Run |

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
5. **OCR/Vision**: **Gemini 2.0 Flash** で OCR・図版認識（原本 → 抽出テキスト/構造の素材）
6. **Structure/Embed-prep**: **Gemini 2.5 Flash-Lite** で Markdown 整理・要約・Embedding用ドキュメント生成
7. **Store**: Professor が Postgres（pgvector）へ永続化（UUIDv7、subject_id/user_id による物理制約を前提）

### Reasoning Loop（検索・回答）
1. **Start**: Frontend が質問を Professor に送信
2. **Call Librarian**: Professor が gRPC で Librarian を起動（質問 + user_id + subject_id + 制約）
3. **Search Request**: Librarian が必要に応じて「検索意図」を Professor へ返す
4. **Execute Search（Professor）**: Professor が DB を検索し、**subject_id/user_id/is_active 等の WHERE を強制**して結果を返却
5. **Loop**: Librarian が結果を評価し、足りなければ再検索（最大回数/時間は Professor が制御）
6. **Finalize**: Librarian が「収集完了」として資料ID/根拠候補を返す
7. **Synthesize（Professor）**: Professor が該当資料のMarkdown/ChunkをDBから取得し、**Gemini 3.0 Pro** で最終回答を生成（引用元のページ番号等を含める）
8. **Stream**: Professor が SSE で回答・引用・進捗をFrontendへストリーミング

## 不変条件（MUST）
- Librarian は DB/GCS に直接アクセスしない（資格情報を持たない）
- 検索の物理制約（subject_id/user_id/is_active 等）は Professor が必ず強制する
- MVP では Elasticsearch は使用しない（検索は Postgres を正とする）

## 変更時のチェックリスト
- “新しいデータ” を誰が所有するか：**原則 Professor** になっているか
- Frontend が Librarian を直接呼べる経路が生えていないか
- gRPC/Proto と OpenAPI の SSOT の場所が変わっていないか（`proto/`, `docs/openapi.yaml`）
- 検索クエリに subject_id/user_id が必ず入る設計・実装になっているか
