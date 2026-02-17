# CLEAN_ARCHITECTURE

## 目的
`eduanima-librarian`（Python 推論マイクロサービス）のディレクトリ構成と依存方向を固定し、推論ロジック（検索戦略・停止判断・選定）の保守性とテスト容易性を最大化する。

## 前提（重要）
- Librarian は **DB-less**（永続化なし）。DB/インデックス/バッチは Go 側（Professor）の責務。
- Librarian の外部依存は原則として以下のみ:
  - **Gemini API（高速推論モデル）**
  - **Professor が公開する検索ツール（HTTP/JSON）**

## 推奨レイアウト（Python + Clean Architecture）
```
src/
  eduanima_librarian/
    app/            # Litestar app / routing / DI wiring
    controllers/    # HTTP 入出力（request/response, auth, error mapping）
    usecases/       # ユースケース（検索戦略/ループ/停止判断/選定）
    domain/         # ドメインモデル（Evidence, StopCondition, Task, errors）
    ports/          # 外部I/Fの抽象（ProfessorSearchPort, LlmPort 等）
    adapters/       # ports の実装（GeminiClient, ProfessorClient）
    observability/  # logging/tracing/metrics（サービス内で完結）
tests/
```

## 依存方向（必須）
- `controllers` → `usecases` → `domain`
- `adapters` → `ports` / `domain`
- `usecases` は `adapters` に依存しない（必ず `ports` 経由）

## LangGraph の置き場所
- LangGraph（検索ループの状態機械/グラフ）は `usecases` に置く。
- `domain` は LangGraph/Litestar/Gemini SDK などの実装詳細に依存しない。

## 禁止事項（Librarian のガードレール）
- DB 接続・マイグレーション・キャッシュサーバ導入（状態を外に持たない）
- 最終回答文の生成（出力はエビデンスの構造化データのみ）
- 無制限の検索ループ（MaxRetry と停止条件を必ず適用）
