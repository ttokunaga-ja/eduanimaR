# PROGRESSIVE_DELIVERY

## 目的
変更のリスクをコントロールし、
SLO を守りながら安全にリリースするための段階的デリバリー標準を定義する。

## 適用範囲
- Professor（外向きHTTP API / SSE）
- Professor の非同期処理（Kafka consumer / worker）
- Professor → Librarian（gRPC client 依存）
- DB マイグレーション（関連: `05_operations/MIGRATION_FLOW.md`）

## 原則（MUST）
- **小さく出す**: 変更は小さく分割して出す
- **観測できない変更は出さない**: 事前に SLIs を用意する
- **ロールバック可能性**を設計に含める（アプリだけでなくDBも）
- **Error Budget を使う**: 失敗許容量が枯渇しているならリリースを止める

## デプロイ戦略
### 1) Rolling（最小）
- 基本のローリング更新
- リスクが高い変更には不向き（影響が一気に広がる）

### 2) Blue/Green（推奨）
- 旧環境（Blue）と新環境（Green）を並行稼働
- 切替は一度だが、切替前に検証できる

### 3) Canary（推奨）
- 新バージョンへ段階的にトラフィックを流す
- 例: 1% → 10% → 50% → 100%

### 4) Feature Flag（推奨）
- デプロイと機能公開を分離
- 事故時は「フラグOFF」で緊急緩和できる

> ただし、フラグは負債化しやすい。期限と削除計画を必須にする。

## リリース前の必須条件（MUST）
- 監視:
  - エラー率（HTTP 5xx / gRPC status）
  - p95/p99 レイテンシ
  - 依存先（DB/Kafka/GCS/Librarian gRPC）の失敗率/レイテンシ
- 相関:
  - `request_id` / `trace_id` が追える
- ロールバック:
  - アプリのロールバック手順がある
  - DB は expand/contract に沿っている（下記）

## SSE を含むリリース注意点（MUST）
- 切替時の挙動を定義する:
  - 既存 SSE 接続は切断され得る（再接続・再送設計はクライアント契約に含める）
  - シャットダウン時は可能なら graceful に close し、処理中のリクエストをキャンセル伝播する
- 監視で追う:
  - 同時接続数、切断率、再接続率
  - ストリーム処理のエラー率（ハンドラ内部の失敗/コンテキストキャンセル）

## 自動ロールバック（推奨）
- カナリア中は SLO/SLI をゲートにする
  - 例: 5分窓で 5xx が閾値超過 → 即ロールバック
  - 例: p99 が基準より悪化 → 停止
- “原因候補（DB遅延など）”ではなく、まずは“症状（ユーザー影響）”で止める

## DB 互換（MUST）: Expand/Contract
破壊的変更を安全に出すため、スキーマ変更は段階的に行う。

### パターン（例）
1. Expand: 新カラム追加（NULL可）
2. アプリ: 新旧両方を読み書き（互換期間）
3. Backfill: 旧データを埋める
4. Contract: 旧カラム削除、制約強化

> Atlas の運用は `05_operations/MIGRATION_FLOW.md`。

## 非同期（Kafka consumer / worker）のリリース
- 互換性:
  - イベントスキーマは後方互換（フィールド追加は任意、既存 consumer が壊れない）
- カナリア:
  - consumer グループを分ける、もしくは一部パーティションのみ処理するなど段階導入
- 監視:
  - DLQ 増加、処理遅延（lag）、リトライ増加をゲートにする

## リリースチェックリスト（テンプレ）
- 変更種別: API/DB/イベント/インフラ
- 互換性: 旧クライアント/旧consumerが動くか
- 監視: 追加した指標が見えるか
- ロールバック: 何を戻せば復旧するか
- 連絡: 影響がある場合の告知

## 関連
- `05_operations/RELEASE_DEPLOY.md`
- `05_operations/SLO_ALERTING.md`
- `05_operations/OBSERVABILITY.md`
- `05_operations/MIGRATION_FLOW.md`
- `03_integration/EVENT_CONTRACTS.md`
- `04_testing/PERFORMANCE_LOAD_TESTING.md`
- `05_operations/INCIDENT_POSTMORTEM.md`
