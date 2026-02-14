# Project Decisions（SSOT）

このファイルは「プロジェクトごとに選択が必要」な決定事項の SSOT。
AI/人間が推測で穴埋めしないために、まずここを埋めてから実装する。

## 基本
- プロジェクト名：
- リポジトリ：
- 環境：local / staging / production

## サービス境界（Must）
- サービス一覧（責務/所有データ/公開IF）：
- Gateway の責務（何をする/しない）：
- サービス間依存（同期/非同期、禁止依存）：

## 契約（Must）
- 外向き（HTTP/JSON）：OpenAPI の SSOT 置き場：
- 内向き（gRPC/Proto）：`.proto` の SSOT 置き場：
- 互換性ポリシー：追加/変更/削除の扱い（`API_VERSIONING_DEPRECATION.md` と整合）：

## データ（Must）
- DB：PostgreSQL（採用/バージョン）：
- スキーマ SSOT：Atlas の置き場：
- SQL SSOT：sqlc クエリ置き場：
- ID 方針（UUIDv7/NanoID 等）：

## 運用（Must）
- 観測性（ログ/メトリクス/トレース）導入範囲：
- SLO（対象導線/指標/アラート閾値）：
- Secrets 管理（どこで管理し、どう配るか）：

## 配信（Must）
- デプロイ先（Cloud Run 等）：
- ロールバック方針（戻せる条件/DB変更の取り扱い）：
