# EVENT_CONTRACTS

## 目的
Kafka 等のイベント連携における **契約（スキーマ）・互換性・再処理** の標準を定義する。

## SSOT（Single Source of Truth）
- イベントスキーマ（方式はプロジェクトで決定）を SSOT とする
  - 例: Avro / JSON Schema / Protobuf
- Producer がスキーマと意味を所有し、Consumer は後方互換を前提に実装する

## 互換性ルール（MUST）
- フィールド追加: 原則許容（任意フィールドとして）
- 既存フィールドの削除/型変更/意味変更: 破壊的
- 破壊的変更が必要な場合: トピック名またはイベント名をバージョン分離

## イベントの最小フィールド（推奨）
- `event_id`: 一意ID（重複排除・冪等処理）
- `occurred_at`: 発生時刻
- `producer`: 発行元サービス
- `trace_id` / `request_id`: 相関（観測性）

## パーティション/順序
- 順序保証が必要な単位（例: `user_id`）を partition key にする
- 順序を前提にしすぎない（再処理/遅延で崩れる可能性を考慮）

## 配信保証と冪等性
- 原則 at-least-once を前提
- Consumer は必ず冪等に実装する
- 冪等キーは `event_id`（または業務キー）を用いる

## DLQ / 再処理（MUST）
- DLQ（dead letter queue）を用意する
- poison message（永続的に失敗するメッセージ）の扱いを決める
- 再処理の手順を Runbook として残す（関連: SLO_ALERTING / OBSERVABILITY）

## CDC との関係
- Debezium CDC を使う場合、
  - スキーマ変更時の互換性
  - 再スナップショット/再同期手順
  - 期待する順序保証
を事前に決める（関連: SYNC_STRATEGY）

## 関連
- 01_architecture/SYNC_STRATEGY.md
- 03_integration/PROTOBUF_GRPC_STANDARDS.md（イベント互換性の方針）
- 05_operations/OBSERVABILITY.md
