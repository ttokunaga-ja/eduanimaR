"""
LangGraph ベースの推論エージェント

Gemini Structured Output を使用してクエリ生成・検索結果評価を行う。

SDK: google-genai（新公式SDK）
  旧: google-generativeai（非推奨）→ 新: google-genai

Temperature 設計:
  - build_search_queries  (クエリ生成)   : Temperature=0.3
      多様なクエリバリエーションで検索カバレッジを向上させる。
  - evaluate_search_results (充足度評価) : Temperature=0.0 + Seed=42
      決定論的な終了判断を強制する（ランダム性は誤判断の原因）。
      Seed=42 で再現性を確保する（Gemini 3 ベストエフォート決定論）。
  - select_evidence (エビデンス選択)     : LLM なし
      スコア順位による機械的選択（再スコアリングは Phase 3 で検討）。

ループ上限: max_loops (デフォルト 5)
  平均 3 回で収束する設計だが、上限 5 回で強制終了する。
  実際のループ制御は server.py の gRPC ストリームで行う。
"""

from __future__ import annotations

import json
from typing import Any, NotRequired, TypedDict

import structlog

logger = structlog.get_logger(__name__)


# ─── Structured Output スキーマ定義 ──────────────────────────────────


class _QueryItem(TypedDict):
    """クエリ生成の Structured Output スキーマ。"""

    query: str      # 検索クエリ文字列
    rationale: str  # このクエリを選んだ理由（監査・デバッグ用）


class _EvalResult(TypedDict):
    """検索結果評価の Structured Output スキーマ。"""

    should_continue: bool       # True: 再検索が必要、False: CompleteAction へ
    reason: str                 # 判断理由（ログ・監査用）
    missing_aspects: list[str]  # 不足している情報の側面（次のクエリ生成に使用）
    refined_query_hint: NotRequired[str]  # 次の検索クエリへのヒント（任意）


# ─── Gemini クライアント（モジュールレベルシングルトン） ─────────────

_gemini_client = None    # google.genai.Client シングルトン
_gemini_model_name: str | None = None  # 使用するモデル名


def init_gemini(api_key: str, model_name: str) -> None:
    """
    Gemini クライアントを初期化する（起動時に一度だけ呼ぶ）。

    新 SDK（google-genai）は Client を 1 インスタンス作成し、
    generate_content 呼び出し時に GenerateContentConfig でパラメータを指定する。
    クエリ生成と充足度評価で異なる Temperature/Seed を使用するため、
    設定は関数呼び出し時に渡す。

    Args:
        api_key: GEMINI_API_KEY
        model_name: 使用するモデル名（LIBRARIAN_MODEL_SEARCH 環境変数から設定）
    """
    global _gemini_client, _gemini_model_name
    try:
        from google import genai

        _gemini_client = genai.Client(api_key=api_key)
        _gemini_model_name = model_name
        logger.info("Gemini クライアント初期化完了", model=model_name)
    except Exception as exc:
        logger.error("Gemini クライアント初期化失敗", error=str(exc))
        _gemini_client = None
        _gemini_model_name = None


# ─── グラフ状態 ─────────────────────────────────────────────────────


class AgentState(TypedDict):
    """LangGraph が管理する状態スキーマ。"""

    request_id: str
    user_query: str
    subject_id: str

    # 検索結果（Professor から受け取った生データ）
    search_results: list[dict[str, Any]]

    # 推論ループカウンタ
    loop_count: int

    # 最終的なエビデンスインデックスリスト
    evidence_indices: list[int]

    # エラー情報
    error: str | None


# ─── クエリ生成 ──────────────────────────────────────────────────────


