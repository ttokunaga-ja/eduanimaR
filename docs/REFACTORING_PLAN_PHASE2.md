# リファクタリング計画 Phase 2 — 技術的負債最小化ロードマップ

> 作成日: 2026-03-07  
> 対象: eduanimaR モノレポ（Python / Go / TypeScript マイクロサービス）  
> 前提: リリース前開発中。Phase 1 ディレクトリ統一 (ADR 001, 002) は**完了済み**。

---

## 1. 現状アーキテクチャの概要

```
eduanimaR/
├── apps/
│   ├── professor/    Go — Echo REST API + PostgreSQL/pgvector + Kafka producer + gRPC client
│   ├── librarian/    Python — gRPC server + LangGraph RAG エージェント + Gemini AI
│   └── web/          TypeScript — Next.js 15 App Router + FSD + orval コード生成
├── contracts/        SSOT: OpenAPI (professor.yaml) + Proto (librarian.proto)
├── ops/              Docker Compose / Cloud Run / CI ドキュメント
├── handbook/         プロダクト・ビジネスドキュメント
└── docs/             ADR / 技術ドキュメント
```

**Phase 1 で完了した主要施策:**

| ✅ | 内容 |
|---|---|
| ADR 001 | `contracts/` に OpenAPI・Proto を集約（SSOT） |
| ADR 002 | `apps/<name>/` にサービスを統一（旧: `eduanimaR_Professor` 等） |
| gen/ 分離 | 各サービスの生成コードを `gen/` ディレクトリに配置 |
| shim 層 | librarian の `src/librarian/v1/*.py` が `gen/` を re-export |
| テスト骨格 | unit/integration/contract のディレクトリ構造を全サービスに導入 |
| CI 統合 | contract-drift / repo-verify / quality / i18n-check ワークフロー |
| Phase2 標準チェック | health/readyz エンドポイント + 可観測性キーを CI で強制 |

---

## 2. 発見した技術的負債の全体マップ

### 重要度分類

| 優先度 | 基準 |
|---|---|
| 🔴 **Critical** | リリースブロッカー / セキュリティリスク |
| 🟠 **High** | 保守コスト増大 / 将来の手戻りリスク |
| 🟡 **Medium** | 一貫性・運用品質の問題 |
| 🟢 **Low** | 可読性・クリーンアップ |

---

## 3. 個別問題の詳細

### 🔴 [CRIT-1] 認証スタブが本番コードパスに混入

**場所:** `apps/professor/internal/adapters/http/middleware/devuser.go`

```go
// DevUser は Phase 1 用の固定ユーザーミドルウェア。
// Phase 2 では JWT 検証ミドルウェアに差し替える。
func DevUser() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            c.Set(ctxKeyUserID, domain.DevUserID)  // ← 常に固定UUIDを設定
            return next(c)
        }
    }
}
```

**リスク:** `APP_ENV` などによるガードがなく、本番デプロイ時に全リクエストが
`DevUserID (00000000-0000-0000-0000-000000000001)` として処理される。
OWASP A01: Broken Access Control に該当する。

**対処計画:**

```
Phase 2a — JWT ミドルウェアを実装し DevUser を完全に置き換える
Phase 2b — domain.DevUserID / DevUserEmail をドメイン層から削除
Phase 2c — ssot_test.go に「プロダクションビルドに DevUserEmail が含まれないこと」を追加
```

---

### 🔴 [CRIT-2] Go モジュール名が旧パスを参照

**場所:** `apps/professor/go.mod`

```go
module github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor  // ← 旧命名
```

全 import パスが `eduanimaR_Professor` を参照しており、ADR 002 で決定した
`apps/professor` ディレクトリ名と不一致。

**影響ファイル:**

```
apps/professor/cmd/professor/main.go        (import 23行)
apps/professor/internal/usecases/*.go       (import ~20行)
apps/professor/internal/adapters/**/*.go    (import ~30行)
apps/professor/internal/ports/*.go          (import ~10行)
apps/professor/Makefile                     (MODULE 変数)
```

**修正コマンド（アトミック）:**

