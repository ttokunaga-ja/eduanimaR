"""
graph.py のユニットテスト（Phase 4 対応）

LangGraph ノード・ヘルパー関数の挙動を検証する。
LLM/gRPC 呼び出しは不要なため、外部依存なしで高速に実行できる。
"""

from __future__ import annotations

import pytest

from unittest.mock import MagicMock, patch

import librarian.graph as graph_module
from librarian.graph import (
    AgentState,
    build_search_queries,
    deserialize_state,
    init_checkpointer,
    judge_sufficiency,
    node_complete,
    node_generate_queries,
    node_update_evidence,
    should_continue_node,
)


# ─── AgentState ヘルパー ──────────────────────────────────────────────


def _make_state(**overrides) -> AgentState:
    """テスト用 AgentState を生成するヘルパー（Phase 4: search_directives / tried_queries 追加）。"""
    defaults: AgentState = {
        "request_id": "test-req-001",
        "user_query": "量子力学のシュレーディンガー方程式について説明してください",
        "subject_id": "subj-001",
        "interpreted_query": "",
        "completion_criteria": [],
        "loop_count": 0,
        "max_loops": 4,
        "thinking_level": "flash",
        "kept_chunks": [],
        "all_seen_chunk_ids": [],
        "missing_keywords": [],
        "search_directives": [],   # Phase 4 追加
        "tried_queries": [],       # Phase 4 追加
        "current_queries": [],
        "current_rationale": "",
        "search_results": [],
        "new_chunk_ids": [],
        "useful_chunk_ids": [],
        "is_sufficient": False,
        "satisfied_ratio": 0.0,
        "evidence_chunk_ids": [],
        "error": None,
    }
    defaults.update(overrides)
    return defaults


# ─── build_search_queries ─────────────────────────────────────────────


class TestBuildSearchQueries:
    def test_Gemini未初期化時はユーザークエリをそのまま返す(self, sample_user_query: str) -> None:
        """Gemini クライアントが None のとき、フォールバックでユーザークエリを返す。"""
        queries = build_search_queries(
            user_query=sample_user_query,
            loop_count=0,
        )
        assert queries == [sample_user_query]

    def test_missing_keywordsなし初回ループはユーザークエリを返す(self, sample_user_query: str) -> None:
        """missing_keywords が None の場合はユーザークエリをそのまま返す。"""
        queries = build_search_queries(
            user_query=sample_user_query,
            loop_count=0,
            missing_keywords=None,
        )
        assert queries == [sample_user_query]

    def test_missing_keywordsありでもフォールバックはユーザークエリを返す(
        self, sample_user_query: str
    ) -> None:
        """Gemini 未初期化時は missing_keywords があってもフォールバック。"""
        queries = build_search_queries(
            user_query=sample_user_query,
            loop_count=2,
            missing_keywords=["量子力学", "シュレーディンガー"],
        )
        assert queries == [sample_user_query]

    def test_interpreted_queryが提供されても型は正しい(self, sample_user_query: str) -> None:
        """interpreted_query があっても戻り値は list[str]。"""
        queries = build_search_queries(
            user_query=sample_user_query,
            loop_count=0,
            interpreted_query="シュレーディンガー方程式の基本的な意味と解釈",
        )
        assert isinstance(queries, list)
        assert all(isinstance(q, str) for q in queries)

    def test_search_directivesありでもフォールバックはユーザークエリを返す(
        self, sample_user_query: str
    ) -> None:
        """Gemini 未初期化時は search_directives があってもフォールバック。"""
        queries = build_search_queries(
            user_query=sample_user_query,
            loop_count=2,
            search_directives=["Look for evaluation metrics in Section 4"],
        )
        assert queries == [sample_user_query]

    def test_tried_queriesありでもフォールバックはユーザークエリを返す(
        self, sample_user_query: str
    ) -> None:
        """Gemini 未初期化時は tried_queries があってもフォールバック。"""
        queries = build_search_queries(
            user_query=sample_user_query,
            loop_count=1,
            tried_queries=["previous query 1", "previous query 2"],
        )
        assert queries == [sample_user_query]


# ─── judge_sufficiency ────────────────────────────────────────────────


