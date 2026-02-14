# SQLC_QUERY_RULES

## 目的
sqlc 1.30.0 + pgx v5.8.0 を用いたデータアクセスの規約を定義し、SQL品質と型安全性を担保する。

## 禁止事項
- ORM（GORM/ent等）の導入
- handler からDBへ直アクセス
- 生成コードの手編集

## ファイル配置
- `sql/queries/*.sql`: アプリが実行するクエリ
- `sql/schema/*.sql` or `schema.hcl`: スキーマ定義（本リポジトリは Atlas の `schema.hcl` が正）

## クエリ設計
- 1クエリ=1ユースケースのI/Oに寄せる（汎用クエリ乱立を避ける）
- N+1 を発生させない（必要ならJOIN/バッチ）

## トランザクション境界
- usecase でTx境界を定義し、repository へTxを渡す
- 読み取り一貫性が必要な場合は、Tx内に閉じる

## NULL/型
- NULLは最小化する（NOT NULL + DEFAULT優先）
- NULLカラムは生成される nullable 型を統一して扱う（変換ヘルパを作る場合は `pkg/` へ）
