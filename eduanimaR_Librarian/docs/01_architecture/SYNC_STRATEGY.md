# SYNC_STRATEGY

## 目的
PostgreSQL と Elasticsearch のデータ整合性を、運用可能なコストで維持するための同期戦略を定義する。

## 前提
- 検索/集計は Elasticsearch 9.2.4 を第一候補とする
- 差分同期は Debezium CDC（Kafka経由）を基本とする

## 代表的パターンと採用指針
### 1) Debezium CDC（推奨）
- 概要: PostgreSQL の論理レプリケーション → Debezium → Kafka → Indexer → Elasticsearch
- 長所: アプリの書き込み経路を汚さない、再処理がしやすい
- 注意:
  - スキーマ変更時の互換性（トピック/イベント）
  - 冪等（idempotency）と順序保証

### 2) Transactional Outbox
- 概要: アプリがDB書き込みと同一Txで outbox にイベントを書き、別プロセスが配送
- 長所: 整合性が取りやすい
- 注意: outbox掃除/再配送/遅延の設計が必要

### 3) Dual Write（非推奨）
- 概要: アプリがDBとESへ同時書き込み
- リスク: 部分失敗で不整合が発生しやすい

## 整合性レベル（定義必須）
- 検索は結果整合（eventual consistency）を許容するか
- 許容遅延（例: P95で5秒以内等）
- 不整合検知とリカバリ（フルリインデックス手順）

> SLO/アラート（遅延・DLQ）は `05_operations/SLO_ALERTING.md` を参照。

## 失敗時の原則
- Indexerは冪等に実装し、同一イベント再処理で結果が変わらないこと
- デッドレター（DLQ）を用意し、手動介入で復旧できること

> イベント契約・DLQ・再処理は `03_integration/EVENT_CONTRACTS.md` を参照。

## 関連
- `03_integration/EVENT_CONTRACTS.md`
- `05_operations/SLO_ALERTING.md`
