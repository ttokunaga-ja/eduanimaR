# RESILIENCY

## 目的
分散システムで必ず発生する「遅延・部分失敗・再試行」に対し、サービス横断で一貫した設計基準を定義する。

## 適用範囲
- Next.js(BFF) ↔ Go API Gateway（HTTP）
- Go API Gateway ↔ Go Microservices（gRPC）
- DB / Elasticsearch / Kafka / 外部API

## 基本原則（MUST）
- **Timeout は必須**（無限待ち禁止）。上流の deadline/cancellation を下流へ伝播する。
- **Retry は例外**。冪等性が確認できる操作のみ。失敗増幅（retry storm）を防ぐ。
- **Idempotency を設計に組み込む**（特に「作成/決済/状態遷移」）。
- **Concurrency を制御**（接続プール、ワーカー数、キュー長、バックプレッシャ）。

## Timeout / Deadline
- HTTP/gRPC のクライアントは必ず deadline を設定する
- DB/ES/Kafka も context でキャンセル可能にする
- 入口（BFF/Gateway）で設定した deadline を超える処理は禁止

## Retry
### Retry してよい（SHOULD）
- 読み取り（GET 相当）で、依存先が一時的に不安定な場合
- **冪等キー** 付きの書き込み（後述）

### Retry してはいけない（SHOULD NOT）
- 冪等性がない書き込み（同一操作が重複実行されうる）
- 依存先が過負荷で落ちているとき（回復を遅らせる）

### ルール（SHOULD）
- 指数バックオフ + ジッタ
- 最大試行回数を固定（例: 2〜3回）
- リトライ対象はエラー種別/ステータスで絞る

## Idempotency
### 目的
- ネットワーク再送/タイムアウト/クライアント再試行で同一リクエストが重複しても、結果を一意にする。

### 適用対象（SHOULD）
- `POST /orders` のような「作成」
- 決済/在庫引当/ポイント付与などの業務フロー

### 方式（例）
- `Idempotency-Key`（外部HTTP）または `idempotency_key`（gRPC metadata/field）を受け取る
- オーナーサービスがキーを保存し、同一キーは同一結果を返す

## Circuit Breaker / Bulkhead（推奨）
- 依存先（外部API/ES 等）ごとに隔離（bulkhead）し、連鎖障害を防ぐ
- 失敗率/遅延が閾値を超えたら短時間遮断し、回復を待つ

## Rate Limit / Backpressure（推奨）
- Gateway で外部からの流量制限（ユーザー/トークン/ルート単位）
- 内部はワーカー数・キュー長・接続プールで制御

## Graceful Shutdown（推奨）
- 新規受付停止 → 処理中完了待ち → タイムアウトで強制終了
- readiness/liveness と整合する（関連: PROTOBUF_GRPC_STANDARDS）

## 関連
- 03_integration/INTER_SERVICE_COMM.md
- 03_integration/PROTOBUF_GRPC_STANDARDS.md
- 05_operations/SLO_ALERTING.md