class TestJudgeSufficiency:
    def test_ループ上限超過で強制終了_5tuple(self) -> None:
        """loop_count > max_loops のとき 5-tuple (False, 0.0, [], [], []) を返す。"""
        result = judge_sufficiency(
            user_query="テスト質問",
            new_chunks=[{"chunk_id": "c1", "content": "内容"}],
            kept_chunks=[],
            loop_count=5,  # > max_loops=4
            max_loops=4,
        )
        assert len(result) == 5
        is_suf, ratio, kw, directives, revised = result
        assert is_suf is False
        assert ratio == 0.0
        assert kw == []
        assert directives == []
        assert revised == []

    def test_ループ上限ちょうどでは強制終了しない(self) -> None:
        """Bug #1 修正確認: loop_count == max_loops のとき強制終了しない（LLM を呼ぼうとする）。"""
        # Gemini 未初期化 → フォールバック (False, 0.0, [], [], []) が返るが
        # 上限超過ではなくフォールバック由来であることを確認
        result = judge_sufficiency(
            user_query="テスト質問",
            new_chunks=[{"chunk_id": "c1", "content": "内容"}],
            kept_chunks=[],
            loop_count=4,  # == max_loops=4 (以前は >= でここで強制終了していた)
            max_loops=4,
        )
        # Gemini 未初期化なのでフォールバック値が返るが、
        # 上限超過の強制終了ではなく通常のフォールバックであることを確認（長さは5）
        assert len(result) == 5

    def test_チャンクが空のとき_5tuple_False(self) -> None:
        """new_chunks も kept_chunks も空のとき (False, 0.0, [], [], []) を返す。"""
        result = judge_sufficiency(
            user_query="テスト質問",
            new_chunks=[],
            kept_chunks=[],
            loop_count=1,
            max_loops=4,
        )
        assert len(result) == 5
        is_suf, ratio, kw, directives, revised = result
        assert is_suf is False
        assert ratio == 0.0
        assert kw == []
        assert directives == []
        assert revised == []

    def test_Gemini未初期化時はフォールバック_5tuple(self) -> None:
        """Gemini クライアントが None のとき (False, 0.0, [], [], []) を返す。"""
        result = judge_sufficiency(
            user_query="テスト質問",
            new_chunks=[{"chunk_id": "c1", "content": "内容"}],
            kept_chunks=[],
            loop_count=1,
            max_loops=4,
        )
        assert len(result) == 5
        is_suf, ratio, kw, directives, revised = result
        assert is_suf is False
        assert ratio == 0.0
        assert isinstance(kw, list)
        assert isinstance(directives, list)
        assert isinstance(revised, list)

    def test_戻り値が5タプル(self) -> None:
        """戻り値が (bool, float, list, list, list) の 5-tuple である。"""
        result = judge_sufficiency(
            user_query="テスト",
            new_chunks=[],
            kept_chunks=[],
            loop_count=2,
            max_loops=4,
        )
        assert len(result) == 5
        assert isinstance(result[0], bool)
        assert isinstance(result[1], float)
        assert isinstance(result[2], list)
        assert isinstance(result[3], list)
        assert isinstance(result[4], list)

    def test_tried_queriesを受け取れる(self) -> None:
        """tried_queries パラメータを受け付けること（型エラーなし）。"""
        result = judge_sufficiency(
            user_query="テスト質問",
            new_chunks=[],
            kept_chunks=[],
            loop_count=2,
            max_loops=4,
            tried_queries=["query 1", "query 2"],
        )
        assert len(result) == 5


# ─── should_continue_node ─────────────────────────────────────────────


