# RESILIENCY

## 目的
分散システムで必ず発生する「遅延・部分失敗・再試行」に対し、サービス横断で一貫した設計基準を定義する。

## 適用範囲（Librarian）
- Professor（Go）↔ Librarian（Python）（**gRPC、双方向ストリーミング**）
- Librarian ↔ Gemini API（HTTPS）

## 基本原則（MUST）
- **Timeout は必須**（無限待ち禁止）。上流の deadline/cancellation を下流へ伝播する。
- **Retry は例外**。冪等性が確認できる操作のみ。失敗増幅（retry storm）を防ぐ。
- **Idempotency を設計に組み込む**（特に「作成/決済/状態遷移」）。
- **Concurrency を制御**（接続プール、ワーカー数、キュー長、バックプレッシャ）。

## Timeout / Deadline
- すべての gRPC/HTTP クライアントは timeout を必須化する（無限待ち禁止）
- Professor→Librarian の gRPC 呼び出しは request timeout を前提に設計し、Librarian は上流の締切内で「最善の結果」を返す
- gRPC の deadline/cancellation を受け取った場合は速やかに処理を中断する
- Gemini 呼び出しも timeout を設定し、上流キャンセル時は速やかに中断する

## Retry
### Retry してよい（SHOULD）
- Professor の検索ツール呼び出し（gRPC経由、読み取り相当）で、ネットワーク起因の一時失敗のみ
- Gemini API 呼び出しで 5xx/一時エラーのみ（指数バックオフ + ジッタ）

### Retry してはいけない（SHOULD NOT）
- 同一リクエストで結果が大きく変動しうる「推論ループの再実行」を、ネットワーク retry と混同して行うこと
	- 推論ループの繰り返しは MaxRetry と停止条件で制御する（アプリケーションレベルの制御）

### ルール（SHOULD）
- 指数バックオフ + ジッタ
- 最大試行回数を固定（例: 2〜3回）
- リトライ対象はエラー種別/ステータスで絞る

## Idempotency
### 目的
- ネットワーク再送/タイムアウト/クライアント再試行で同一リクエストが重複しても、結果を一意にする。

### 適用対象（SHOULD）
- `POST /v1/librarian/search-agent` は「同じ入力なら同じ出力が望ましい」処理であり、Professor 側で `request_id` を付与し相関する

> Librarian は永続化しないため、厳密な意味での idempotency-store は保持できない。
> 代わりに、Professor 側で冪等キー/キャッシュ/重複排除を行う。

### 方式（例）
- `Idempotency-Key`（HTTP header）を受け取る
- オーナーサービスがキーを保存し、同一キーは同一結果を返す

## Circuit Breaker / Bulkhead（推奨）
- 依存先（Gemini / Professor tool endpoints）ごとに隔離（bulkhead）し、連鎖障害を防ぐ
- 失敗率/遅延が閾値を超えたら短時間遮断し、回復を待つ

## Rate Limit / Backpressure（推奨）
- Librarian 側は同時実行数（ワーカー/コネクション）を制御し、Gemini への過剰送信を防ぐ
- Professor 側は Librarian 呼び出しを含めた全体流量を制御する（外部公開面がある場合）

## Graceful Shutdown（推奨）
- 新規受付停止 → 処理中完了待ち → タイムアウトで強制終了
- readiness/liveness と整合する

## 関連
- `03_integration/INTER_SERVICE_COMM.md`
- `03_integration/ERROR_HANDLING.md`
