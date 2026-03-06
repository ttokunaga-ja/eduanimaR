# DATA_PROTECTION_DR

## 目的
データ保護（PII/秘密情報/保持/削除）と、障害・災害時の復旧（DR）方針を最小限定義する。

## RPO / RTO（定義必須）
- **RPO**: どこまでのデータ損失を許容するか
- **RTO**: どれくらいで復旧させる必要があるか

## バックアップ（MUST）
- 定期バックアップを取得する（フル + 増分/PITR は要件に応じて）
- バックアップは暗号化し、アクセス権を最小化する
- **復元テスト（リストア演習）** を定期実施し、手順を Runbook 化する

## PITR（Point-in-Time Recovery）（推奨）
- 誤操作/不正更新に備え、可能なら PITR を用意する

## データ保持・削除（MUST）
- PII を含むデータの保持期間を決める
- 削除（論理/物理）方針を決める
  - 監査・会計要件がある場合は、匿名化/トークナイズ等も検討

## ログ/トレースと機微情報
- request_id/trace_id など相関に必要な情報は出す
- ただし PII/秘密情報は出さない（関連: OBSERVABILITY）

## DR（推奨）
- 依存コンポーネント（DB/ES/Kafka）ごとに復旧手順を用意
- フェイルオーバ/リージョン切替の手順と責任者を決める

## 関連
- 05_operations/OBSERVABILITY.md
- 05_operations/RELEASE_DEPLOY.md
- 05_operations/MIGRATION_FLOW.md
- 05_operations/SECRETS_KEY_MANAGEMENT.md
- 05_operations/INCIDENT_POSTMORTEM.md
