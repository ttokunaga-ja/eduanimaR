# MICROSERVICES_MAP

## 目的
本ドキュメントは、eduanima+R における **Librarian（Python推論マイクロサービス）** のサービス境界（責務）・依存関係・公開IF・運用単位を一覧化し、変更時に「どのサービスを触るべきか」を最短で判断できるようにする。

## 大原則（Librarian SSOT）
- **Librarian は DB を持たない（DB-less）**。永続化・インデックス・検索の物理実行は Go 側（Professor）が担う。
- Librarian は **推論特化**（戦略立案/検索ループ制御/資料選定）。最終回答文の生成は行わない。
- 依存は一方向（循環禁止）: Professor → Librarian（推論委譲） / Librarian → Professor（検索ツール呼び出し）。

## サービス一覧（eduanima+R 実態）
| Service | Responsibility | Owning Data | Exposed APIs | Depends On | Port (dev) |
| --- | --- | --- | --- | --- | --- |
| **eduanima-professor (Go)** | バッチ処理、DB管理、インデックス作成/更新、検索の物理実行（全文/ベクトル）、最終回答生成 | DB・検索インデックス・ドキュメントメタデータ | （外部/内部向けのAPI群。Librarian向けには Search Tool / Doc Fetch 等を提供） | (none) | (TBD) |
| **eduanima-librarian (Python)** | **検索戦略立案・検索ループ制御（LangGraph）・停止判断・エビデンス選定/重複排除** | (none) | **HTTP/JSON**: `POST /v1/librarian/search-agent` | professor(search tools) / Gemini API | 8091 |

> UI や外部 API Gateway の構成は本サービスのスコープ外。

## 依存関係（通信方向）
```
eduanima-professor (Go)
	├─(HTTP/JSON)→ eduanima-librarian (Python)  # 推論の委譲（検索戦略/選定）
	└←(HTTP/JSON)─ eduanima-librarian (Python)  # 検索ツール実行結果の受け渡し

eduanima-librarian (Python)
	└─(HTTPS)→ 高速推論モデル API
```

## 境界の判定ルール（どっちに置く？）
- **DB/インデックス/バッチ/取り込み**が絡む → Professor。
- **「何を探すべきか」「十分か」「どれを証拠にするか」** → Librarian。
- **文章生成（最終回答）** → Professor。

## 変更時のチェックリスト
- Librarian に DB 接続やマイグレーション責務が混入していないか
- Librarian が「最終回答文」を出していないか（証拠の構造化データに限定）
- 検索ループが無限化していないか（MaxRetry/停止条件）
- Professor/Librarian 間の契約（入力/出力）を変更したら OpenAPI/契約テストも更新したか