```bash
# go.mod の module パスを変更
go mod edit -module github.com/ttokunaga-ja/eduanimaR/apps/professor

# 既存の import を一括置換
find apps/professor -name '*.go' -exec sed -i \
  's|github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor|github.com/ttokunaga-ja/eduanimaR/apps/professor|g' {} +

# apps/professor/Makefile の MODULE 変数も更新
```

> **ポイント:** ディレクトリは ADR 002 で既に `apps/professor` に移動済みなので、
> モジュール名を合わせるだけ。**go.sum の再生成は不要**（モジュールの依存関係は変わらない）。

---

### 🟠 [HIGH-1] sqlcgen が `gen/` ではなく `internal/` 内に生成される

**場所:** `apps/professor/sqlc.yaml → out: internal/adapters/postgres/sqlcgen/`

`gen/proto/` は明示的に `gen/` に配置しているが、SQLコード生成は `internal/` 内に混在。
`gen/` ディレクトリが「生成コード置き場」という一貫した規則を崩している。

**選択肢:**

| 案 | `sqlc out:` | `import` パス変更 |
|---|---|---|
| A (推奨) | `gen/sql/` | あり (2〜3ファイル) |
| B (現状維持) | `internal/adapters/postgres/sqlcgen/` | なし |

**推奨:** 案 A。`gen/` になんでも生成コードを集約する一貫性の方が、
初見の開発者が生成コードと手書きコードを区別しやすい。
ただし sqlcgen は postgres アダプター専用のため、**案 B の現状維持も許容範囲**。
ADR として明示しておけば負債ではなくなる。

**結論:** `docs/adr/003-sqlcgen-placement.md` を追加し、意図的な決定として記録。

---

### 🟠 [HIGH-2] librarian の proto shim 層を解消する

**現状:**

```
apps/librarian/
├── src/librarian/v1/
│   ├── __init__.py          # "Runtime shim package"
│   ├── librarian_pb2.py     # gen/proto/python/ から re-export
│   └── librarian_pb2_grpc.py
└── gen/proto/python/librarian/v1/  ← 実体
```

`main.py` が `from librarian.v1 import librarian_pb2_grpc` とシムを経由する。

**shim 解消手順:**

1. `server.py` / `main.py` の import を直接 `gen.proto.python.librarian.v1` に変更
2. `PYTHONPATH=src:gen/proto/python` などに変更するか、
   または `src/` の PYTHONPATH は維持したまま `gen/proto/python` を追加
3. `src/librarian/v1/` ディレクトリを削除
4. `make proto` の生成先を確認して Makefile の `typecheck` ターゲットを更新

**代替案 (より簡単):** shim を残し、ADR で「移行完了まで維持」と明記する。
shim 自体 3 行のファイルなので実害は小さい。优先度は Medium と判断してもよい。

---

### 🟠 [HIGH-3] 起動時に API キー存在確認がない

**場所:** `apps/professor/internal/config/config.go` / `apps/librarian/src/librarian/config.py`

```go
GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),  // 空文字がデフォルト
```

`GEMINI_API_KEY=""` のまま起動した場合、LLM 呼び出し時まで検出されない。

**修正:** 起動時バリデーション

```go
// config.go
func Validate(cfg *Config) error {
    if cfg.GeminiAPIKey == "" {
        return fmt.Errorf("GEMINI_API_KEY is required")
    }
    return nil
}
```

```python
# librarian/main.py の serve() 内
if not cfg.gemini_api_key:
    logger.error("GEMINI_API_KEY is required")
    sys.exit(1)
```

---

### 🟡 [MED-1] structlog がインストール済みだが未使用 (librarian)

**場所:** `apps/librarian/pyproject.toml` + `src/librarian/*.py`

```toml
dependencies = ["structlog>=24.1.0"]  # インストール済み
```

```python
# 実際のコード
import logging  # stdlib のみ使用、structlog は未使用
logger = logging.getLogger(__name__)
```

**修正:** すべてのモジュールで `structlog` に統一。JSON ログが Cloud Run / GCP Logging
と統合しやすく、`request_id` / `trace_id` / `user_id` キーが構造化ログに含まれる。

