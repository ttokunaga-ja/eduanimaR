# SLICES_MAP（Frontend slices ↔ Professor API）

## 位置づけ
フロントエンド（Web / Chrome拡張）の **機能スライス (slices)** と、バックエンド（Professor）の **公開IF（OpenAPI + SSE）** の対応関係を定義する。

## 前提
- Frontend は Professor のみを直接呼ぶ（Librarianは内部サービスであり直接呼ばない）
- 外向き契約は OpenAPI（`docs/openapi.yaml`）を正とする

## 対応表（例）
| FE Slice | Professor API / Stream | 説明 |
| --- | --- | --- |
| `entities/subject/` | subjects API | 科目の取得・表示 |
| `entities/material/` | materials API | 資料一覧・メタデータ表示 |
| `features/auth/` | auth API | OIDCログイン/セッション管理 |
| `features/ingest/upload/` | ingest API | アップロード → GCS → Kafka投入 |
| `features/chat/ask/` | chat API + SSE | 質問送信 + 進捗/回答/引用のストリーミング |
| `widgets/chat-panel/` | SSE consumer | ストリーミングUI（再接続を考慮） |
| `widgets/materials-tree/` | subjects/materials | 科目ツリー + 資料ブラウズ |

## 最小ルール
- 新機能追加時は、まず Professor 側の責務（usecase）を決める
- 次に slice の配置（entities/features/widgets）を決める
- “検索” は Professor が DB を検索する（MVPで Elasticsearch は使用しない）

