# AUDIT_LOGGING

## 目的
「誰が・いつ・何をしたか」を後から検証できる状態を作り、
不正/事故/規制対応/内部統制に耐える監査ログ標準を定義する。

監査ログは “デバッグログ” や “アプリログ” と目的が異なる。

## 適用範囲
- 外部API（Gateway）
- 内部API（gRPC）
- 管理操作（運用API/管理画面）
- 重要データの参照・更新

## 監査ログに向くイベント（MUST）
- 認証/認可
  - ログイン成功/失敗
  - 権限変更
  - トークン失効/セッション破棄
- 重要操作（業務フロー）
  - 決済、注文確定、返金
  - 個人情報の参照/更新/削除
  - 設定変更（レート制限、フラグ変更 等）
- 管理者操作
  - ユーザー凍結/解除
  - ロール付与

## 最小フィールド（MUST）
- `occurred_at`: 発生時刻（UTC推奨）
- `action`: 操作名（例: `order.create`, `user.role.grant`）
- `result`: 成功/失敗
- `actor`:
  - end-user identity（`user_id` 等）
  - workload identity（`source_workload_id` 等）
- `target`:
  - 対象リソース種別/ID（例: `order_id`）
- `request_id` / `trace_id`
- `source_ip`（外部操作の場合）

### 禁止（MUST NOT）
- PII の丸ごと記録（氏名、住所、決済情報など）
- 認証情報（パスワード、トークン、APIキー）

## 形式
- JSON など機械可読形式を推奨
- `action` は安定IDとして扱い、変更時は移行方針を決める

## 改ざん耐性（MUST）
- 監査ログは **書き込み専用** に寄せる
- ログストレージのアクセス権を最小化（閲覧は限定）
- 保持期間（retention）と削除ルールを明確にする

> “完全な不変（immutable）” が必要な場合は、WORM ストレージ等を検討。

## 収集/保管（推奨）
- アプリログと分けてもよい（重要度が高い）
- バックアップとリストア手順を用意（関連: `05_operations/DATA_PROTECTION_DR.md`）

## 実装ポイント
- 監査ログは “失敗しても処理を止めない” かどうかを操作ごとに決める
  - 決済など最重要は「監査ログが書けないなら失敗」にしてもよい
- 監査ログ出力もレート制限/バッファ等で過負荷対策する

## 検証（推奨）
- 重要操作に対して監査ログが必ず出るテスト
- actor/target/request_id が揃うこと

## 関連
- `05_operations/OBSERVABILITY.md`
- `05_operations/IDENTITY_ZERO_TRUST.md`
- `05_operations/API_SECURITY.md`
- `05_operations/DATA_PROTECTION_DR.md`
