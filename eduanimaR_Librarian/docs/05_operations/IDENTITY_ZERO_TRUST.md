# IDENTITY_ZERO_TRUST

## 目的
マイクロサービス環境で「誰が誰と通信しているか」を**強く**証明し、
横移動・秘密情報漏えい・設定ミスによる侵害拡大を防ぐ。

本書は **Zero Trust（ネットワークを信用しない）** を前提に、
サービス間（workload-to-workload）通信の identity / 認証 / 認可 / 運用を標準化する。

## 適用範囲
- Professor ↔ Librarian（HTTP/JSON）
- Librarian ↔ Gemini API（HTTPS）

## 用語
- **End-user identity**: 人（ユーザー）を表すID（JWTのsub等）
- **Workload identity**: サービス/ジョブなど実行主体を表すID（例: SPIFFE ID）
- **mTLS**: 相互TLS（client/server双方が証明書で認証）
- **Attestation**: ワークロードが「期待された環境/形」で動いていることの証明

## 原則（MUST）
- **サービス間通信は“ネットワーク境界”ではなく“Identity”で守る**（Zero Trust）。
- **Service-to-service 認証を必須**（mTLS など）。共有APIキーや固定トークンは原則禁止。
- **短命クレデンシャル**（数分〜数時間）を基本とし、ローテーションを自動化する。
- **認可は明示**する（どの呼び出し元が、どの操作を呼べるか）。
- **最小権限**（サービス単位・メソッド単位）を基本とし、例外はレビューで記録する。
- **相関IDを伝播**（`request_id` / `trace_id`）し、監査と調査可能性を担保する。

## 参照アーキテクチャ（推奨）
### A. SPIFFE/SPIRE による Workload Identity（推奨）
- 各ワークロードに **SPIFFE ID**（例: `spiffe://example.internal/ns/prod/sa/order`）を付与
- SPIRE が attestation に基づき SVID（X.509 証明書）を発行
- サービス間通信は mTLS（x509）で相互認証

> Service Mesh を採用する場合、Envoy/Istio などに委譲してもよいが、
> 本質は「ワークロード identity + mTLS + 認可」が成立していること。

### B. Mesh なしの最小構成（現実解）
- SPIRE もしくは同等機構で x509 を自動配布
- HTTP クライアント/サーバで mTLS を設定
- 認可は middleware 等で workload identity（SPIFFE ID 等）を取り出して判定

## 認証（Authentication）
### mTLS（MUST）
- client は**自分の証明書**を提示し、server はそれを検証する
- server も証明書を提示し、client はそれを検証する

#### 証明書運用（MUST）
- CA/中間CAのローテーション計画を持つ
- 失効（revocation）の扱いを決める（短命証明書で失効依存を下げるのが一般的）
- 開発環境でも「自己署名固定」を避け、可能な限り同じ流れで回す

### JWT-SVID（任意/補助）
- サービス間で HTTP を使う場合や、mTLS終端が別レイヤにある場合に補助として利用
- “署名済みの短命トークン”として扱い、固定の共有シークレットに戻らない

## 認可（Authorization）
### 責務分界（推奨）
- **Gateway**: end-user の認証/大枠の認可（ロール/テナント等）
- **各サービス**: workload identity を前提に、
  - 呼び出し元サービスの許可
  - メソッド単位の権限
  - 業務上の所有者チェック/状態遷移チェック（BOLA/BFLA）

### 最小ポリシー（テンプレ）
- `api-gateway` → `user` の `UserService/*` は許可
- `api-gateway` → `order` の `OrderService/CreateOrder` は許可
- `product` → `order` の直呼びは原則禁止（必要時は明示し、監査ログ対象）

> 実装は OPA 等の policy engine を使ってもよいし、
> 静的な allowlist（サービス×メソッド）から始めてもよい。

## 実装ガイド（HTTP）
### Server 側（推奨）
- mTLS を有効化
- client の ID（SPIFFE ID 等）を取得
- middleware で「許可された呼び出し元か」を検査

### Client 側（推奨）
- 常に mTLS を使う
- timeout/cancellation を伝播（関連: `01_architecture/RESILIENCY.md`）

## 監査・観測（MUST）
- すべての s2s リクエストで以下をログ/トレースに付与
  - `request_id` / `trace_id`
  - `source_workload_id`（SPIFFE ID 等）
  - `destination_service` / `http.route`
  - 認可結果（allow/deny）

> 監査ログの詳細は `05_operations/AUDIT_LOGGING.md`。

## 例外と移行
### 段階導入（推奨）
1. まずは mTLS を導入（認証）
2. 次に allowlist を導入（認可）
3. 最後にポリシー as code / 自動検証

### 例外（SHOULD NOT）
- 固定APIキー、長寿命トークン、手配布証明書
- 例外が必要なら期限つきで記録し、置き換え計画を作る

## 関連
- `03_integration/INTER_SERVICE_COMM.md`
- `05_operations/API_SECURITY.md`
- `05_operations/SECRETS_KEY_MANAGEMENT.md`
- `05_operations/AUDIT_LOGGING.md`
- `01_architecture/RESILIENCY.md`
