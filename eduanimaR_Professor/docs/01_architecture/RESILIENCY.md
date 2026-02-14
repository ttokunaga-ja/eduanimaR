# RESILIENCY（eduanima+R）

## 目的
Professor（OpenAPI + SSE / Kafka Worker / DB/GCS）と Librarian（gRPC）からなる分散フローで必ず起きる、遅延・部分失敗・再試行・再処理を安全なデフォルトで扱う。

## 適用範囲
- Frontend（Web/Chrome拡張） ↔ Professor（HTTP/JSON + SSE）
- Professor ↔ Librarian（gRPC）
- Professor ↔ Cloud SQL(PostgreSQL/pgvector)
- Professor ↔ GCS
- Professor ↔ Kafka（produce/consume）
- Professor ↔ Gemini API（OCR/構造化/最終生成）

## 基本原則（MUST）
- **Timeout は必須**（無限待ち禁止）。上流の deadline/cancellation を下流へ伝播する
- **Retry は例外**。冪等性が確認できる操作のみ（retry storm を防ぐ）
- **Idempotency を設計に組み込む**（特に IngestJob / 派生生成 / DB保存）
- **Backpressure を制御**（Kafka consumer、DB pool、Gemini並列数）

## Timeout / Deadline
### 外向きHTTP + SSE
- 入口は Professor。リクエスト（質問/アップロード）には上限タイムアウトを設ける
- SSE はクライアント切断を前提にする
	- 切断したらサーバ側処理を中断できる箇所（context）と、中断しない箇所（非同期ジョブ）を分ける
	- 進捗イベントは **重複してもUIが破綻しない**設計にする（再接続・再購読を想定）

### gRPC（Professor ↔ Librarian）
- gRPC クライアントは必ず deadline を設定し、Librarianへ伝播する
- Librarian側も context cancel に追随し、探索を停止できるようにする

### DB/GCS/Kafka/Gemini
- DB: クエリは常に deadline 付き（subject_id/user_id の物理絞り込み前提）
- Kafka: consume は graceful shutdown を実装し、in-flight の扱い（ack/commit）を定義する
- Gemini: モデル別（2.0 Flash / 2.5 Flash-Lite / 3.0 Pro）に timeout と並列数を分ける（bulkhead）

## Retry / 再処理
### HTTP
- 読み取りは短いリトライを許容する（対象エラーを限定）
- 書き込みは原則リトライしない（必要なら Idempotency-Key を設計する）

### Kafka（Ingestion）
- at-least-once を前提に、**結果側（DB永続化）を冪等**にする
- DLQ を用意し、手動リドライブ（再投入）で復旧できるようにする

## Bulkhead / Rate Limit（推奨）
- Gemini・DB・Librarian を別々の同時実行数/キューで隔離し、連鎖障害を防ぐ
- 外向きはユーザー単位で rate limit を設定し、過負荷時でも全体が落ちないようにする

## Graceful Shutdown（MUST）
- HTTP: 新規受付停止 → SSE/処理中の終了待ち → タイムアウトで終了
- Worker: in-flight の完了/中断ルールを定義し、二重処理が起きても冪等で吸収する

## 関連
- `03_integration/INTER_SERVICE_COMM.md`
- `03_integration/PROTOBUF_GRPC_STANDARDS.md`
- `05_operations/SLO_ALERTING.md`
