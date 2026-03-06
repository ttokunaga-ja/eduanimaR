# RELEASE_DEPLOY

## 目的
Professor（このリポジトリのGoサービス）を安全にリリースするためのビルド/デプロイ方針を定義する。

> Librarian は別サービスとして独立デプロイされる前提だが、Professor 側は gRPC 契約互換（proto）とタイムアウト/リトライ方針を維持する。

## 原則
- 1サービス = 1成果物（コンテナイメージ）
- 設定は環境変数/シークレットで注入し、イメージに埋め込まない
- DBマイグレーションは Atlas フロー（`MIGRATION_FLOW.md`）に従う
- 外向き契約（OpenAPI）と内向き契約（Proto）はSSOTとしてCIで破壊的変更を検知する
- SSE を提供するため、デプロイ切替時の接続切断/再接続を前提にする

## デプロイ手順（例）
1. CIでテスト/静的解析を実施
2. Dockerイメージをビルドしてレジストリへpush
3. （必要なら）DBマイグレーションをステージングで適用
4. ステージングへデプロイ
5. 本番でローリング（またはblue/green）

### Cloud Run を想定した補足
- ヘルスチェック（readiness）に失敗している間はトラフィックを受けない
- シャットダウン時は、SSE/HTTP の処理中リクエストを可能な範囲で graceful に終了する
- 過負荷時はスロットリング/同時接続制御（SSE）を優先し、SLO を守る

> 段階的リリース（カナリア/自動ロールバック/フラグ）は `05_operations/PROGRESSIVE_DELIVERY.md` を参照。

## ロールバック
- アプリのロールバックとDBスキーマの互換性を事前に確認する
- 破壊的スキーマ変更は段階的に行う（expand/contract）
- SSE はロールバック時も「クライアント再接続」を前提にする（切断は仕様に含める）

## 関連
- `05_operations/PROGRESSIVE_DELIVERY.md`
- `05_operations/MIGRATION_FLOW.md`
- `05_operations/SUPPLY_CHAIN_SECURITY.md`
- `05_operations/SECRETS_KEY_MANAGEMENT.md`
- `05_operations/SLO_ALERTING.md`

