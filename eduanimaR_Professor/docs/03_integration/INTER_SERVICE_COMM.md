# INTER_SERVICE_COMM

## 目的
Frontend ↔ Professor（Go）および Professor ↔ Librarian（推論サービス）の通信規約を定義し、結合度を制御する。

## サービス境界（確定）
```
Frontend ↔ Professor ↔ Librarian
[HTTP/JSON + SSE]  [gRPC]
[OpenAPI SSOT]     [Proto SSOT]
```

### 1) Frontend ↔ Professor（外向き）
- プロトコル: HTTP/JSON
- ストリーミング: SSE（必須）
- 契約（SSOT）: `docs/openapi.yaml`
- ルール:
  - タイムアウト必須（無限待ち禁止）
  - リトライは冪等性が確認できる操作のみに限定する
  - SSE は途中経過も含めて契約として扱う（イベント名/形を安定させる）

### 2) Professor ↔ Librarian（内部）
- プロトコル: gRPC
- 契約（SSOT）: `proto/` 配下の `.proto`
- ルール:
  - すべての RPC に deadline を設定し、キャンセルを伝播する
  - リトライは慎重に（冪等性を確認）
  - Librarian は DB/GCS へ直接アクセスしない（Professor がデータの守護者）
- 認証: service-to-service 認証を必須（mTLS / workload identity 等）

### 3) Kafka（非同期）
- 目的: IngestJob 等の非同期処理（OCR/構造化/Embedding準備）
- 契約: `EVENT_CONTRACTS.md` を正とする
- ルール:
  - 原則 at-least-once を前提に consumer は冪等に実装する
  - DLQ / 再処理手順を用意する

> service-to-service 認証/認可/運用の標準は `05_operations/IDENTITY_ZERO_TRUST.md` を参照。

## 認可（重要）
- **usecase（業務層）での所有者チェック/状態遷移チェック** を必須とする（BOLA/BFLA 対策）。
- 詳細: `05_operations/API_SECURITY.md`

## レジリエンス（重要）
- timeout/retry/idempotency の横断ルールは分散させず、`01_architecture/RESILIENCY.md` を正とする。

## 非同期（イベント）
- 原則: 状態変化の伝搬はイベントで行う（Kafka）
- ルール:
  - スキーマ互換性（後方互換）を維持
  - consumer は冪等に実装

## 契約
- **外部API契約（Frontend ↔ Professor）**: OpenAPI（`API_CONTRACT_WORKFLOW.md`）
- **内部API契約（Professor ↔ Librarian）**: Protocol Buffers（`PROTOBUF_GRPC_STANDARDS.md`）
- イベント契約: スキーマ定義（Avro/JSON Schema等）をSSOTにする（方式はプロジェクトで決定）

> イベント契約・DLQ・再処理の標準は `03_integration/EVENT_CONTRACTS.md` を参照。

## 明確に「やらない」こと
- Frontend から Librarian への直接アクセス
- Librarian が DB/GCS へ直接アクセス
- サービス間でアドホックな JSON を握る（必ず OpenAPI / Proto / スキーマに寄せる）

## 関連
- `01_architecture/RESILIENCY.md`
- `03_integration/EVENT_CONTRACTS.md`
- `05_operations/API_SECURITY.md`
- `05_operations/IDENTITY_ZERO_TRUST.md`
