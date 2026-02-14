# INCIDENT_POSTMORTEM

## 目的
障害対応を属人化させず、
- 早く復旧し
- 再発を防ぎ
- 学びを組織に残す
ための最小プロセスを定義する。

## 用語
- **Incident**: ユーザー影響/重大なリスクを伴う事象
- **SEV**: 重大度（Severity）
- **Postmortem**: 事後検証（責任追及ではなく再発防止）

## 原則（MUST）
- “犯人探し” をしない（Blameless）
- “症状” と “原因” を分けて記録する
- アクションはオーナーと期限を持ち、完了まで追う

## 重大度（テンプレ）
- SEV1: 主要機能が広範に停止、売上/法令/信用に重大影響
- SEV2: 一部機能の停止/大幅劣化、回避策あり
- SEV3: 限定的な影響、短時間で復旧

> 実際の基準はプロダクト要件に合わせる。

## 体制（推奨）
- Incident Commander（指揮）
- Comms（対外/社内連絡）
- Ops（復旧作業）
- Scribe（記録係）

## 初動（MUST）
1. 重大度判定（SEV）
2. 影響範囲の特定（どのユーザー/機能）
3. 直近変更の確認（デプロイ/マイグレーション）
4. 緩和策の実行（レート制限、機能切離し、ロールバック）
5. 監視/ログ/トレースで原因候補を絞る

> SLO/アラート運用は `05_operations/SLO_ALERTING.md`。

## 連絡（MUST）
- 連絡チャネルと責任者を決めておく
- 顧客影響がある場合は、
  - 現状
  - 影響
  - 次回更新時刻
  を定期更新する

## 復旧後（MUST）
- 影響が収束したことを確認
- 一時対処（mitigation）を恒久対応へ繋げる計画を立てる

## Postmortem テンプレ（最小）
- 概要（何が起きたか）
- タイムライン（いつ何が起きたか）
- 影響（ユーザー/金額/データ）
- 検知（どう検知したか、遅れはあったか）
- 原因（直接原因/根本原因）
- 何がうまくいったか
- 何がうまくいかなかったか
- アクション（owner/期限/優先度）

## 代表的アクション例
- 監視: SLI/SLO の追加、アラート閾値の調整
- リリース: カナリア導入、ロールバック自動化（関連: `05_operations/PROGRESSIVE_DELIVERY.md`）
- 設計: タイムアウト/リトライの統一（関連: `01_architecture/RESILIENCY.md`）
- セキュリティ: secret ローテ手順整備（関連: `05_operations/SECRETS_KEY_MANAGEMENT.md`）

## 関連
- `05_operations/SLO_ALERTING.md`
- `05_operations/OBSERVABILITY.md`
- `05_operations/PROGRESSIVE_DELIVERY.md`
- `05_operations/DATA_PROTECTION_DR.md`
- `05_operations/VULNERABILITY_MANAGEMENT.md`
- `05_operations/SUPPLY_CHAIN_SECURITY.md`
- `05_operations/AUDIT_LOGGING.md`
- `05_operations/SECRETS_KEY_MANAGEMENT.md`
