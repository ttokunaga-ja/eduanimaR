# DB_SCHEMA_DESIGN

## 目的
PostgreSQL 18.1 + Atlas + sqlc 前提で、スキーマ設計の意思決定（型/制約/インデックス）を統一する。

## データ所有（最重要）
- Cloud SQL（PostgreSQL）と GCS の直接アクセス権限は **Professor のみに付与**する
- Librarian は DB/GCS の資格情報を持たない（設計・運用の不変条件）

## ID戦略（UUID + NanoID）
- 内部主キー: UUID（推奨: UUIDv7 / `uuidv7()` を利用）
- 外部公開ID: NanoID（URL/ログ/問い合わせで扱いやすい短ID）
- ルール:
  - 参照整合性は内部UUIDで維持する
  - 外部公開IDはユニーク制約 + 露出するAPIのみで使用する

## ENUMの積極採用
- 固定値（status/type/category）は **PostgreSQL ENUM を必須** とする
  - **VARCHAR で固定値を管理する設計は禁止**（typo/バリデーション漏れ/性能劣化を誘発する）
- 利点: 型安全性、制約の明確化、アプリ側の分岐漏れ検知
- 変更方針: 追加は許容、削除/名前変更は慎重に（互換性を壊しやすい）

## NULLとデフォルト
- 原則: `NOT NULL` + `DEFAULT` を優先
- NULLが必要な場合:
  - sqlc/pgx が生成する nullable 型を統一して使う
  - APIのJSON表現（省略/明示null）も合わせて決める

## インデックス
- B-tree を基本とし、検索要件に応じて GIN / GiST / HNSW(pgvector) を選定
- 18.1の機能（例: B-tree Skip Scan 等）は「要件を満たす場合のみ」採用し、必ずベンチマークを残す

## ベクトル検索（pgvector 0.8.1）
- OLTPとベクトル検索を同居させる場合は、テーブル分離/更新頻度/インデックス再構築コストを考慮
- HNSW を使う場合:
  - 取り込みバッチ/再構築戦略（オフピーク）を定める
  - 近似検索の許容誤差（recall）を要件化する

## Atlas運用前提
- スキーマ変更は `schema.hcl` が唯一の正
- 手動 `ALTER TABLE` は禁止（差分が壊れる）

## マルチテナント/物理制約（MUST）
- 検索・参照の主経路は「user_id / subject_id による物理絞り込み」を前提にする
- 主要テーブルは原則として以下のカラムを持つこと
  - `user_id`
  - `subject_id`
  - `is_active` または `deleted_at`

## LLM派生データの世代管理（推奨）
- OCR/構造化/Embedding は将来のモデル更新で再生成される
- 「原本（GCS）」と「派生（DB）」を分け、派生は version/generation を持てる設計にする
