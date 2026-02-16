# STACK（eduanima-librarian SSOT）

## サマリ
`eduanima-librarian` は **Python のステートレス推論マイクロサービス**。
Gemini 3 Flash を用いて検索戦略・停止判断・エビデンス選定を行い、検索の物理実行は Go 側（Professor）へ委譲する。

## 技術スタック（2026-02-14 時点の方針）

| 項目 | 採用 | 備考 |
| :--- | :--- | :--- |
| **Runtime** | Python（ASGI） | 具体バージョンはプロジェクトで pin する（例: 3.12+）。本サービスは DB を持たない。 |
| **Web Framework** | Litestar | gRPCサーバー実装（`/v1/librarian/search-agent`）。 |
| **Schema/Serialization** | msgspec / Protocol Buffers | Professor ↔ Librarian の契約（DTO）を高速・厳格に扱う。gRPC契約は Protocol Buffers。 |
| **Agent Orchestration** | LangGraph | 検索ループの状態管理、MaxRetry/停止条件の保証。 |
| **HTTP Client** | （例: httpx） | Professor の検索ツール呼び出し（gRPC経由）、および Gemini API 呼び出しに使用。 |
| **LLM** | Gemini 3 Flash | Librarian の標準推論モデル（戦略立案/停止判断/選定）。 |
| **Observability** | OpenTelemetry（Python） | trace/log correlation を前提にする（request_id/trace_id）。 |
| **Packaging/Build** | pyproject ベース | 依存は lock して再現性を確保（ツールはプロジェクトで固定）。 |

## Gemini 3 Flash の使い分け（推奨）
- 戦略立案（Plan）: 思考コストを許容（例: medium 相当）
- ループ中の微修正（Refine/Evaluate）: 低コスト（例: low 相当）

> 具体のパラメータ名/SDKは採用SDKに依存するため、実装側で SSOT 化する。

## SSOT（Single Source of Truth）
- Librarian の仕様: `01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md`
- 責務境界: `01_architecture/MICROSERVICES_MAP.md`
- gRPC 契約: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`

## 明確に「やらない」こと
- DB/インデックス/バッチ処理（Professor の責務）
- gRPC 以外の内部 RPC 方式の独自採用（Professor との契約は gRPC/Proto が正）

