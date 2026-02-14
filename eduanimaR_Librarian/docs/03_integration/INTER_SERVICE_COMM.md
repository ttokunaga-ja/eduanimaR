# INTER_SERVICE_COMM

## 目的
フロントエンド（Next.js BFF）、API Gateway、マイクロサービス間の通信規約を定義し、結合度を制御する。

## 2段階ゲートウェイ構成（確定）
```
Browser → Next.js (BFF) → Go API Gateway → Go Microservices
```

### 1) Browser ↔ Next.js (BFF)
- プロトコル: HTTP/JSON
- 認証: Cookie/Session (Next.js が管理)
- 責務:
  - Next.js が UI 向けにデータを整形・集約
  - 複数の Go API を呼び出して結果をマージする場合あり
  - RSC (React Server Components) で初期データをサーバー側で取得

### 2) Next.js (BFF) ↔ Go API Gateway
- プロトコル: HTTP/JSON
- 認証: JWT Bearer Token (Next.js が Cookie から取り出して付与)
- ルール:
  - タイムアウト必須（Next.js 側で設定）
  - エラーハンドリング: `ERROR_CODES.md` に従う
  - リトライは呼び出し側（Next.js）で方針を統一（指数バックオフ等）

### 3) Go API Gateway ↔ Go Microservices (内部通信)
- **プロトコル: gRPC (基本)**
  - 型安全、高速、双方向ストリーミング対応
  - Proto定義(.proto)がサービス間契約のSSOTとなる
  - HTTP/JSONは例外的な用途のみ(レガシー連携等)
- **Gateway の役割: gRPC → OpenAPI 変換**
  - 内部マイクロサービスはgRPCで実装
  - Gateway が gRPC サービスを呼び出し、結果をHTTP/JSON(OpenAPI形式)に変換してフロントエンドへ返す
  - 変換ツール: grpc-gateway, connectrpc 等を活用
- 認証: JWT 検証は Gateway で完了しているため、内部は **service-to-service 認証を必須** とする（mTLS / workload identity）
- ルール:
  - タイムアウト必須（無限待ち禁止）
  - リトライは慎重に（冪等性を確認）
  - 冪等性が必要な操作は idempotency key を導入

> service-to-service 認証/認可/運用の標準は `05_operations/IDENTITY_ZERO_TRUST.md` を参照。

## 認可（重要）
- Gateway での認証/認可に加え、**usecase（業務層）での所有者チェック/状態遷移チェック** を必須とする（BOLA/BFLA 対策）。
- 詳細: `05_operations/API_SECURITY.md`

## レジリエンス（重要）
- timeout/retry/idempotency の横断ルールは分散させず、`01_architecture/RESILIENCY.md` を正とする。

## 非同期（イベント）
- 原則: 状態変化の伝搬はイベントで行う（Debezium CDC / Kafka）
- ルール:
  - スキーマ互換性（後方互換）を維持
  - consumer は冪等に実装

## 契約
- **外部API契約 (Gateway → Frontend)**: OpenAPI(`API_CONTRACT_WORKFLOW.md`)
  - Go API Gateway が公開するHTTP/JSON仕様
  - Next.js (Orval) で型生成し、フロントエンドで使用
- **内部API契約 (Gateway → Microservices)**: Protocol Buffers (.proto)
  - gRPCサービス定義が内部通信の正
  - `protoc` でGoコードを生成
  - Gateway が .proto から OpenAPI へ変換する際は、grpc-gateway の annotations を活用
- イベント契約: スキーマ定義（Avro/JSON Schema等）をSSOTにする（方式はプロジェクトで決定）

> イベント契約・DLQ・再処理の標準は `03_integration/EVENT_CONTRACTS.md` を参照。

## 明確に「やらない」こと
- **Browser から Go Microservices への直接アクセス**（内部構造の露出を招く）
- **Go API Gateway にビジネスロジック** を書く（Gateway は認証/認可/ルーティングに徹する）

## 関連
- `01_architecture/RESILIENCY.md`
- `03_integration/EVENT_CONTRACTS.md`
- `05_operations/API_SECURITY.md`
- `05_operations/IDENTITY_ZERO_TRUST.md`
