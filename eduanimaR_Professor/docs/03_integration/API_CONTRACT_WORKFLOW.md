# API_CONTRACT_WORKFLOW

## 目的
OpenAPI を正（SSOT）とし、そこからサーバー実装（Echo）とクライアント実装（Next.js/TypeScript）の両方に必要な型/インターフェースを生成することで、契約逸脱を防ぐ。

## 前提:2段階ゲートウェイ構成 + gRPC内部通信
本プロジェクトは以下の構成を採用する:

```
Browser → Next.js (BFF) → Go API Gateway → Go Microservices (User/Product/Order...)
         [HTTP/JSON]        [HTTP/JSON]     [gRPC]
         [OpenAPI]                          [Protocol Buffers]
```

- **Next.js(BFF)**: UI向けゲートウェイ。データ整形・集約・Cookie/Session管理。
- **Go API Gateway**: システム全体のゲートウェイ。認証/認可/レート制限/ルーティング。
  - **gRPC → OpenAPI 変換**を担当(grpc-gateway等を使用)
  - 外向きはHTTP/JSON(OpenAPI)、内向きはgRPC(Protocol Buffers)
- **Go Microservices**: ビジネスロジック実装。DB/ES等への永続化。
  - **gRPCサービス**として実装(.protoが契約)

## 原則（MUST）
- **ハンドラーのシグネチャを手動で変更しない**
- 変更は必ず OpenAPI 定義から始める（Contract First）
- 生成物は手編集しない（再生成で消える）
- **フロントエンドは Orval で生成された型・Hooks のみを使用する**（手書き型定義禁止）

## フロー（推奨）
### バックエンド側
#### A) 内部サービス (gRPC) の開発
1. `.proto` ファイルを定義・更新する
   - service / rpc / message を定義
   - 必要に応じて grpc-gateway の annotations を付与(HTTP mapping用)
2. `protoc` でGoコードを生成する
3. `internal/service` で生成されたgRPCサーバーインターフェースを実装する

#### B) Gateway (gRPC → OpenAPI) の構成
1. Gateway が内部gRPCサービスを呼び出せるように設定
2. grpc-gateway または connectrpc を使って、gRPCをHTTP/JSONに変換
3. 変換結果として `docs/openapi.yaml` を生成・公開する
4. `ERROR_HANDLING.md` の共通形式に沿ってHTTPエラーをマッピングする
5. 破壊的変更の場合は、バージョニング（`/v1` 等）か互換運用の方針を決める

### フロントエンド側（Orval による自動生成）
1. バックエンドが出力した `openapi.yaml` (または JSON) を取得
2. `npm run api:generate` (Orval) を実行
3. `src/shared/api/` 配下に TypeScript の型・React Hooks が生成される
4. FSD の `entities` / `features` 層で生成された Hooks（`useGetUser` 等）を使用
5. **手書きで `fetch` や `axios` を書かない**（生成に統一）

## レビュー観点
- 互換性: 既存クライアントに影響する変更か（必須/任意、型変更、enum削除等）
- セキュリティ: 認証/認可が明示されているか（scope/role等）
- 例外系: 4xx/5xx のレスポンスが定義されているか
- **命名規則**: フロントエンドが理解しやすい名前か（`user_id` vs `userId` 等、camelCase統一を推奨）

## APIライフサイクル（推奨）
- 廃止（deprecation）の方針（期間、告知、削除手順）を決める
- 破壊的変更は原則避け、必要なら `/v1` 等でバージョン分離する
- 公開APIの棚卸し（インベントリ）を維持する（OWASP: Improper Inventory Management 対応）

> セキュリティ観点の詳細は `05_operations/API_SECURITY.md` を参照。
