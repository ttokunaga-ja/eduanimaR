"""
graph.py のユニットテスト

LangGraph ノード・ヘルパー関数の挙動を検証する。
LLM/gRPC 呼び出しは不要なため、外部依存なしで高速に実行できる。
"""
from __future__ import annotations

import pytest

from librarian.graph import (
    AgentState,
    build_search_queries,
    select_evidence,
    should_continue,
    node_search,
    node_complete,
)


# ─── build_search_queries ─────────────────────────────────────────


class TestBuildSearchQueries:
    def test_初回ループではユーザークエリをそのまま返す(
        self, sample_user_query: str
    ) -> None:
        queries = build_search_queries(
            user_query=sample_user_query,
            loop_count=0,
            search_results=[],
        )
        assert queries == [sample_user_query]

    def test_2回目以降は短縮クエリを返す(self, sample_user_query: str) -> None:
        queries = build_search_queries(
            user_query=sample_user_query,
            loop_count=1,
            search_results=[{"content": "既存結果"}],
        )
        # Phase 1: 先頭3単語
        assert len(queries) == 1
        assert len(queries[0].split()) <= 3

    def test_短いクエリはそのまま返す(self) -> None:
        short_query = "量子力学"
        queries = build_search_queries(
            user_query=short_query,
            loop_count=1,
            search_results=[],
        )
        assert queries == [short_query]


# ─── select_evidence ──────────────────────────────────────────────


class TestSelectEvidence:
    def test_max_results件数以内のエビデンスを返す(
        self, sample_search_results: list[dict]
    ) -> None:
        evidence = select_evidence(sample_search_results, max_results=2)
        assert len(evidence) == 2

    def test_結果が少ない場合は全件返す(
        self, sample_search_results: list[dict]
    ) -> None:
        evidence = select_evidence(sample_search_results, max_results=100)
        assert len(evidence) == len(sample_search_results)

    def test_空の検索結果では空リストを返す(
        self, empty_search_results: list[dict]
    ) -> None:
        evidence = select_evidence(empty_search_results, max_results=10)
        assert evidence == []

    def test_インデックスが0始まりで連番(
        self, sample_search_results: list[dict]
    ) -> None:
        evidence = select_evidence(sample_search_results, max_results=3)
        for i, e in enumerate(evidence):
            assert e["temp_index"] == i

    def test_why_relevantフィールドが存在する(
        self, sample_search_results: list[dict]
    ) -> None:
        evidence = select_evidence(sample_search_results, max_results=1)
        assert "why_relevant" in evidence[0]
        assert evidence[0]["why_relevant"]  # 空文字でないこと


# ─── should_continue ──────────────────────────────────────────────


class TestShouldContinue:
    def _make_state(
        self,
        loop_count: int = 0,
        search_results: list | None = None,
        error: str | None = None,
    ) -> AgentState:
        return AgentState(
            request_id="test-req",
            user_query="テスト質問",
            subject_id="subject-001",
            search_results=search_results or [],
            loop_count=loop_count,
            evidence_indices=[],
            error=error,
        )

    def test_初回ループでは検索を続ける(self) -> None:
        state = self._make_state(loop_count=0, search_results=[])
        assert should_continue(state) == "search"

    def test_1回検索後に結果があれば完了(
        self, sample_search_results: list[dict]
    ) -> None:
        state = self._make_state(loop_count=1, search_results=sample_search_results)
        assert should_continue(state) == "complete"

    def test_エラーがあれば完了(self) -> None:
        state = self._make_state(loop_count=0, error="something went wrong")
        assert should_continue(state) == "complete"


# ─── node_search ──────────────────────────────────────────────────


class TestNodeSearch:
    def test_状態を変更せずに返す(self, sample_user_query: str) -> None:
        state: AgentState = {
            "request_id": "req-1",
            "user_query": sample_user_query,
            "subject_id": "sub-1",
            "search_results": [],
            "loop_count": 0,
            "evidence_indices": [],
            "error": None,
        }
        result = node_search(state)
        assert result is state  # 同一オブジェクトを返すこと


# ─── node_complete ────────────────────────────────────────────────


class TestNodeComplete:
    def test_エビデンスインデックスが設定される(
        self, sample_search_results: list[dict]
    ) -> None:
        state: AgentState = {
            "request_id": "req-1",
            "user_query": "質問",
            "subject_id": "sub-1",
            "search_results": sample_search_results,
            "loop_count": 1,
            "evidence_indices": [],
            "error": None,
        }
        result = node_complete(state)
        assert len(result["evidence_indices"]) > 0

    def test_空の検索結果ではエビデンスインデックスが空(self) -> None:
        state: AgentState = {
            "request_id": "req-1",
            "user_query": "質問",
            "subject_id": "sub-1",
            "search_results": [],
            "loop_count": 1,
            "evidence_indices": [],
            "error": None,
        }
        result = node_complete(state)
        assert result["evidence_indices"] == []
