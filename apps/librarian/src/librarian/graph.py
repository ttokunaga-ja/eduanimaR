"""
LangGraph ベースの推論エージェント（Phase 1 簡略版）

Phase 1 スコープ:
  - Gemini Flash でシンプルな検索クエリ生成
  - 最大 max_loops ループで SEARCH → 結果受信 → COMPLETE のサイクルを実行
  - LangGraph の StateGraph で状態を管理

Phase 3 では:
  - マルチステップ推論
  - 検索結果の自己評価・リクエスト再発行
  - ツール呼び出し（grounding）
"""

from __future__ import annotations

import json
import logging
from typing import Any, TypedDict

logger = logging.getLogger(__name__)


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


# ─── クエリ生成ヘルパー ──────────────────────────────────────────────


def build_search_queries(user_query: str, loop_count: int, search_results: list[dict[str, Any]]) -> list[str]:
    """
    Phase 1: ユーザークエリを直接使用（LLM呼び出しなし）。
    Phase 3 で Gemini を使ったクエリ生成に置き換える。

    Args:
        user_query: ユーザーの質問
        loop_count: 現在のループ番号（0始まり）
        search_results: 前回までの検索結果

    Returns:
        検索クエリのリスト
    """
    if loop_count == 0:
        # 初回: そのまま使用
        return [user_query]
    else:
        # 2回目以降: キーワード抽出（Phase 1では簡略化）
        # Phase 3 で Gemini によるクエリリファインメントに置き換え
        words = user_query.split()
        return [" ".join(words[:3])] if len(words) > 3 else [user_query]


def select_evidence(search_results: list[dict[str, Any]], max_results: int) -> list[dict[str, Any]]:
    """
    検索結果から上位 N 件を選択してエビデンスリストを返す。

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


def should_continue(state: AgentState) -> str:
    """
    ループ終了条件を判定するエッジ関数。

    Returns:
        "search" | "complete"
    """
    if state.get("error"):
        return "complete"
    if state["loop_count"] >= 1 and len(state["search_results"]) > 0:
        # Phase 1: 1回検索したら完了
        return "complete"
    return "search"


def node_search(state: AgentState) -> AgentState:
    """SEARCH アクションの状態更新（クエリ生成のみ、実際の検索は Professor が行う）。"""
    logger.debug(
        "node_search",
        extra={"loop_count": state["loop_count"], "request_id": state["request_id"]},
    )
    # このノードはクエリを生成するだけ。
    # 実際の検索実行・結果受信は server.py の gRPC ストリームで行う。
    return state


def node_complete(state: AgentState) -> AgentState:
    """COMPLETE アクションの状態更新（エビデンス選択）。"""
    logger.debug(
        "node_complete",
        extra={"results_count": len(state["search_results"]), "request_id": state["request_id"]},
    )
    evidence = select_evidence(state["search_results"], max_results=10)
    state["evidence_indices"] = [e["temp_index"] for e in evidence]
    return state


# ─── グラフ構築 ─────────────────────────────────────────────────────


def build_graph():
    """
    LangGraph StateGraph を構築して返す。

    Phase 1 では: START → search_node → complete_node → END
    Phase 3 では: search_node からループバックを追加
    """
    try:
        from langgraph.graph import END, START, StateGraph

        builder = StateGraph(AgentState)
        builder.add_node("search", node_search)
        builder.add_node("complete", node_complete)

        builder.add_edge(START, "search")
        builder.add_conditional_edges(
            "search",
            should_continue,
            {"search": "search", "complete": "complete"},
        )
        builder.add_edge("complete", END)

        return builder.compile()

    except ImportError:
        logger.warning("langgraph が利用できません。フォールバックモードで動作します。")
        return None


# グラフのシングルトン（起動時に一度だけ構築）
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
