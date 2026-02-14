# PROTOBUF_GRPC_STANDARDS

## 目的
内部契約（.proto / gRPC）を **安全に進化** させ、Professor ↔ Librarian の結合度を制御するための標準を定義する。

本ドキュメントは以下を対象とする:
- Professor ↔ Librarian の gRPC API
- 互換性・生成・CI でのガード

## SSOT
- 内部API契約の正: `.proto`
- 配置: `proto/` 配下（例: `proto/librarian/v1/librarian.proto`）

## 前提（eduanimaR）
- Professor は DB/GCS への唯一の直接アクセス権を持つ（データの守護者）
- Librarian はステートレスな推論サービスで、DB/GCS へ直接アクセスしない
- gRPC は Professor ↔ Librarian の内部通信にのみ使用する

## ID（MUST）
gRPC/Proto 上の ID は原則として `string`（UUID文字列）で表現し、互換性のため型変更をしない。

### ルール（MUST）
- 既存の ID フィールドの型は将来も変更しない（`string` 固定）

## Enum（MUST）
- ゼロ値は `*_UNSPECIFIED`（または `_UNKNOWN`）とし、意味的な値にしない
- 既存の enum 値（番号）の再利用禁止。削除する場合は deprecated + reserved を検討する
- 追加は末尾に行い、既存番号を変更しない

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
- generator/runtime のバージョン不一致を避ける

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
- Buf lint/breaking: https://buf.build/docs/lint/ / https://buf.build/docs/breaking/
