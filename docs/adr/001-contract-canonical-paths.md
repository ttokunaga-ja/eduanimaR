# ADR 001: サービス間契約の正本（SSOT）を contracts/ に集約する

- **日付**: 2026-03-06
- **ステータス**: 採択済み

---

## コンテキスト

eduanimaR はモノレポに 3 つのマイクロサービスを含む。

| サービス | 言語 | 使用する契約 |
|---|---|---|
| `apps/professor` | Go | OpenAPI（REST）/ Proto（gRPC クライアント） |
| `apps/librarian` | Python | Proto（gRPC サーバー） |
| `apps/web` | TypeScript | OpenAPI（REST クライアント生成） |

リファクタリング前は契約ファイルが各サービスディレクトリに散在し、次の技術的負債が発生していた:

- `eduanimaR_Professor/docs/openapi.yaml`（SSOT）と `eduanimaR/openapi/openapi.yaml`（コピー）が手動同期
- `eduanimaR_Professor/proto/` と `eduanimaR_Librarian/proto/` に同一 `.proto` が重複
- SSOT がどのファイルかを README コメントでのみ示しており、機械的な保証がなかった

---

## 決定

**すべてのサービス間インタフェース定義を `contracts/` ディレクトリに集約する。**

```
contracts/
├── openapi/
│   └── professor.yaml          # REST API 契約（SSOT）
└── proto/
    ├── buf.yaml                 # buf モジュール定義（lint / breaking）
    └── librarian/
        └── v1/
            └── librarian.proto  # gRPC 契約（SSOT）
```

各サービスは **contracts/ を参照する（コピーしない）**:

| サービス | 参照方法 |
|---|---|
| `apps/web` | `orval.config.ts` → `../../contracts/openapi/professor.yaml` |
| `apps/professor` | `buf.gen.yaml` → `inputs: directory: ../../contracts/proto` |
| `apps/librarian` | `Makefile` → `grpc_tools.protoc -I ../../contracts/proto ...` |

---

## 理由

1. **単一の真実（SSOT）を保証する**: ファイルが 1 つなのでコピーの乖離（contract drift）が起きない。
2. **機械的な検証が可能**: CI（`contract-drift.yml`）が PR ごとに乖離を検出する。
3. **変更の影響範囲が明確**: `contracts/` への PR = すべてのサービスに影響する変更、と判断できる。

---

## 影響・トレードオフ

| 観点 | 内容 |
|---|---|
| デメリット | `apps/professor/buf.gen.yaml` の `directory` パスがモノレポ構造に依存する |
| 対策 | Makefile の `proto` / `generate` ターゲットが相対パスを隠蔽している |
| 将来対応 | サービスが別リポジトリに分離する場合は Buf Schema Registry（BSR）への公開を検討する |

---

## 関連

- `contracts/README.md` — 生成コマンドと運用手順
- `.github/workflows/contract-drift.yml` — CI ゲート
