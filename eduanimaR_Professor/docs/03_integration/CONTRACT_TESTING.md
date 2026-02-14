# CONTRACT_TESTING

## 目的
契約（OpenAPI / Proto / Event schema）が破壊されるのを、
レビューの目視だけに頼らず **CI で機械的に検出**する。

## 対象
- 外向き契約: OpenAPI（Gateway → BFF/Frontend）
- 内部契約: Protocol Buffers（Gateway ↔ Microservices）
- 非同期契約: Event schema（Kafka）

## 原則（MUST）
- 契約は SSOT から生成し、生成物の手編集を禁止
- “互換性” を breaking check で自動判定する
- 破壊的変更は、バージョン分離/移行期間/告知までセットで扱う

## OpenAPI
### 1) 生成差分検出（MUST）
- `openapi.yaml` の再生成で差分が出ないこと
  - 目的: 手編集や生成漏れの検出

### 2) Breaking change 検出（SHOULD）
- `main`（または前回リリース）と比較して breaking を検出
- 破壊的変更は、`API_VERSIONING_DEPRECATION.md` の手順に従う

### 3) 契約駆動のスモーク（推奨）
- OpenAPI からリクエストを生成し、
  - 200系
  - 4xx（バリデーション）
  - 5xx（想定外）
  を最低限確認する

## Proto（gRPC）
### 1) lint（MUST）
- 命名規則、フィールド番号など

### 2) breaking（MUST）
- `main`（またはリリースタグ）と比較して breaking を検出
- breaking が必要なら、サービス/メッセージのバージョン戦略を取る

### 3) 生成差分検出（MUST）
- `buf generate` / `protoc` の結果がコミット内容と一致する

## Event schema（Kafka）
### 互換性ルール（MUST）
- 追加は任意フィールドとして扱う
- 削除/型変更/意味変更は breaking

### Breaking の扱い
- トピック名/イベント名をバージョン分離し、移行期間を設ける

## CI への組み込み（例）
- Stage: Contract
  - OpenAPI: 再生成差分 + breaking check
  - Proto: lint + breaking + 生成差分
  - Event: schema compatibility check

> 具体ツール（buf / openapi-diff / schema registry 等）はプロジェクトで決める。
> 本テンプレは“何を検査すべきか”を標準化する。

## 関連
- `03_integration/API_CONTRACT_WORKFLOW.md`
- `03_integration/API_VERSIONING_DEPRECATION.md`
- `03_integration/PROTOBUF_GRPC_STANDARDS.md`
- `03_integration/EVENT_CONTRACTS.md`
- `05_operations/CI_CD.md`
