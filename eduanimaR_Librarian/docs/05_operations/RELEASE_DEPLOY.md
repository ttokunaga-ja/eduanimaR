# RELEASE_DEPLOY

## 目的
各サービスを独立して安全にリリースするためのビルド/デプロイ方針を定義する。

## 原則
- 1サービス = 1成果物（コンテナイメージ）
- 設定は環境変数/シークレットで注入し、イメージに埋め込まない
- DBマイグレーションは Atlas フロー（`MIGRATION_FLOW.md`）に従う

## デプロイ手順（例）
1. CIでテスト/静的解析を実施
2. Dockerイメージをビルドしてレジストリへpush
3. ステージングでデプロイ
4. 本番でローリング（またはblue/green）

> 段階的リリース（カナリア/自動ロールバック/フラグ）は `05_operations/PROGRESSIVE_DELIVERY.md` を参照。

## ロールバック
- アプリのロールバックとDBスキーマの互換性を事前に確認する
- 破壊的スキーマ変更は段階的に行う（expand/contract）

## 関連
- `05_operations/PROGRESSIVE_DELIVERY.md`
- `05_operations/MIGRATION_FLOW.md`
- `05_operations/SUPPLY_CHAIN_SECURITY.md`
- `05_operations/SECRETS_KEY_MANAGEMENT.md`
- `05_operations/SLO_ALERTING.md`