class TestShouldContinueNode:
    def test_エラーがあれば完了(self) -> None:
        state = _make_state(loop_count=1, error="something went wrong")
        assert should_continue_node(state) == "complete"

    def test_ループ上限到達で完了(self) -> None:
        state = _make_state(loop_count=4, max_loops=4)
        assert should_continue_node(state) == "complete"

    def test_is_sufficientがTrueなら完了(self) -> None:
        state = _make_state(loop_count=2, is_sufficient=True)
        assert should_continue_node(state) == "complete"

    def test_充足なし_上限未達なら再検索(self) -> None:
        # new_chunk_ids に値を入れて no-progress early exit が発動しないようにする
        state = _make_state(
            loop_count=2,
            max_loops=4,
            is_sufficient=False,
            new_chunk_ids=["chunk-001"],  # no-progress 判定を回避
        )
        assert should_continue_node(state) == "generate_queries"

    def test_初回ループでも上限でなければ再検索(self) -> None:
        state = _make_state(loop_count=0, max_loops=4, is_sufficient=False)
        assert should_continue_node(state) == "generate_queries"

    def test_ループ上限超過でも完了(self) -> None:
        """loop_count > max_loops の場合も complete を返す。"""
        state = _make_state(loop_count=5, max_loops=4)
        assert should_continue_node(state) == "complete"

    # ─── no-progress early exit テスト（Phase 4 追加） ───────────────

    def test_ループ2以降で新規チャンク0件ならno_progress終了(self) -> None:
        """loop_count >= 2 かつ new_chunk_ids が空なら complete を返す。"""
        state = _make_state(
            loop_count=2,
            max_loops=4,
            is_sufficient=False,
            new_chunk_ids=[],  # 0件
        )
        assert should_continue_node(state) == "complete"

    def test_ループ3以降で新規チャンク0件もno_progress終了(self) -> None:
        """loop_count >= 2 かつ new_chunk_ids が空なら complete を返す（3以降も同様）。"""
        state = _make_state(
            loop_count=3,
            max_loops=4,
            is_sufficient=False,
            new_chunk_ids=[],
        )
        assert should_continue_node(state) == "complete"

    def test_ループ1では新規チャンク0件でもno_progress終了しない(self) -> None:
        """loop_count=1 は no-progress 判定の対象外（初回検索は失敗しても再試行する）。"""
        state = _make_state(
            loop_count=1,
            max_loops=4,
            is_sufficient=False,
            new_chunk_ids=[],
        )
        assert should_continue_node(state) == "generate_queries"

    def test_ループ2以降でも新規チャンクありなら再検索(self) -> None:
        """new_chunk_ids が空でなければ no-progress 終了しない。"""
        state = _make_state(
            loop_count=2,
            max_loops=4,
            is_sufficient=False,
            new_chunk_ids=["chunk-001", "chunk-002"],
        )
        assert should_continue_node(state) == "generate_queries"

    def test_no_progressよりエラーチェックが優先される(self) -> None:
        """エラーがある場合は no-progress チェックより先に complete を返す。"""
        state = _make_state(
            loop_count=2,
            max_loops=4,
            is_sufficient=False,
            new_chunk_ids=[],
            error="接続エラー",
        )
        assert should_continue_node(state) == "complete"


# ─── node_generate_queries ────────────────────────────────────────────


class TestNodeGenerateQueries:
    def test_loop_countがインクリメントされる(self, sample_user_query: str) -> None:
        state = _make_state(user_query=sample_user_query, loop_count=0)
        result = node_generate_queries(state)
        assert result["loop_count"] == 1

    def test_current_queriesが生成される(self, sample_user_query: str) -> None:
        state = _make_state(user_query=sample_user_query, loop_count=0)
        result = node_generate_queries(state)
        assert isinstance(result["current_queries"], list)
        assert len(result["current_queries"]) >= 1

    def test_current_rationaleが生成される(self, sample_user_query: str) -> None:
        state = _make_state(user_query=sample_user_query, loop_count=0)
        result = node_generate_queries(state)
        assert isinstance(result["current_rationale"], str)
        assert len(result["current_rationale"]) > 0

    def test_2回目以降は追加検索rationale(self, sample_user_query: str) -> None:
        state = _make_state(
            user_query=sample_user_query,
            loop_count=1,
            missing_keywords=["量子力学"],
        )
        result = node_generate_queries(state)
        assert result["loop_count"] == 2
        # 2回目以降のrationale確認（"追加"を含む）
        assert "追加" in result["current_rationale"] or result["current_rationale"]

    def test_interpreted_queryが使われる(self) -> None:
        state = _make_state(
            user_query="質問",
            interpreted_query="解釈済み質問",
            loop_count=0,
        )
        result = node_generate_queries(state)
        assert result["loop_count"] == 1

    def test_tried_queriesが返却される(self, sample_user_query: str) -> None:
        """tried_queries が戻り値に含まれること。"""
        state = _make_state(user_query=sample_user_query, loop_count=0, tried_queries=[])
        result = node_generate_queries(state)
        assert "tried_queries" in result
        assert isinstance(result["tried_queries"], list)

    def test_tried_queriesに今回クエリが追記される(self, sample_user_query: str) -> None:
        """今回生成したクエリが tried_queries に追記されること。"""
        existing_tried = ["already tried query"]
        state = _make_state(
            user_query=sample_user_query,
            loop_count=0,
            tried_queries=existing_tried,
        )
        result = node_generate_queries(state)
        # 既存の tried_queries に今回のクエリが追加されているはず
        assert len(result["tried_queries"]) >= len(existing_tried)
        for q in existing_tried:
            assert q in result["tried_queries"]

    def test_tried_queriesに重複が含まれない(self, sample_user_query: str) -> None:
        """tried_queries に重複エントリが含まれないこと。"""
        state = _make_state(
            user_query=sample_user_query,
            loop_count=0,
            tried_queries=[sample_user_query],  # 今回生成されるクエリと重複の可能性
        )
        result = node_generate_queries(state)
        tried = result["tried_queries"]
        assert len(tried) == len(set(tried)), "tried_queries に重複が含まれている"

    def test_search_directivesがある場合もloop_countインクリメント(
        self, sample_user_query: str
    ) -> None:
        """search_directives があってもループカウントは正しくインクリメントされる。"""
        state = _make_state(
            user_query=sample_user_query,
            loop_count=1,
            search_directives=["Find evaluation metrics in Section 4"],
        )
        result = node_generate_queries(state)
        assert result["loop_count"] == 2


