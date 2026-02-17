# eduanima+R: Python Microservice Specification (The Librarian)

**Version:** 2026.02.14 (Stable for Gemini 3 Era)

**Service Name:** `eduanima-librarian`

## 0. 目的（このドキュメントが正）
本ドキュメントは `eduanima-librarian` の **役割・責務境界・入出力契約・ガードレール** を SSOT として定義する。

## 1. サービスの役割：知能的専門司書（The Librarian）
`eduanima-librarian` は、システム全体の「探索と判断の脳」を担う **ステートレス推論マイクロサービス**である。

Go サーバー（Professor）が **検索の物理実行・DB/インデックス管理・バッチ処理・最終回答生成** を行うのに対し、Librarian は以下に特化する:

- 「どの資料が回答に必要か」を自律的に特定する
- 検索ループを制御し、停止条件を満たすまでクエリを改善する
- 最終回答生成モデル（Professor）に渡すための **最小かつ重複のないエビデンス集合** を確定する

> Librarian は DB に直接接続しない。

## 2. Python側の核心的責務

### 2.1 自律的検索戦略の立案（Search Strategy & Thinking）
- Professor から渡された「分析済み情報（ユーザー意図）」と「タスク・停止条件」に基づき、検索戦略を策定する。
- 高速推論モデル を用い、ヒット率を最大化するクエリ（言い換え/制約/複合）を生成する。

### 2.2 LangGraph による循環型検索の制御（Agentic Loop）
- **検索の実行はしない**。Professor が提供する検索インターフェース（Tool）へリクエストを発行する。
- Professor から返された検索候補（テキスト断片 + 付随情報）を評価し、停止条件を満たすか判定する。
- 不足する場合はクエリを修正し、最大 `max_retries` 回まで検索ループを回す。

重要:
- **取得件数（k）など「物理検索」のパラメータは Professor が所有**し、Librarian は指定しない。
- **LLM には安定ID（document_id 等）を見せない**。検索候補は `temp_index` で参照し、安定IDへの対応は Professor が保持する。

推奨ノード（例）:
- `Plan`（検索戦略の組み立て）
- `CallSearchTool(Professor)`
- `Evaluate`（充足性/停止判断）
- `Refine`（クエリ修正）

### 2.3 最終資料セットの確定と重複排除（Deduplication & Selection）
- 複数の検索試行で得られた結果から、回答に不可欠な参照（`temp_index`）を厳選する。
- 同一資料・同一ページの重複を排除し、最小のエビデンス集合を返す。

## 3. 採用技術スタック（Librarian）

### 3.1 推論モデル：高速推論モデル
- **採用状況（2026-02-14 時点）**: Librarian の推論（戦略立案/停止判断/選定）における標準モデル。
- **選定理由**: 高速・低コストの推論により、検索ループを現実的なレイテンシで回せる。
- **役割**: 質問解析、検索クエリ生成、情報充足性判定、停止判断。

> 具体的な SDK/認証方式・モデル切替の運用は `02_tech_stack/STACK.md` を正とする。

### 3.2 Web フレームワーク：Litestar
- **選定理由**: 高速な ASGI、型・バリデーション設計との相性、軽量なサービス実装に適する。
- **役割**: ステートレス推論エンドポイント（HTTP/JSON）の提供。

### 3.3 データ検証：msgspec
- **選定理由**: 高速な JSON パースとスキーマ厳格化。
- **役割**: Professor ↔ Librarian の入出力契約を強制し、運用時の契約逸脱を早期に検知する。

### 3.4 エージェント制御：LangGraph (Python)
- **選定理由**: 検索ループの state と分岐（停止条件/最大試行回数）をグラフで明示できる。
- **役割**: ループの制御・最大試行回数の保証。

## 4. ワークフローとインターフェース設計

### 4.1 処理シーケンス
1. **Professor → Librarian**: ユーザー質問、Professor 側の分析結果、タスク、停止条件を送信。
2. **Librarian**: 高速推論モデル が戦略を策定。
3. **Librarian → Professor**: 検索クエリを Request（ツール呼び出し）。
4. **Professor → Librarian**: 検索候補（LLM可視: `temp_index` + テキスト断片）を Response。
5. **Librarian**: 充足性を判断。不足なら 3 へ戻る（最大 `max_retries` 回）。
6. **Librarian → Professor**: 選定した `temp_index` の集合を返却（Professor が安定参照へ変換）。

### 4.2 主要 API エンドポイント
`POST /v1/librarian/search-agent`

#### Input（Professor → Librarian）
```json
{
  "user_query": "決定係数の計算式を教えて",
  "analyzed_info": {
    "intent": "formula_lookup",
    "context": "Lecture 03: Regression Analysis"
  },
  "task": {
    "target_items": ["definition", "formula", "interpretation_0_1"],
    "max_retries": 5
  },
  "stop_conditions": "All target_items are found with specific references."
}
```

補足:
- `max_retries` は **Librarian の推論ループ上限**（無限ループ防止）のための値。
- 物理検索の試行回数や取得件数調整（k）に使う `retry_count` 等の状態は **Professor 側で保持**する。

#### Output（Librarian → Professor）
```json
{
  "status": "COMPLETED",
  "selected_evidence": [
    { "temp_index": 1, "why": "Contains the definition and formula." },
    { "temp_index": 4, "why": "Explains the 0/1 interpretation." }
  ],
  "reasoning_summary": "Selected the minimum snippets that satisfy all target_items."
}
```

## 5. 実装上のガードレール（Librarian's Oath）
1. **DBレスの徹底:** Librarian は DB/検索基盤へ直接接続しない。
2. **ステートレス:** 会話履歴・キャッシュ等の永続化を持たず、リクエスト内 state に閉じる。
3. **推論の特化:** 最終回答文は生成しない。出力は Professor が安定参照へ変換できる構造化データ（`temp_index` 等）。
4. **無限ループ防止:** `max_retries` を厳守し、未達でも現時点の最善で終了する。

## 6. eduanima+R における意義
Librarian は、Professor（表現者/統合者）を支える「究極の調査員」として、検索コストを抑えつつ引用品質を引き上げる。

高速推論モデル の推論能力を、検索ループの「粘り強さ」と「正確な停止判断」に集中投下することで、不必要な資料読み込みを減らし、最終回答の精度とコスト効率を高める。
