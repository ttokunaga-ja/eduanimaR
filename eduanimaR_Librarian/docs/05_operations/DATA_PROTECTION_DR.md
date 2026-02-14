# DATA_PROTECTION_DR

## 目的
データ保護（PII/秘密情報/保持/削除）と、障害・災害時の復旧（DR）方針を最小限定義する。

## RPO / RTO（定義必須）
- **RPO**: どこまでのデータ損失を許容するか
- **RTO**: どれくらいで復旧させる必要があるか

## バックアップ（MUST）
Librarian は DB-less / stateless であり、バックアップ対象となる永続データを持たない。

ただし、以下はデータ保護の対象として扱う:
- 設定/シークレット（APIキー等）
- 監査ログや運用ログの保管（ログ基盤側の保持/削除）

## PITR（Point-in-Time Recovery）（推奨）
（Not Applicable）

## データ保持・削除（MUST）
- PII を含むデータの保持期間を決める
- 削除（論理/物理）方針を決める
  - 監査・会計要件がある場合は、匿名化/トークナイズ等も検討

## ログ/トレースと機微情報
- request_id/trace_id など相関に必要な情報は出す
- ただし PII/秘密情報は出さない（関連: OBSERVABILITY）

## DR（推奨）
Librarian の DR は「再デプロイで復旧できる」前提で設計する。

- 依存先（Professor / Gemini）が部分失敗しても、Librarian が安全に劣化できること
- リージョン障害時の再デプロイ手順（IaC/手順/責任者）

## 関連
- 05_operations/OBSERVABILITY.md
- 05_operations/RELEASE_DEPLOY.md
- 05_operations/SECRETS_KEY_MANAGEMENT.md
- 05_operations/INCIDENT_POSTMORTEM.md