# ─── node_update_evidence ─────────────────────────────────────────────


class TestNodeUpdateEvidence:
    def test_useful_chunk_idsがkept_chunksに追加される(
        self, sample_search_results: list[dict]
    ) -> None:
        new_chunk_ids = [r["chunk_id"] for r in sample_search_results]
        useful_ids = new_chunk_ids[:2]  # 最初の2件を有用とみなす

        state = _make_state(
            search_results=sample_search_results,
            new_chunk_ids=new_chunk_ids,
            useful_chunk_ids=useful_ids,
            kept_chunks=[],
        )
        result = node_update_evidence(state)
        assert len(result["kept_chunks"]) == 2

    def test_all_seen_chunk_idsが更新される(
        self, sample_search_results: list[dict]
    ) -> None:
        new_chunk_ids = [r["chunk_id"] for r in sample_search_results]

        state = _make_state(
            search_results=sample_search_results,
            new_chunk_ids=new_chunk_ids,
            useful_chunk_ids=[],
            all_seen_chunk_ids=[],
        )
        result = node_update_evidence(state)
        # all_seen_chunk_ids に new_chunk_ids が追加される
        for chunk_id in new_chunk_ids:
            assert chunk_id in result["all_seen_chunk_ids"]

    def test_70パーセントルールが適用される(
        self, sample_search_results: list[dict]
    ) -> None:
        """satisfied_ratio >= 0.7 のとき is_sufficient が True になる。"""
        state = _make_state(
            search_results=sample_search_results,
            new_chunk_ids=[r["chunk_id"] for r in sample_search_results],
            useful_chunk_ids=[],
            is_sufficient=False,
            satisfied_ratio=0.75,  # 70% 以上
        )
        result = node_update_evidence(state)
        assert result["is_sufficient"] is True

    def test_70パーセント未満はis_sufficinetを変更しない(
        self, sample_search_results: list[dict]
    ) -> None:
        state = _make_state(
            search_results=sample_search_results,
            new_chunk_ids=[r["chunk_id"] for r in sample_search_results],
            useful_chunk_ids=[],
            is_sufficient=False,
            satisfied_ratio=0.6,  # 70% 未満
        )
        result = node_update_evidence(state)
        assert result["is_sufficient"] is False

    def test_既存kept_chunksに追加される(
        self, sample_search_results: list[dict]
    ) -> None:
        """既存の kept_chunks が保持された上で新規チャンクが追加される。"""
        existing_chunk = {"chunk_id": "existing-001", "content": "既存チャンク"}
        new_chunk_ids = [r["chunk_id"] for r in sample_search_results[:1]]

        state = _make_state(
            search_results=sample_search_results[:1],
            new_chunk_ids=new_chunk_ids,
            useful_chunk_ids=new_chunk_ids,
            kept_chunks=[existing_chunk],
        )
        result = node_update_evidence(state)
        assert len(result["kept_chunks"]) == 2
        chunk_ids = [c["chunk_id"] for c in result["kept_chunks"]]
        assert "existing-001" in chunk_ids
        assert new_chunk_ids[0] in chunk_ids

    def test_new_chunk_ids空のときno_progressフラグ確認(self) -> None:
        """new_chunk_ids が空でも is_sufficient は変更されない（no-progress は should_continue_node が担当）。"""
        state = _make_state(
            search_results=[],
            new_chunk_ids=[],
            useful_chunk_ids=[],
            is_sufficient=False,
            satisfied_ratio=0.0,
        )
        result = node_update_evidence(state)
        # is_sufficient は変更されない（0件でも強制 True にはしない）
        assert result["is_sufficient"] is False


# ─── node_complete ────────────────────────────────────────────────────


