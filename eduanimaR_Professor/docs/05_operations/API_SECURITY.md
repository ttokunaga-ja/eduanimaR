# API_SECURITY

## 目的
API を安全に提供するための **最低限の設計・実装・運用ルール** を定義する。
OWASP API Security Top 10 の観点を、日々の設計/レビュー/実装に落とし込む。

## 適用範囲
- Browser ↔ Next.js(BFF)
- Next.js(BFF) ↔ Go API Gateway
- Go API Gateway ↔ Go Microservices（gRPC）

## 基本原則（MUST）
- **認可は二段構え**
  - Gateway: 認証・大枠の権限（ロール/スコープ/テナント）
  - Usecase: **最終的な所有者/状態遷移/業務フローの正当性** を検証（BOLA/BFLA の最後の防波堤）
- **入力は必ず検証**（型・範囲・長さ・列挙・正規化）し、拒否時は共通エラー形式に統一する（関連: ERROR_HANDLING）。
- **リソース消費を必ず制限**（タイムアウト/ページング上限/レート制限/最大ボディサイズ）。
- **機微情報（PII/秘密情報）をログに出さない**（関連: OBSERVABILITY）。
- **公開 API の棚卸し（インベントリ）** を維持し、未使用/非推奨を計画的に廃止する（関連: API_CONTRACT_WORKFLOW）。

## チェックリスト（レビューで使う）

### A. 認可（BOLA / BFLA / Property-level）
- オブジェクト参照（`/users/{id}` 等）で **呼び出し主体がそのオブジェクトへアクセス可能か** を usecase で検証している
- 一覧系（検索/フィルタ）で **テナント境界** が常に適用される（クエリ条件の入れ忘れを防ぐ）
- 機能単位（管理API/運用API/高権限操作）で **ルートごとに権限要件** が明確
- 更新系で **プロパティ単位の書き換え可否** を明確化（Mass Assignment 対策）

### B. 認証
- トークンの検証（署名/期限/issuer/audience）を統一
- セッション/トークン失効（ログアウト、権限変更、退会など）を考慮
- 内部通信は service-to-service 認証（mTLS / workload identity）を必須化（関連: IDENTITY_ZERO_TRUST / INTER_SERVICE_COMM）

### C. リソース消費（Unrestricted Resource Consumption）
- すべての外部 I/O に timeout がある（HTTP/gRPC/DB/ES/Kafka）
- ページングはデフォルト値と上限を持つ（例: `limit <= 100`）
- ソート/フィルタは許容リスト方式（任意フィールド指定を許さない）
- アップロードや大きいリクエストは **最大サイズ** を決める

### D. セキュリティ設定ミス（Misconfiguration）
- CORS/CSRF/セキュリティヘッダの責務（BFF or Gateway）を決めている
- 本番で debug エンドポイント、reflection（gRPC）、詳細エラーを無効化
- 依存先接続文字列/秘密情報がログに出ない

### E. SSRF / Unsafe Consumption
- 外部 URL を受け取る場合は allowlist（ドメイン/スキーム）で制限する
- 3rd party API 呼び出しはタイムアウト・リトライ・サーキットブレーカ方針がある（関連: RESILIENCY）

### F. API インベントリ（Improper Inventory Management）
- OpenAPI が外部公開の SSOT として維持され、非推奨/廃止が追跡可能
- 影響範囲（利用クライアント/運用ジョブ/バッチ）を把握できる

## 推奨のドキュメント連携
- エラー形式: 03_integration/ERROR_HANDLING.md / 03_integration/ERROR_CODES.md
- 契約変更: 03_integration/API_CONTRACT_WORKFLOW.md
- レジリエンス: 01_architecture/RESILIENCY.md
- 観測性: 05_operations/OBSERVABILITY.md

## 関連
- `03_integration/API_VERSIONING_DEPRECATION.md`
- `03_integration/CONTRACT_TESTING.md`
- `03_integration/INTER_SERVICE_COMM.md`
- `05_operations/IDENTITY_ZERO_TRUST.md`
- `05_operations/SECRETS_KEY_MANAGEMENT.md`
- `05_operations/AUDIT_LOGGING.md`
- `05_operations/VULNERABILITY_MANAGEMENT.md`