```python
import structlog
logger = structlog.get_logger(__name__)

# main.py の setup_logging() を structlog.configure() に置き換え
```

---

### 🟡 [MED-2] OpenTelemetry が依存に追加されているが未実装

**場所:** `apps/professor/go.mod`

```
go.opentelemetry.io/otel v1.32.0                     # 宣言済み
go.opentelemetry.io/otel/trace v1.32.0               # 宣言済み
go.opentelemetry.io/contrib/instrumentation/...      # 宣言済み
```

**現状の可観測性:** 構造化ログ (`slog`) にトレースIDを埋め込む方式のみ。
分散トレーシング（OTLP エクスポーター）は未配線。

**実装計画 (Phase 2):**

```go
// cmd/professor/main.go に追加
tp := setupTracerProvider(cfg)  // OTLP or Cloud Trace exporter
otel.SetTracerProvider(tp)

// Echo ミドルウェアに otelecho を追加
e.Use(otelecho.Middleware("professor"))
```

GCP Cloud Run 環境では `OTEL_EXPORTER_OTLP_ENDPOINT` または
Cloud Trace の gRPC エンドポーターを使用する。

---

### 🟡 [MED-3] レート制限ミドルウェアが未設置

**場所:** `apps/professor/cmd/professor/main.go`

Echo の標準ミドルウェア `echomw.RateLimiter` が未設定。
API エンドポイント（特に `/api/v1/subjects/:id/chat` の SSE ストリーム）への
リクエスト폭走に対する保護がない。

```go
e.Use(echomw.RateLimiterWithConfig(echomw.RateLimiterConfig{
    Store: echomw.NewRateLimiterMemoryStore(20),  // 20 req/s per IP
}))
```

---

### 🟡 [MED-4] テスト不足 (librarian Python)

**現状:**

```
apps/librarian/tests/
├── unit/
│   └── test_graph.py   # 1ファイルのみ
├── integration/
│   └── README.md       # 空（スタブのみ）
└── contract/
    └── README.md       # 空（スタブのみ）
```

Go サービスには `usecases/chat_usecase_test.go` や `tests/contract/ssot_test.go` があるが、
Python サービスのテストカバレッジが大幅に不足。

**優先追加テスト:**

1. `tests/unit/test_config.py` — Config 読み込み・バリデーション
2. `tests/unit/test_server.py` — LibrarianServicer の gRPC メッセージ処理
3. `tests/contract/test_proto_ssot.py` — contracts/proto の存在と互換性検証

---

### 🟡 [MED-5] `apps/web/package.json` の name がテンプレートのまま

```json
{ "name": "codingagent-fsd-template" }  // ← テンプレートアーティファクト
```

→ `"name": "eduanima-web"` に変更。

---

### 🟢 [LOW-1] ドキュメント3重構造と内容重複

**現状:**

```
apps/professor/docs/  (05_operations/OBSERVABILITY.md 等 50+ ファイル)
apps/librarian/docs/  (同一構造 50+ ファイル)
apps/web/docs/        (同一構造 50+ ファイル)
ops/docs/             (運用ドキュメント)
docs/                 (ADR + トップレベルドキュメント)
```

横断的な内容（CI/CD, 可観測性, セキュリティ, デプロイ）が 3 サービスで重複。

**長期方針:** 横断的ドキュメントは `docs/` または `ops/docs/` に統合し、
サービス固有の docs/ は「サービス固有の差分」のみに絞る。ただし:

- 現時点でビルド・テストには影響しない
- `apps/*/docs/` は Cursor Rules 等のコンテキストとして機能している
- スコープ外でリリースをブロックしない

**Phase 3 以降で段階的に対応。** まず重複箇所にコメントを追加し意図を明記。

---

### 🟢 [LOW-2] `apps/professor/docs/openapi.librarian.yaml` の位置づけが不明確

このファイルは Librarian API の**参考定義**（HTTP/JSON フォールバック用）であり、
SSOT は `contracts/proto/librarian/v1/librarian.proto`。
ファイル内コメントには記載があるが、`docs/` 内 YAML が contracts/ と混同されるリスクがある。