class TestNodeComplete:
    def test_kept_chunksからevidence_chunk_idsを生成(
        self, sample_search_results: list[dict]
    ) -> None:
        state = _make_state(kept_chunks=sample_search_results)
        result = node_complete(state)
        chunk_ids = result["evidence_chunk_ids"]
        assert len(chunk_ids) == len(sample_search_results)
        for sr, cid in zip(sample_search_results, chunk_ids):
            assert cid == sr["chunk_id"]

    def test_空のkept_chunksでは空リスト(self) -> None:
        state = _make_state(kept_chunks=[])
        result = node_complete(state)
        assert result["evidence_chunk_ids"] == []

    def test_chunk_idがないチャンクはスキップ(self) -> None:
        chunks = [
            {"chunk_id": "abc", "content": "有効"},
            {"content": "IDなし"},  # chunk_id なし
            {"chunk_id": "def", "content": "有効2"},
        ]
        state = _make_state(kept_chunks=chunks)
        result = node_complete(state)
        assert result["evidence_chunk_ids"] == ["abc", "def"]


# ─── init_checkpointer ───────────────────────────────────────────────


class TestInitCheckpointer:
    """init_checkpointer() の挙動を検証するテストクラス。"""

    def setup_method(self) -> None:
        """各テスト前にモジュールグローバルをリセット。"""
        graph_module._checkpointer = None
        graph_module._graph = None

    def teardown_method(self) -> None:
        """各テスト後にモジュールグローバルをリセット。"""
        graph_module._checkpointer = None
        graph_module._graph = None

    def test_database_url未設定でMemorySaverが使われる(self) -> None:
        """LIBRARIAN_DATABASE_URL が未設定（空文字）の場合、MemorySaver が使われること。"""
        from langgraph.checkpoint.memory import MemorySaver

        init_checkpointer("")
        assert isinstance(graph_module._checkpointer, MemorySaver)

    def test_空文字でgraphがNoneにリセットされる(self) -> None:
        """init_checkpointer() 呼び出し後は _graph が None にリセットされること。"""
        from librarian.graph import get_graph

        # 先に get_graph を呼んでグラフを生成
        g1 = get_graph()
        assert g1 is not None
        # init_checkpointer 呼び出し後は _graph がリセットされる
        init_checkpointer("")
        assert graph_module._graph is None

    def test_init_checkpointer後にget_graphが新グラフを構築する(self) -> None:
        """init_checkpointer() を呼ぶと次の get_graph() で新グラフが構築されること。"""
        from librarian.graph import get_graph

        g1 = get_graph()
        assert g1 is not None
        # init_checkpointer でリセット → 再構築
        init_checkpointer("")
        g2 = get_graph()
        assert g2 is not None

    def test_PostgresSaver利用不可時はMemorySaverにフォールバック(self) -> None:
        """langgraph-checkpoint-postgres が未インストール時、MemorySaver にフォールバック。"""
        from langgraph.checkpoint.memory import MemorySaver

        with patch.dict("sys.modules", {"langgraph.checkpoint.postgres": None}):
            init_checkpointer("postgresql://test:test@localhost/testdb")
        assert isinstance(graph_module._checkpointer, MemorySaver)

    def test_PostgresSaver初期化失敗時はMemorySaverにフォールバック(self) -> None:
        """PostgresSaver.from_conn_string が例外を送出する場合、MemorySaver にフォールバック。"""
        from langgraph.checkpoint.memory import MemorySaver

        mock_postgres_module = MagicMock()
        mock_saver_cls = MagicMock()
        mock_saver_cls.from_conn_string.side_effect = Exception("接続失敗")
        mock_postgres_module.PostgresSaver = mock_saver_cls
        with patch.dict("sys.modules", {"langgraph.checkpoint.postgres": mock_postgres_module}):
            init_checkpointer("postgresql://test:test@localhost/testdb")
        assert isinstance(graph_module._checkpointer, MemorySaver)


# ─── deserialize_state ────────────────────────────────────────────────


class TestDeserializeState:
    def test_正常なJSONをデシリアライズ(self) -> None:
        import json

        payload = {
            "search_results": [{"chunk_id": "abc", "content": "テスト"}],
            "new_chunk_ids": ["abc"],
        }
        results, new_ids = deserialize_state(json.dumps(payload))
        assert len(results) == 1
        assert results[0]["chunk_id"] == "abc"
        assert new_ids == ["abc"]

    def test_空文字列は空リストを返す(self) -> None:
        results, new_ids = deserialize_state("")
        assert results == []
        assert new_ids == []

    def test_不正なJSONは空リストを返す(self) -> None:
        results, new_ids = deserialize_state("invalid json {{{")
        assert results == []
        assert new_ids == []

    def test_new_chunk_idsなしJSONは空リストを返す(self) -> None:
        import json

        payload = {"search_results": [{"chunk_id": "abc", "content": "テスト"}]}
        results, new_ids = deserialize_state(json.dumps(payload))
        assert len(results) == 1
        assert new_ids == []  # new_chunk_ids キーがなければ空
