# ADR 004: librarian Python proto shim 層を過渡期まで維持する

- **日付**: 2026-03-07
- **ステータス**: 採択済み（Phase 3 で解消予定）

---

## コンテキスト

librarian サービスの proto stubs は `make proto` によって `gen/proto/python/` に生成される。
一方で、`src/librarian/v1/` に 3 ファイルの "shim" モジュールが存在し、
`gen/proto/python/` から re-export している。

```
apps/librarian/
├── src/librarian/v1/
│   ├── __init__.py          # "Runtime shim package"（3行）
│   ├── librarian_pb2.py     # from gen.proto.python.librarian.v1.librarian_pb2 import *
│   └── librarian_pb2_grpc.py
└── gen/proto/python/librarian/v1/  ← make proto による実体
```

`main.py` / `server.py` は `from librarian.v1 import librarian_pb2_grpc` とシム経由でインポートする。

---

## 問題点

- `PYTHONPATH=src` 設定では `gen/proto/python/` が直接 import できず、shim が必要だった
- `gen/proto/python/` は `make proto` 実行前は存在しないため、`PYTHONPATH` に追加するだけでは
  Docker ビルド順序に依存した問題が生じる可能性がある
- shim 自体は各 3 行であり、実害は小さい

---

## 決定

**現在の shim 構造（`src/librarian/v1/`）を Phase 3 まで維持する。**

削除のための前提条件が整い次第（以下）、Phase 3 で解消する:

1. `PYTHONPATH=src:gen/proto/python` に変更し、`gen/` 直インポートが可能か検証
2. Dockerfile の `PYTHONPATH` を相応に更新
3. `src/librarian/v1/` ディレクトリを削除し、import を直接 `gen.proto.python.librarian.v1` に変更
4. `make typecheck` が通ることを確認

---

## 変更禁止事項

- shim ファイルを手動で編集しない（`make proto` で上書きされる想定）
- shim を増やさない（contractで定義された proto service が増えた場合も同様）

---

## 関連

- `apps/librarian/Makefile` — `proto` ターゲット
- `contracts/proto/librarian/v1/librarian.proto` — 契約の SSOT
- ADR 001: `contracts/` による SSOT
