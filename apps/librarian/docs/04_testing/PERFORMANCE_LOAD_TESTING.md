# PERFORMANCE_LOAD_TESTING

## 目的
性能/負荷の問題を本番で初めて発見しないために、
負荷試験・容量計画・性能回帰検知の最小標準を定義する。

## 対象
- Librarian の HTTP API（Professor から呼ばれる）
- Librarian → Professor（HTTP/JSON tool endpoints）
- Librarian → Gemini API（HTTPS）

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
- 代表的なユースケースを列挙（例: 検索ループ: 1回/3回/最大回数、Gemini 成功/失敗）
- 比率（Mix）を決める（例: short 80% / long 20%）
- 条件を揃える（max_retries、同時実行数、タイムアウト、Professor 側の疑似応答など）

## 観測項目（最低限）
- RED（Request, Error, Duration）
  - RPS
  - エラー率（HTTP 5xx / timeout）
  - p95/p99 レイテンシ
- サチュレーション
  - CPU/メモリ
  - outbound 同時実行（Professor / Gemini への接続・キュー）

## ゲート（推奨）
- リリース前に “性能回帰” を検知できるようにする
  - 例: p95 が前回比 +20% 超で失敗
  - 例: エラー率が閾値超過で失敗

## 典型ボトルネック（チェックリスト）
- HTTP/outbound:
  - タイムアウト未設定
  - 直列の外部呼び出し増加（Professor ツール呼び出しの過多）
  - 再試行の誤用（retry storm）
- LLM:
  - トークン/生成時間が大きいプロンプト
  - 停止判断が弱く max_retries まで到達する割合の増加

## 実行環境
- 可能なら本番相当の構成（スケール、制限、設定）で行う
- それが難しい場合でも、
  - 相対比較（前回比）
  - ボトルネック特定
  を目的に継続実施する

## 推奨ツール（例）
- HTTP: k6 / vegeta
- プロファイル: 実行環境のプロファイラ（CPU/メモリ）

> ツールは例。重要なのは “再現性あるワークロード” と “SLOに紐づく測定”。

## 関連
- `04_testing/TEST_STRATEGY.md`
- `05_operations/SLO_ALERTING.md`
- `05_operations/OBSERVABILITY.md`
- `01_architecture/RESILIENCY.md`
- `05_operations/RELEASE_DEPLOY.md`
- `05_operations/PROGRESSIVE_DELIVERY.md`
- `05_operations/INCIDENT_POSTMORTEM.md`

## 本サービスで扱わないもの
- DB/検索基盤/イベント基盤の負荷試験・運用（Professor 側の責務）
- HTTP/JSON 以外の内部 RPC 方式の性能試験（Librarian の SSOT は HTTP/JSON）
