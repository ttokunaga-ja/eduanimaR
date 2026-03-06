# OBSERVABILITY

## 目的
障害時の「原因特定までの時間」を短縮するため、ログ/メトリクス/トレースの最低限の標準を定義する。

## SLO/アラート（推奨）
観測したデータを「運用の意思決定」に繋げるため、SLO とアラート設計を別紙で定義する。
- 参照: `05_operations/SLO_ALERTING.md`

## ログ（必須）
- 言語標準のロガー（および採用済みのログ方針）に統一する
- すべてのリクエストに `request_id` を付与し、ログに含める
- 可能なら `trace_id` も付与（導入方式はプロジェクトで統一）
- ルール:
	- PII（個人情報）を出さない
	- SQLやクレデンシャルを生ログしない

## トレース（推奨）
- OpenTelemetry を標準とし、サービス間（HTTP）で trace context を伝播する
- Span の最小要件:
	- Librarian（HTTP handler）: 入口 span
	- Librarian→Professor（HTTP client）: client span
	- Librarian→Gemini（HTTPS client）: client span
- 属性は OpenTelemetry の semantic conventions（特に RPC/HTTP/DB）に寄せる

## Semantic Conventions（推奨）
ダッシュボード/検索/相関の品質は「属性名の統一」で決まるため、
OpenTelemetry Semantic Conventions を実装のSSOTとして扱う。

### 最小要件（SHOULD）
- HTTP: ルート/メソッド/ステータス/URL属性
- （必要なら）外部依存（Professor / Gemini）の種別・結果
- Feature flag: 評価結果（フラグ名/バリアント）

> 具体の属性名は OTel の semantic conventions を正とする。

## メトリクス（推奨）
- SLI として最低限、以下を収集する（粒度は `service` / `method` / `status` を基本）:
	- リクエスト数（RPS）
	- レイテンシ（p50/p95/p99）
	- エラー率（HTTP status / timeout）
	- 依存先（Professor / Gemini）の失敗率・レイテンシ

## エラー監視
- 5xx は原則アラート対象
- 4xx の急増は仕様変更/攻撃/不具合のシグナルとして監視

## 相関（ログ↔トレース）
- `request_id` と `trace_id` をログに必ず出し、障害時に相互に辿れるようにする
- 上流（Professor）で生成/受領した `request_id` は下流へ伝播する（HTTP header）

## 参考（一次情報）
- OpenTelemetry Python: https://opentelemetry.io/docs/languages/python/
- OpenTelemetry semantic conventions: https://opentelemetry.io/docs/specs/semconv/

## 関連
- `05_operations/SLO_ALERTING.md`
- `05_operations/DATA_PROTECTION_DR.md`
- `05_operations/AUDIT_LOGGING.md`

