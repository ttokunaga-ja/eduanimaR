# PERFORMANCE_LOAD_TESTING

## 目的
性能/負荷の問題を本番で初めて発見しないために、
負荷試験・容量計画・性能回帰検知の最小標準を定義する。

## 対象
- 外部API（Professor: HTTP/JSON + SSE）
- 内部依存（Professor → Librarian: gRPC）
- DB（PostgreSQL）
- Kafka（consumer lag / DLQ）
- GCS（アップロード/ダウンロード/依存障害）

## 原則（MUST）
- SLO の指標（成功率/レイテンシ）と一致する形で測る
- 代表的ワークロードを定義して再現性を持たせる
- 1回の測定結果ではなく、トレンド（回帰）を見て判断する

## 試験の種類
- スモーク負荷: 小さな負荷で疎通
- ステップ負荷: 段階的にRPSを上げて限界点を探る
- ソーク（長時間）: メモリリーク/コネクション枯渇検知
- スパイク: 急増時の挙動（レート制限/バックプレッシャ）

## ワークロード設計（MUST）
- 重要ユーザージャーニーを列挙（例: ingest開始→処理完了、質問→SSEで回答、検索→表示）
- 比率（Mix）を決める（例: read 90% / write 10%）
- データ条件を揃える（キャッシュ有無、ホット/コールド等）

## 観測項目（最低限）
- RED（Request, Error, Duration）
  - RPS
  - エラー率（HTTP 5xx / gRPC status）
  - p95/p99 レイテンシ
- サチュレーション
  - CPU/メモリ
  - DB connection pool
  - Kafka consumer lag
  - gRPC concurrency / queue（Librarian呼び出し）

## ゲート（推奨）
- リリース前に “性能回帰” を検知できるようにする
  - 例: p95 が前回比 +20% 超で失敗
  - 例: エラー率が閾値超過で失敗

## 典型ボトルネック（チェックリスト）
- DB:
  - N+1
  - インデックス不足
  - ロック競合
  - コネクション枯渇
- SSE:
  - クライアント切断後も処理が継続してしまう（キャンセル伝播漏れ）
  - 1接続あたりのメモリ/バッファ肥大
- gRPC:
  - deadline 未設定
  - 直列呼び出しの増加
- Kafka/worker:
  - consumer が追いつかない（lag増大）
  - DLQ増加（恒久エラーの混入）
- 外部依存（GCS/LLM等）:
  - 依存障害によるタイムアウト連鎖

## 実行環境
- 可能なら本番相当の構成（スケール、制限、設定）で行う
- それが難しい場合でも、
  - 相対比較（前回比）
  - ボトルネック特定
  を目的に継続実施する

## 推奨ツール（例）
- HTTP: k6 / vegeta
- SSE: 最小の負荷クライアント（HTTPストリームを読み切る）を用意する
- gRPC: ghz
- プロファイル: Go pprof

> ツールは例。重要なのは “再現性あるワークロード” と “SLOに紐づく測定”。

## 関連
- `04_testing/TEST_STRATEGY.md`
- `05_operations/SLO_ALERTING.md`
- `05_operations/OBSERVABILITY.md`
- `01_architecture/RESILIENCY.md`
- `05_operations/RELEASE_DEPLOY.md`
- `05_operations/PROGRESSIVE_DELIVERY.md`
- `05_operations/INCIDENT_POSTMORTEM.md`