**対処:** ファイル名を `openapi.librarian.reference.yaml` にリネームするか、
または先頭のコメントをより目立つ形式（YAML アンカー等）で強調する。

---

### 🟢 [LOW-3] `tmp/` の追跡ファイルと root レベルの孤立ファイル

```
tmp/_dsstore_tracked.txt       # .gitignore 管理用メモというが root-level に不要
tmp/_tracked_unwanted_list.txt
researchFindings.md             # リサーチメモ — handbook/ に移動か削除
```

---

## 4. Phase 別実施計画

### Phase 1.5 — リリース前必須 (1〜3日)

> 既存機能を壊さず、リリースブロッカーを解消する

| ID | タスク | 担当ファイル | 工数 |
|---|---|---|---|
| CRIT-1a | JWT 認証ミドルウェアのスタブ実装（最低限: ヘッダー検証） | `apps/professor/internal/adapters/http/middleware/auth.go` (新規) | M |
| CRIT-1b | `DevUser` を `APP_ENV=development` でのみ有効化 | `apps/professor/cmd/professor/main.go` | S |
| CRIT-2 | Go モジュール名を `apps/professor` に修正 | `apps/professor/go.mod` + 全 `.go` import | M |
| HIGH-3 | 起動時 API キーバリデーション追加 | `config.go` / `main.py` | S |
| MED-5 | `package.json` の name 修正 | `apps/web/package.json` | XS |

**Exit Criteria Phase 1.5:**
- `make verify` が CI でパス
- `APP_ENV=production` 相当で起動した場合、認証なしリクエストが 401 を返す
- `GEMINI_API_KEY` 未設定で起動失敗する

---

### Phase 2 — 技術的負債の体系的解消 (1〜2週)

| ID | タスク | 工数 |
|---|---|---|
| CRIT-1c | JWT 認証を本番 IdP（例: Google Identity）に接続 | L |
| HIGH-1 | sqlcgen 配置の ADR 追加 (`docs/adr/003-sqlcgen-placement.md`) | XS |
| HIGH-2 | librarian shim 層の解消 OR ADR で正式化 | M |
| MED-1 | librarian を `structlog` に統一 | S |
| MED-2 | Professor に OTEL トレーサー設定 + Cloud Trace エクスポーター | M |
| MED-3 | Professor に Echo レート制限ミドルウェア追加 | S |
| MED-4 | librarian のユニット・コントラクトテスト追加 (最低 3 ファイル) | M |
| PR-5 | Phase 1 テスト移行バックログの完了 | M |

---

### Phase 3 — 品質・運用向上 (1ヶ月)

| ID | タスク | 工数 |
|---|---|---|
| LOW-1 | 横断的ドキュメントを `docs/` に統合、サービス docs は差分のみ | L |
| LOW-2 | `openapi.librarian.reference.yaml` リネーム | XS |
| LOW-3 | `researchFindings.md` → `handbook/` 移動、`tmp/` .gitignore | XS |
| - | librarian に Kafka コンシューマー統合 (インジェストジョブ) | L |
| - | Professor の pgvector HNSW インデックスのチューニング + パフォーマンステスト | M |
| - | E2E テスト拡充 (Playwright) | M |

---

## 5. ディレクトリ構造の目標形 (Phase 2 完了後)

