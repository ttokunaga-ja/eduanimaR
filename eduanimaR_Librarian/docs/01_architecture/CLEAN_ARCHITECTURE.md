# CLEAN_ARCHITECTURE

## 目的
各サービスのディレクトリ構成と依存方向を固定し、機能追加・保守・テスト容易性を最大化する。

## 推奨レイアウト（Standard Go Layout + Clean Architecture）
- `cmd/<service>/`:
  - エントリポイント（main）。DI（依存注入）の組み立てのみ。
- `internal/handler/`:
  - HTTP層（Echo）。入出力変換、認可、バリデーション、エラーマッピング。
- `internal/usecase/`:
  - ビジネスロジック（ユースケース）。最重要。
  - ルール: handler/DB/Search の詳細に依存しない。
- `internal/domain/`:
  - エンティティ/値オブジェクト/ドメインエラー。
- `internal/repository/`:
  - 永続化・外部I/F（PostgreSQL, Elasticsearch等）の実装。
- `pkg/`:
  - サービス横断で共有してよい（かつ安定）なライブラリのみ。

## 依存方向（必須）
- `handler` → `usecase` → `domain`
- `repository` → `domain`
- `usecase` は `repository` の「インターフェース」に依存し、実装には依存しない。

## 境界の作り方
- `usecase` 側で ports を定義する（例: `UserRepository` interface）
- `repository` で adapters を実装する

## 禁止事項
- handler から直接DBクエリを実行しない
- domain が pgx/sqlc/Echo に依存しない
- 生成コード（sqlc / OpenAPI）を手で編集しない
