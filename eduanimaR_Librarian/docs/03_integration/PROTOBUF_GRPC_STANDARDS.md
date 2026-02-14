# PROTOBUF_GRPC_STANDARDS

## 目的
内部契約（.proto / gRPC）を **安全に進化** させ、Gateway の gRPC→OpenAPI 変換とフロントの自動生成を破綻させないための標準を定義する。

本ドキュメントは以下を対象とする:
- Gateway ↔ Microservices の gRPC API
- gRPC→HTTP/JSON(OpenAPI) 変換（grpc-gateway 等）
- 互換性・生成・CI でのガード

## SSOT
- 内部API契約の正: `.proto`
- 外部API契約の正: `openapi.yaml`（Gateway が生成・公開するもの）

## 本プロジェクトの統合方針（前提）
本標準は、以下のアーキテクチャ方針を前提にする。

- Database per Service: 各マイクロサービスが自身の DB を所有する
- 単一オーナーシップ: 各テーブルの書き込み権限は当該サービスのみ（他サービスは API / Kafka イベント経由）
- イベント駆動統合: サービス間の協調は Kafka イベントで実現する（同期 gRPC は補助）
- 最終整合性: ドメインサービス間の同期は Eventual Consistency を基本とする
  - 例外: 金銭取引（ウォレット等）は強整合性を維持する
- 責務分離の徹底: サービスは API / イベントで連携し、責務範囲を明確に分離する
- 固定値: PostgreSQL ENUM を積極採用し、型安全性・パフォーマンス・可読性を向上させる
- ID: 内部主キーは UUID、外部公開キーは NanoID を採用する

## gRPC の責務（同期 API を使う条件）
Kafka による非同期イベント連携を基本としつつ、gRPC は以下の用途に限定して用いる。

### gRPC を使う（SHOULD）
- 強整合性が必要な操作（例: ウォレットの残高更新、二重計上を避ける決済フロー）
- リクエスト/レスポンスで即時結果が必要な問い合わせ（例: 認可チェック、表示に必須な参照）
- 失敗時に呼び出し側がその場でリカバリ/ユーザー通知を行う必要がある操作

### gRPC を避ける（SHOULD）
- 「他サービスの状態反映・同期」を目的とした呼び出し（同期カップリングを招く）
  - 代替: Kafka のドメインイベント購読 + ローカル Read Model 更新

## イベント連携（Kafka）を前提とした設計ルール
本ドキュメントは gRPC/.proto を主対象とするが、イベント（Kafka）も互換性と運用性が最重要となる。

### 契約とオーナーシップ（MUST）
- イベントは「状態の正」を公開するものではなく、「状態変化の通知」として設計する
- Producer がイベントのスキーマ・意味を所有し、Consumer は後方互換を前提に実装する
- Consumer は必ず冪等に実装する（少なくとも at-least-once を前提）

### 互換性（MUST）
- イベントスキーマも後方互換を維持する
  - フィールド追加は基本許容（任意フィールドとして）
  - 既存フィールドの削除/型変更/意味変更は破壊的
- 破壊的変更が必要な場合はトピック/イベント名/バージョンを分ける（例: `user.profile.v1` → `user.profile.v2`）

### 相関（SHOULD）
- gRPC リクエスト→イベント発行がある場合、相関可能な ID を運ぶ（例: `request_id`, `trace_id`）
- イベントには一意な `event_id` を持たせる（冪等処理・重複排除のキー）

## ID（UUID + NanoID）標準
gRPC/Proto 上の ID は原則として `string` で表現し、内部向け/外部向けの意図をフィールド名で分離する。

### ルール（MUST）
- 内部主キー（UUID）: `*_id` は UUID を格納する `string` とする
  - 例: `user_id`, `order_id`
- 外部公開キー（NanoID）: `*_public_id` を NanoID を格納する `string` とする
  - 例: `user_public_id`
- 互換性のため、ID フィールドの型は将来も変更しない（`string` 固定）

### 使い分け（SHOULD）
- サービス間（内部）通信: UUID を基本にする（オーナーサービスの一意性と運用性を優先）
- 外部公開（HTTP/OpenAPI）: NanoID を基本にする（推測困難性と UX を優先）
- 外部入力で NanoID を受ける場合、Gateway/オーナーサービスで UUID へ解決し、内部は UUID に統一する

## Enum（Proto と PostgreSQL ENUM）標準
固定値は DB では PostgreSQL ENUM、API 契約では Proto enum を使い、両者の対応関係を安定させる。

