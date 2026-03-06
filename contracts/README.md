# contracts/ — サービス間契約の SSOT

すべてのサービス間インタフェース定義（OpenAPI / Protobuf）の **Single Source of Truth** です。

## ディレクトリ構成

```
contracts/
├── openapi/
│   └── professor.yaml    # Professor REST API 契約（SSOT）
└── proto/
    └── librarian/
        └── v1/
            └── librarian.proto   # Professor ↔ Librarian gRPC 契約（SSOT）
```

## 原則

| ルール | 理由 |
|---|---|
| このディレクトリのファイルだけが「正」 | 各サービスのディレクトリ内に契約ファイルのコピーを持たない |
| 契約変更は PR レビューを必須とする | breaking change の影響が両サービスに及ぶため |
| 変更後は各サービスの生成コマンドを再実行する | Professor: `make generate` / Librarian: `make proto` / Web: `npm run api:generate` |

## 生成コマンド（変更後に実行）

```bash
# Professor（Go コード生成）
cd apps/professor && make generate

# Librarian（Python stub 生成）
cd apps/librarian && make proto

# Web（TypeScript クライアント生成）
cd apps/web && npm run api:generate
```

## contract drift 検知（CI）

PR 時に `.github/workflows/contract-drift.yml` が自動チェックを実行します。
- OpenAPI → TypeScript 生成コードの差分
- Proto breaking change
