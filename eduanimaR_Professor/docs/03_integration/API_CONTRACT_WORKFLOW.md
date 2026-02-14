# API_CONTRACT_WORKFLOW

## 目的
Frontend ↔ Professor（Go）の外向きAPIを **OpenAPI（SSOT）** で契約駆動にし、契約逸脱を防ぐ。

補足: Professor ↔ Librarian（推論サービス）の内部通信は gRPC/Proto を SSOT とし、本ドキュメントでは外向き OpenAPI を主対象とする。

## 前提（確定）
本プロジェクトの通信は以下を採用する:

```
Frontend ↔ Professor
[HTTP/JSON + SSE]
[OpenAPI SSOT]
```

## SSOT
- 外向き（OpenAPI）: `docs/openapi.yaml`
- エラー形式/コード: `ERROR_HANDLING.md` / `ERROR_CODES.md`

## 原則（MUST）
- 変更は必ず OpenAPI 定義から始める（Contract First）
- 生成物は手編集しない（再生成で消える）
- Consumer 側でクライアント生成を行う場合、生成物を正とし手書き型定義を避ける
- SSE を含む外向き契約は「破壊的変更を避ける」を基本とする（必要なら versioning/deprecation を行う）

## フロー（推奨）
### Professor（Go）側
1. `docs/openapi.yaml` を更新する
2. Professor の HTTP handler / SSE を実装し、契約と整合させる
3. エラー形式/コードを `ERROR_HANDLING.md` / `ERROR_CODES.md` と整合させる
4. 破壊的変更が必要な場合は `API_VERSIONING_DEPRECATION.md` の手順に従う

### Consumer（例: フロントエンド）側
1. `docs/openapi.yaml` を取得する
2. 必要に応じてクライアント/型を生成し、生成物を正として利用する

## レビュー観点
- 互換性: 既存クライアントに影響する変更か（必須/任意、型変更、enum削除等）
- セキュリティ: 認証/認可が明示されているか（scope/role等）
- 例外系: 4xx/5xx のレスポンスが定義されているか
- SSE: イベント名/形が安定しているか（再接続や順序の前提を壊していないか）
- **命名規則**: consumer が理解しやすい名前か（例: `user_id` vs `userId`）

## APIライフサイクル（推奨）
- 廃止（deprecation）の方針（期間、告知、削除手順）を決める
- 破壊的変更は原則避け、必要なら `/v1` 等でバージョン分離する
- 公開APIの棚卸し（インベントリ）を維持する（OWASP: Improper Inventory Management 対応）

> セキュリティ観点の詳細は `05_operations/API_SECURITY.md` を参照。