```
eduanimaR/
├── apps/
│   ├── professor/
│   │   ├── cmd/professor/main.go
│   │   ├── internal/
│   │   │   ├── adapters/
│   │   │   │   ├── http/
│   │   │   │   │   ├── handlers/
│   │   │   │   │   └── middleware/
│   │   │   │   │       ├── auth.go       ← NEW (JWT)
│   │   │   │   │       ├── devuser.go    ← APP_ENV=development のみ有効
│   │   │   │   │       └── requestlog.go
│   │   │   │   ├── grpc/
│   │   │   │   ├── llm/
│   │   │   │   ├── messaging/
│   │   │   │   ├── postgres/
│   │   │   │   │   ├── sqlcgen/          ← 現状維持 (ADR 003 で明記)
│   │   │   │   │   └── *.go
│   │   │   │   └── storage/
│   │   │   ├── config/
│   │   │   │   └── config.go             ← Validate() 追加
│   │   │   ├── domain/
│   │   │   ├── ports/
│   │   │   └── usecases/
│   │   └── gen/
│   │       └── proto/                    ← buf generate の出力
│   │
│   ├── librarian/
│   │   ├── src/librarian/
│   │   │   ├── __init__.py
│   │   │   ├── config.py                 ← 起動時バリデーション追加
│   │   │   ├── graph.py
│   │   │   ├── main.py
│   │   │   ├── server.py
│   │   │   └── v1/                       ← shim (ADR で正式化 or 削除)
│   │   ├── gen/
│   │   │   └── proto/python/librarian/v1/
│   │   └── tests/
│   │       ├── unit/
│   │       │   ├── test_config.py        ← NEW
│   │       │   ├── test_graph.py
│   │       │   └── test_server.py        ← NEW
│   │       └── contract/
│   │           └── test_proto_ssot.py    ← NEW
│   │
│   └── web/
│       ├── package.json                  ← name: "eduanima-web"
│       ├── src/
│       └── gen/openapi/web/generated/
│
├── contracts/                            ← SSOT (変更なし)
├── docs/
│   └── adr/
│       ├── 001-contract-canonical-paths.md
│       ├── 002-apps-directory-structure.md
│       └── 003-sqlcgen-placement.md      ← NEW
├── ops/
└── handbook/
```

---

## 6. 最優先アクションリスト (今週中)

1. **🔴 CRIT-1b** — `DevUser` ミドルウェアを `APP_ENV != production` でのみ有効化  
   ファイル: `apps/professor/cmd/professor/main.go`

2. **🔴 CRIT-2** — Go モジュール名を `apps/professor` に一括 sed 置換  
   ファイル: `apps/professor/go.mod` + `apps/professor/Makefile`

3. **🟠 HIGH-3** — 起動時バリデーション追加 (両サービス)  
   ファイル: `apps/professor/internal/config/config.go`, `apps/librarian/src/librarian/main.py`

4. **🟡 MED-5** — package.json の name 修正 (5分)  
   ファイル: `apps/web/package.json`

5. **🟢 ADR-003** — sqlcgen 配置の意図的な決定として記録  
   ファイル: `docs/adr/003-sqlcgen-placement.md`

---

## 7. 技術的負債評価サマリー

| 項目 | 現状スコア | Phase 1.5 後 | Phase 2 後 |
|---|---|---|---|
| セキュリティ（認証） | ⚠️ 開発スタブのみ | ✅ 環境ゲート | ✅ JWT 本番対応 |
| コード一貫性 | 🟡 Go module 名不一致 | ✅ 修正済み | ✅ |
| 可観測性 | 🟡 構造化ログのみ | 🟡 | ✅ OTEL + Cloud Trace |
| テストカバレッジ | 🟡 Go は良好、Python は薄い | 🟡 | ✅ 3層テスト完備 |
| ドキュメント | 🟡 重複大 | 🟡 | 🟡 (Phase 3) |
| 契約管理 | ✅ SSOT contracts/ | ✅ | ✅ |
| ビルド再現性 | ✅ Docker Compose + Make | ✅ | ✅ |
| CI/CD | ✅ GitHub Actions 7ワークフロー | ✅ | ✅ |

---

## 関連ドキュメント

- [docs/REFACTORING_PHASE1_BACKLOG.md](REFACTORING_PHASE1_BACKLOG.md) — Phase 1 施策（完了済み）
- [docs/adr/001-contract-canonical-paths.md](adr/001-contract-canonical-paths.md)
- [docs/adr/002-apps-directory-structure.md](adr/002-apps-directory-structure.md)
- [ops/compose/README.md](../ops/compose/README.md) — ローカル開発環境
- [ops/docs/CLOUD_RUN.md](../ops/docs/CLOUD_RUN.md) — Cloud Run デプロイ