def build_search_queries(
    user_query: str,
    loop_count: int,
    search_results: list[dict[str, Any]],
    missing_aspects: list[str] | None = None,
) -> tuple[list[str], str]:
    """
    Gemini Structured Output でユーザークエリから検索クエリ群を生成する。
    Temperature=0.3: 多様なクエリバリエーションで検索カバレッジを向上させる。

    フォールバック: Gemini 利用不可の場合はユーザークエリをそのまま返す。

    Args:
        user_query: ユーザーの質問
        loop_count: 現在のループ番号（0 始まり）
        search_results: 前回までの検索結果
        missing_aspects: evaluate_search_results で検出された不足情報の側面

    Returns:
        (queries, rationale)
        - queries: 検索クエリのリスト
        - rationale: なぜこれらのクエリを選んだか（gRPC の rationale フィールドに渡す）
    """
    if _gemini_client is None or _gemini_model_name is None:
        # フォールバック: LLM なし（Gemini 未初期化時）
        if loop_count == 0:
            return [user_query], f"ユーザークエリ「{user_query}」に関連するチャンクを検索します"
        words = user_query.split()
        q = " ".join(words[:3]) if len(words) > 3 else user_query
        return [q], f"リファインクエリ「{q}」で再検索します（フォールバック）"

    # プロンプト構築
    if loop_count == 0 or not missing_aspects:
        prompt = f"""You are an academic search query generator.
Generate diverse search queries to find relevant course material chunks for the user's question.

User question: {user_query}

Generate 2-3 distinct search queries that approach the question from different angles.
Queries should be concise and use academic terminology."""
    else:
        missing_str = "\n".join(f"- {a}" for a in missing_aspects)
        prompt = f"""You are an academic search query generator.
The previous search was insufficient. Generate refined queries to find the missing information.

User question: {user_query}

Missing aspects identified by the evaluator:
{missing_str}

Generate 2-3 refined search queries that specifically target these missing aspects."""

    try:
        from google.genai import types

        response = _gemini_client.models.generate_content(
            model=_gemini_model_name,
            contents=prompt,
            config=types.GenerateContentConfig(
                response_mime_type="application/json",
                response_schema=list[_QueryItem],
                # Temperature=0.3: 多様なクエリバリエーションで検索カバレッジを向上
                temperature=0.3,
            ),
        )
        raw_text = response.text or ""
        items: list[_QueryItem] = json.loads(raw_text)
        queries = [item["query"] for item in items if item.get("query")]
        # rationale は最初のクエリの理由を使用（gRPC の rationale フィールドは単一文字列）
        rationale = items[0].get("rationale", "") if items else ""
        if not queries:
            return [user_query], f"ユーザークエリ「{user_query}」に関連するチャンクを検索します"
        logger.debug("クエリ生成完了", queries_count=len(queries), loop_count=loop_count)
        return queries, rationale or f"クエリ生成完了（ループ {loop_count + 1}）"
    except Exception as exc:
        logger.warning("クエリ生成失敗、フォールバック使用", error=str(exc))
        return [user_query], f"ユーザークエリ「{user_query}」に関連するチャンクを検索します（フォールバック）"


# ─── 検索結果評価 ────────────────────────────────────────────────────


def evaluate_search_results(
    user_query: str,
    search_results: list[dict[str, Any]],
    loop_count: int,
    max_loops: int,
) -> tuple[bool, list[str], str]:
    """
    Gemini Structured Output で検索結果の充足度を評価する。
    Temperature=0.0 + Seed=42: 決定論的な終了判断（Gemini 3 ベストエフォート再現性）。

    Args:
        user_query: ユーザーの質問
        search_results: Professor から受け取った最新の検索結果
        loop_count: 現在のループ番号（1 以上）
        max_loops: ループ上限

    Returns:
        (should_continue, missing_aspects, reason)
        - should_continue: True なら再検索、False なら CompleteAction へ
        - missing_aspects: 不足している情報の側面（次のクエリ生成に使用）
        - reason: 判断理由（ログ・監査用）
    """
    # ループ上限チェック（LLM 呼び出し前に判定）
    if loop_count >= max_loops:
        return False, [], f"最大ループ数 {max_loops} に達しました（強制終了）"

    # 検索結果が空の場合はループ不要
    if not search_results:
        return False, [], "検索結果が空です"

    if _gemini_client is None or _gemini_model_name is None:
        # フォールバック: 1 回検索したら完了（Gemini 未初期化時）
        return False, [], "LLM 未初期化: 1 回の検索で完了（フォールバック）"

    # 検索結果のスニペットを構築（上位 5 件をプロンプトに含める）
    snippets_parts = []
    for i, r in enumerate(search_results[:5]):
        score = r.get("score", "?")
        content = r.get("content", "")[:300]
        score_str = f"{score:.3f}" if isinstance(score, float) else str(score)
        snippets_parts.append(f"[Chunk {i + 1}] (score: {score_str})\n{content}")
    snippets = "\n\n".join(snippets_parts)

    prompt = f"""You are an academic research evaluator assessing search result sufficiency.

User question: {user_query}

Retrieved chunks (top 5 of {len(search_results)} results):
{snippets}

Evaluate whether these chunks are sufficient to construct a complete answer.
Consider:
- Are the key concepts adequately covered?
- Are definitions, formulas, or examples present where needed?
- Is there enough evidence to answer the question thoroughly?

If insufficient, list the specific aspects that are still missing.
Be concise in your assessment."""

    try:
        from google.genai import types

        response = _gemini_client.models.generate_content(
            model=_gemini_model_name,
            contents=prompt,
            config=types.GenerateContentConfig(
                response_mime_type="application/json",
                response_schema=_EvalResult,
                # Temperature=0.0 + Seed=42:
                #   決定論的な終了判断を強制する（ランダム性は誤判断・無限ループの原因）。
                #   Gemini 3 は Seed をベストエフォートで尊重する。
                temperature=0.0,
                seed=42,
            ),
        )
        raw_text = response.text or ""
        result: _EvalResult = json.loads(raw_text)
        should_continue = bool(result.get("should_continue", False))
        missing_aspects = list(result.get("missing_aspects", []))
        reason = str(result.get("reason", ""))
        logger.debug(
            "検索結果評価完了",
            should_continue=should_continue,
            missing_count=len(missing_aspects),
            loop_count=loop_count,
        )
        return should_continue, missing_aspects, reason
    except Exception as exc:
        logger.warning("検索結果評価失敗、フォールバック使用", error=str(exc))
        return False, [], f"評価失敗（フォールバック）: {exc}"


