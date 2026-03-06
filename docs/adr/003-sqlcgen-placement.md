# ADR 003: sqlcgen の出力先を internal/ 内に配置する

- **日付**: 2026-03-07
- **ステータス**: 採択済み

---

## コンテキスト

eduanimaR では生成コードを `gen/` ディレクトリに集約する方針を採用している（ADR 001）。
Proto codegen の出力は `apps/professor/gen/proto/` に配置しており、
一方で sqlc（SQL→Go コード生成）の出力は `apps/professor/internal/adapters/postgres/sqlcgen/` に配置している。

これは以下の理由による意図的な例外である。

---

## 決定

**sqlcgen の出力先を `internal/adapters/postgres/sqlcgen/` に維持する。**

```
apps/professor/
├── gen/
│   └── proto/          ← buf generate の出力（契約生成）
└── internal/
    └── adapters/
        └── postgres/
            └── sqlcgen/ ← sqlc generate の出力（DB アダプター生成）
```

---

## 理由

1. **スコープの局所性**: sqlcgen は `postgres` アダプターパッケージ専用の依存物であり、
   他のパッケージからは参照されない。`gen/` に出すと「どこからでも参照できる」印象を与え、
   アーキテクチャの意図が不明確になる。

2. **sqlc.yaml の慣習**: sqlc 公式サンプルは多くの場合 `internal/` 内に生成先を設ける。
   `gen/` への配置は Proto 生成コードの慣習（buf, protoc）であり、SQL 生成コードとは文脈が異なる。

3. **変更コストと便益のバランス**: 生成先を `gen/sql/` に変更すると
   `internal/adapters/postgres/*.go` のインポートパスを全更新する必要がある。
   現時点での便益はほぼなく、リファクタリングリスクのみが生じる。

---

## ルール

| 生成ツール | 出力先 | 理由 |
|---|---|---|
| `buf generate` (Proto) | `gen/proto/` | 契約生成・複数パッケージから参照される可能性 |
| `sqlc generate` (SQL) | `internal/adapters/postgres/sqlcgen/` | DB アダプター専有・スコープが限定的 |

---

## 将来対応

サービスが複数の DB（マルチテナント等）を持つようになった場合は改めて検討する。

---

## 関連

- `apps/professor/sqlc.yaml` — sqlc 設定ファイル
- ADR 001: `contracts/` によるサービス間契約の SSOT
