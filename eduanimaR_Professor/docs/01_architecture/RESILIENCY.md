# RESILIENCY（eduanima+R）

## 目的
Professor（OpenAPI + SSE / Kafka Worker / DB/GCS）と Librarian（gRPC）からなる分散フローで必ず起きる、遅延・部分失敗・再試行・再処理を安全なデフォルトで扱う。

## 適用範囲
- Frontend（Web/Chrome拡張） ↔ Professor（HTTP/JSON + SSE）
- Professor ↔ Librarian（gRPC）
- Professor ↔ Cloud SQL(PostgreSQL/pgvector)
- Professor ↔ GCS
- Professor ↔ Kafka（produce/consume）
- Gemini API（Ingestion/Phase 2/Phase 3/Phase 4）

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
- 推論API: **フェーズ別**に timeout と並列数を分ける（bulkhead）
	- Ingestion（高速推論モデル）: 重い入力（PDF/画像）を想定し timeout を長め、並列数は控えめ
	- Phase 2（高速推論モデル / Professor）: 短時間でPlan（調査項目/停止条件）生成。並列は上げても良いが rate limit を尊重
	- Phase 3（高速推論モデル / Librarian）: ループ回数が増えるため 1回あたりのtimeoutを短め＋MaxRetryで上限を固定
	- Phase 4（高精度推論モデル / Professor）: 最終生成は高コストなので並列を強く絞る

> 注: Phase 1〜3 は同一モデル（高速推論モデル）でも、**用途別にキュー/並列数/timeout を分離**して連鎖障害を防ぐ。モデルIDは環境変数で上書き可能。

## Retry / 再処理
### HTTP
- 読み取りは短いリトライを許容する（対象エラーを限定）
- 書き込みは原則リトライしない（必要なら Idempotency-Key を設計する）

### Kafka（Ingestion）
- at-least-once を前提に、**結果側（DB永続化）を冪等**にする
- DLQ を用意し、手動リドライブ（再投入）で復旧できるようにする

### Phase 3（検索ループ）の無限ループ対策（MUST）
- LangGraph の `MaxRetry`（回数）に加え、Professor 側で wall-clock の上限時間も設ける
- **推奨 MaxRetry: 5回（3回 + 2回のリカバリ）**。それ以上は「嘘の検索」やコスト増のリスクが上がる
- 停止条件に達しない場合でも、Librarian は「不足している情報」を明示して終了できること（早期諦め/無限ループを防ぐ）
- Phase 3は `thinking_level: Low` を基本にし、**最終回のみ Medium に上げて“本当に見つからないか”を再検討**してよい

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
