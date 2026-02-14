# PROGRESSIVE_DELIVERY

## 目的
変更のリスクをコントロールし、
SLO を守りながら安全にリリースするための段階的デリバリー標準を定義する。

## 適用範囲
- Librarian（HTTP API）
- 依存先（Professor / Gemini）を含むエラー率/レイテンシ

## 原則（MUST）
- **小さく出す**: 変更は小さく分割して出す
- **観測できない変更は出さない**: 事前に SLIs を用意する
- **ロールバック可能性**を設計に含める（アプリ/設定/契約）
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
  - エラー率（HTTP 5xx / timeout）
  - p95/p99 レイテンシ
  - 依存先（Professor / Gemini）の失敗率/レイテンシ
- 相関:
  - `request_id` / `trace_id` が追える
- ロールバック:
  - アプリのロールバック手順がある
  - DB は expand/contract に沿っている（下記）

## 自動ロールバック（推奨）
- カナリア中は SLO/SLI をゲートにする
  - 例: 5分窓で 5xx が閾値超過 → 即ロールバック
  - 例: p99 が基準より悪化 → 停止
- “原因候補（DB遅延など）”ではなく、まずは“症状（ユーザー影響）”で止める

## データ/同期の扱い
Librarian は DB-less のため、DB スキーマ変更・CDC・Indexer 等の段階リリースは Professor 側の責務として扱う。

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
- `04_testing/PERFORMANCE_LOAD_TESTING.md`
- `05_operations/INCIDENT_POSTMORTEM.md`

