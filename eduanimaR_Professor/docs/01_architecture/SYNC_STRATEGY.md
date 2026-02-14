# SYNC_STRATEGY（MVP: GCS / Kafka / PostgreSQL）

## 目的
原本（GCS）と派生データ（PostgreSQL: Markdown/Chunk/Embedding 等）の整合性を、運用可能なコストで維持するための同期・再処理戦略を定義する。

## 前提（MVP）
- 検索は PostgreSQL（pgvector）を正とする
- 資料解析は Kafka を用いた非同期（IngestJob）で実行する
- DB/GCSへ直接アクセスできるのは Professor のみ
- MVPでは Elasticsearch は使用しない

## 整合性の単位
### 1) 原本（GCS）
- 保存が成功した時点で、以降の再処理が可能になる（原本が正）

### 2) 派生（PostgreSQL）
- Markdown、Chunk、Embedding（必要ならSummary等）は “派生” として世代管理できる設計にする
- 不整合の修復は「再解析（再ジョブ投入）」で行えること

## Ingestion Loop（整合性の考え方）
1. Receive: Professor がファイル受領（user_id/subject_id を確定）
2. Upload: GCSへ保存（gcs_uri と checksum を確定）
3. Produce: Kafkaへ `IngestJob` を publish
4. Consume: ワーカーが consume
5. Ingestion（Vision→Chunks）: Gemini 3 Flash（Structured Outputsで `chunks[]` を生成。Summaryは原則なし）
6. Store: Postgresへ永続化

## 冪等（Idempotency）（MUST）
Kafka は at-least-once を前提にするため、同一ジョブが複数回実行されても結果が増殖しないようにする。

- 推奨: 原本由来の冪等キーを持つ
  - 例: `(user_id, subject_id, source_checksum, derivative_version)`
- “ジョブを冪等にする”よりも、**結果の永続化を冪等にする**ことを優先する

## 不整合の検知と復旧
- 検知例
  - GCSに原本はあるが、DBに派生がない
  - materialはあるが embedding が欠けている
- 復旧原則
  - `IngestJob` を再投入して再生成する（DLQ/リドライブ前提）

## 失敗時の原則
- DLQ（デッドレター）を用意し、手動介入で復旧できること
- 再処理時も subject_id/user_id の物理制約を破らないこと

## 関連
- `03_integration/EVENT_CONTRACTS.md`
- `05_operations/SLO_ALERTING.md`