- DB側で固定値を **VARCHAR（文字列）で管理する設計は禁止**。必ず PostgreSQL ENUM を使用する。

### Proto enum のルール（MUST）
- ゼロ値は `*_UNSPECIFIED`（または `_UNKNOWN`）とし、意味的な値にしない
- 既存の enum 値（番号）の再利用禁止。削除する場合は deprecated + reserved を検討する
- 追加は末尾に行い、既存番号を変更しない

### DB（PostgreSQL ENUM）との対応（SHOULD）
- Proto enum には「意味としての値」を置き、DB の ENUM ラベル（文字列）へはアプリ層で明示的にマッピングする
- DB ENUM の変更はマイグレーションを伴うため、API 契約（Proto enum）との同時リリース可否を事前に判断する

## gRPC Metadata / Observability（推奨）
運用上の相関とデバッグ性のため、メタデータとインターセプタで横断関心を統一する。

- `request_id` 相当のメタデータを伝播する（SHOULD）
- Interceptor で deadline 強制、ログ相関、トレース計装を行う（SHOULD）

## Proto の基本設計（MUST）
- `package` は **バージョン付き** にする（例: `foo.bar.v1`）
- `go_package` を必ず指定し、生成物の import が揺れないようにする
- 既存フィールド番号は再利用しない
  - 削除する場合は `reserved` を使う（番号・名前）
- 互換性を壊す変更（例）:
  - 既存フィールドの型変更
  - フィールド番号の変更
  - enum の既存値の削除/番号変更
  - 必須化（proto3 での semantic required 相当の変更）

## 命名・レイアウト（SHOULD）
- ディレクトリは `service_name/v1/*.proto` のように整理する
- 1ファイルに詰め込みすぎない（メッセージ/サービスを適度に分割）

## Buf（lint / breaking）によるガード（推奨）
`.proto` の品質と互換性は人手レビューだけだと破綻しやすいため、機械的に検査する。

### Lint
- `buf lint` を CI で必須にする
- ルールセットは `STANDARD` を基本にし、例外は `ignore_only` 等で明示する

### Breaking Change Detection
- `buf breaking` を CI で必須にする
- 比較対象は原則 `main`（または直近リリースタグ）

> 注意: Buf の breaking 検査は `google.api.http` 等の **カスタムオプション変更**（HTTP mapping）を互換性判定に含めない。
> そのため、HTTP 公開仕様（OpenAPI）については「生成物差分検出」を別途 CI で行う。

## 生成（protoc / buf generate）
- `.proto` の変更は必ずコード生成までを1セットとし、生成物を手編集しない
- generator/runtime のバージョン不一致を避ける（特に grpc-gateway）
- Go 1.24+ の `tool` directive を使い、生成ツールのバージョンを `go.mod` で固定する（推奨）

## Gateway: gRPC → HTTP/JSON(OpenAPI)
- HTTP 公開が必要な RPC は、grpc-gateway の HTTP mapping（`google.api.http`）で **明示的に** 経路を定義する（推奨）
- OpenAPI の出力は `docs/openapi.yaml` を正とし、フロントは Orval で型・Hooks を生成する

## gRPC ランタイム標準（MUST/SHOULD）

### Deadlines / Cancellation（MUST）
- すべての gRPC 呼び出しに deadline を設定する（無限待ち禁止）
- 上流のキャンセルを下流へ伝播する（context の伝播）

### Retry（SHOULD）
- リトライは「冪等性」が確認できる操作のみに限定する
- リトライは上流（呼び出し側）で統一し、重複実行に耐える設計（idempotency key 等）を前提にする

### Graceful Shutdown（SHOULD）
- 終了シグナル受信後は新規受付停止→処理中RPCの完了待ち→タイムアウトで強制終了、の順で停止する

### Health Checking（SHOULD）
- gRPC Health Checking Protocol（`grpc.health.v1.Health`）を実装する
- readiness と liveness の扱いを統一する（例: readiness は依存先接続/マイグレーション完了を条件に SERVING）

### Reflection（OPTIONAL）
- 開発環境でのみ reflection を有効化する（本番は原則無効）

## 参考（一次情報）
- gRPC Guides（deadlines/retry/health checking 等）: https://grpc.io/docs/guides/
- gRPC Health Checking Protocol: https://github.com/grpc/grpc/blob/master/doc/health-checking.md
- grpc-gateway: https://grpc-ecosystem.github.io/grpc-gateway/
- Buf lint/breaking: https://buf.build/docs/lint/ / https://buf.build/docs/breaking/