# ─── エビデンス選択 ──────────────────────────────────────────────────


def select_evidence(search_results: list[dict[str, Any]], max_results: int) -> list[dict[str, Any]]:
    """
    検索結果から上位 N 件を選択してエビデンスリストを返す。

    LLM を使わず、スコア順位で機械的に選択する。
    Gemini による再スコアリングは Phase 3 で検討する。

    Args:
        search_results: Professor から受け取った検索結果
        max_results: 最大件数

    Returns:
        [{"temp_index": int, "why_relevant": str}, ...]
    """
    evidence = []
    limit = min(len(search_results), max_results)
    for i in range(limit):
        evidence.append(
            {
                "temp_index": i,
                "why_relevant": f"検索スコア上位 {i + 1} 位のチャンク",
            }
        )
    return evidence


# ─── LangGraph ノード関数 ─────────────────────────────────────────────


def should_continue_node(state: AgentState) -> str:
    """
    ループ終了条件を判定するエッジ関数（LangGraph グラフ用）。

    Note:
        実際のマルチループ制御は server.py の gRPC ストリームで行う。
        LangGraph グラフは補助的な状態管理ツールとして使用する。

    Returns:
        "search" | "complete"
    """
    if state.get("error"):
        return "complete"
    if state["loop_count"] >= 1 and len(state["search_results"]) > 0:
        return "complete"
    return "search"


def node_search(state: AgentState) -> AgentState:
    """SEARCH アクションの状態更新（クエリ生成のみ、実際の検索は Professor が行う）。"""
    logger.debug(
        "node_search",
        loop_count=state["loop_count"],
        request_id=state["request_id"],
    )
    return state


def node_complete(state: AgentState) -> AgentState:
    """COMPLETE アクションの状態更新（エビデンス選択）。"""
    logger.debug(
        "node_complete",
        results_count=len(state["search_results"]),
        request_id=state["request_id"],
    )
    evidence = select_evidence(state["search_results"], max_results=10)
    state["evidence_indices"] = [e["temp_index"] for e in evidence]
    return state


# ─── グラフ構築 ─────────────────────────────────────────────────────


def build_graph():
    """LangGraph StateGraph を構築して返す。"""
    try:
        from langgraph.graph import END, START, StateGraph

        builder = StateGraph(AgentState)
        builder.add_node("search", node_search)
        builder.add_node("complete", node_complete)

        builder.add_edge(START, "search")
        builder.add_conditional_edges(
            "search",
            should_continue_node,
            {"search": "search", "complete": "complete"},
        )
        builder.add_edge("complete", END)

        return builder.compile()

    except ImportError:
        logger.warning("langgraph が利用できません。フォールバックモードで動作します。")
        return None


_graph = None


def get_graph():
    """グラフのシングルトンを返す。"""
    global _graph
    if _graph is None:
        _graph = build_graph()
    return _graph


def deserialize_state(state_json: str) -> list[dict[str, Any]]:
    """
    Professor から受け取った state JSON を検索結果リストに変換する。

    JSON スキーマ:
    {
      "search_results": [
        {"chunk_id": "...", "content": "...", "score": 0.9, ...},
        ...
      ]
    }
    """
    if not state_json:
        return []
    try:
        data = json.loads(state_json)
        return data.get("search_results", [])
    except json.JSONDecodeError:
        logger.warning("state JSON のデシリアライズに失敗しました: %s", state_json[:100])
        return []
