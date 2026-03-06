# RELEASE_DEPLOY

## 目的
各サービスを独立して安全にリリースするためのビルド/デプロイ方針を定義する。

## 原則
- 1サービス = 1成果物（コンテナイメージ）
- 設定は環境変数/シークレットで注入し、イメージに埋め込まない

## デプロイ手順（例）
1. CIでテスト/静的解析を実施
2. Dockerイメージをビルドしてレジストリへpush
3. ステージングでデプロイ
4. 本番でローリング（またはblue/green）

> 段階的リリース（カナリア/自動ロールバック/フラグ）は `05_operations/PROGRESSIVE_DELIVERY.md` を参照。

## ロールバック
- アプリのロールバック手順を事前に確認する
- 契約（OpenAPI）変更は破壊的変更を避け、必要ならバージョニング/非推奨化を行う

## 関連
- `05_operations/PROGRESSIVE_DELIVERY.md`
- `05_operations/SUPPLY_CHAIN_SECURITY.md`
- `05_operations/SECRETS_KEY_MANAGEMENT.md`
- `05_operations/SLO_ALERTING.md`

