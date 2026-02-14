# API_CONTRACT_WORKFLOW

## 目的
Professor（Go）↔ Librarian（Python）の API を **OpenAPI を正（SSOT）**として管理し、契約逸脱を防ぐ。

## 前提（Librarian）
本サービスは以下の構成を前提とする:

```
eduanima-professor (Go)  ↔  eduanima-librarian (Python)
         [HTTP/JSON]            [Litestar]
         [OpenAPI: SSOT]
```

## 原則（MUST）
- 変更は必ず OpenAPI 定義から始める（Contract First）
- 生成物は手編集しない（再生成で消える）
- Librarian のハンドラーは OpenAPI と整合するように実装する

## フロー（推奨）
### 1) 契約定義（SSOT）
1. OpenAPI を更新する（破壊的変更かどうかを明記）
2. `ERROR_HANDLING.md` の共通形式に沿ってエラーも定義する

### 2) Librarian（Python）側
1. OpenAPI の request/response と一致する DTO（msgspec）を用意する
2. Litestar のルーティング/ハンドラーを実装する

### 3) Professor（Go）側
1. OpenAPI からクライアントを生成する（例: OpenAPI Generator / oapi-codegen 等）
2. Professor から Librarian を呼び出す呼び出し点をユースケースとして統一する

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
