"""
server.py のユニットテスト（Phase 2: LangGraph Cyclic Graph 対応）

LibrarianServicer.Think() のメッセージ処理ロジックを検証する。
gRPC コンテキスト・proto stubs・LangGraph グラフをモックして外部依存なしで実行。

Phase 2 変更点:
  - get_graph() をモック化して LangGraph への依存を排除
  - graph.stream() / graph.get_state() のモックで interrupt ループを再現
  - initial_state に completion_criteria / thinking_level / interpreted_query が含まれること確認
  - SearchAction → CompleteAction / ErrorAction の正しいシーケンスを検証
"""

from __future__ import annotations

import json
from typing import Any
from unittest.mock import MagicMock, patch

from librarian.config import Config
from librarian.server import LibrarianServicer, create_servicer


# ─── ヘルパー関数 ──────────────────────────────────────────────────────


def _make_config(**overrides: Any) -> Config:
    """テスト用 Config を生成する。"""
    defaults: dict[str, Any] = {
        "gemini_api_key": "test-key",
        "max_loops": 2,
        "max_results": 5,
    }
    defaults.update(overrides)
    import os
    from unittest.mock import patch as _patch

    env = {
        "GEMINI_API_KEY": str(defaults["gemini_api_key"]),
        "LIBRARIAN_MAX_LOOPS": str(defaults["max_loops"]),
        "LIBRARIAN_MAX_RESULTS": str(defaults["max_results"]),
    }
    with _patch.dict(os.environ, env):
        return Config()


def _make_proto_mock() -> tuple[MagicMock, MagicMock]:
    """librarian_pb2 / librarian_pb2_grpc の最小モックを返す。"""
    pb2 = MagicMock()
    pb2_grpc = MagicMock()

    def make_response(**kwargs: Any) -> MagicMock:
        m = MagicMock()
        for k, v in kwargs.items():
            setattr(m, k, v)
        return m

    pb2.SearchAction = MagicMock(side_effect=make_response)
    pb2.CompleteAction = MagicMock(side_effect=make_response)
    pb2.ErrorAction = MagicMock(side_effect=make_response)
    pb2.ThinkResponse = MagicMock(side_effect=make_response)
    pb2.Evidence = MagicMock(side_effect=make_response)
    return pb2, pb2_grpc


def _make_initial_request(
    request_id: str = "req-001",
    user_query: str = "量子力学とは",
    subject_id: str = "subj-001",
    max_loops: int = 2,
    thinking_level: str = "flash",
    interpreted_query: str = "",
    completion_criteria: list[str] | None = None,
) -> MagicMock:
    """初回 ThinkRequest モックを生成する。"""
    req = MagicMock()
    req.request_id = request_id
    req.user_query = user_query
    req.subject_id = subject_id
    req.constraints.max_loops = max_loops
    req.constraints.thinking_level = thinking_level
    req.constraints.interpreted_query = interpreted_query
    req.constraints.completion_criteria = completion_criteria or []
    return req


def _make_search_result_request(
    search_results: list[dict] | None = None,
    new_chunk_ids: list[str] | None = None,
) -> MagicMock:
    """検索結果を含む ThinkRequest モックを生成する。"""
    req = MagicMock()
    payload: dict[str, Any] = {"search_results": search_results or []}
    if new_chunk_ids is not None:
        payload["new_chunk_ids"] = new_chunk_ids
    req.state = json.dumps(payload)
    return req


def _make_graph_mock(
    interrupt_payload: dict | None = None,
    final_kept_chunks: list[dict] | None = None,
    loop_count: int = 1,
    satisfied_ratio: float = 0.9,
) -> MagicMock:
    """
    LangGraph グラフのモックを生成する。

    再現するフロー（1 ループ完了のケース）:
      1. graph.stream(initial_state, ...) → []
         （グラフが interrupt まで実行される）
      2. graph.get_state(config) → interrupt_state
         （state_snapshot.next = ["wait_for_search"]）
      3. graph.stream(Command(resume=...), ...) → []
         （グラフが END まで実行される）
      4. graph.get_state(config) → complete_state
         （state_snapshot.next = []）
    """
    mock_graph = MagicMock()
    # stream はイテレータとして消費されるため list を返す（何度でも空を返す）
    mock_graph.stream.return_value = []

    # ── interrupt 状態（1回目の get_state()）──────────────────────────
    interrupt_state = MagicMock()
    interrupt_state.next = ["wait_for_search"]   # グラフが interrupt で停止中
    interrupt_state.values = {"loop_count": 1}

    payload = interrupt_payload or {
        "queries": ["量子力学 基礎", "量子力学 定義"],
        "rationale": "量子力学に関する資料を検索しています...",
        "exclude_chunk_ids": [],
    }
    interrupt_item = MagicMock()
    interrupt_item.value = payload
    task_mock = MagicMock()
    task_mock.interrupts = [interrupt_item]
    interrupt_state.tasks = [task_mock]

    # ── 完了状態（2回目の get_state()）───────────────────────────────
    complete_state = MagicMock()
    complete_state.next = []   # グラフが END に到達（ループ終了）
    complete_state.values = {
        "kept_chunks": final_kept_chunks if final_kept_chunks is not None else [
            {"chunk_id": "chunk-001", "content": "量子力学の説明テキスト"}
        ],
        "loop_count": loop_count,
        "satisfied_ratio": satisfied_ratio,
    }

    mock_graph.get_state.side_effect = [interrupt_state, complete_state]
    return mock_graph


# ─── テストクラス ─────────────────────────────────────────────────────


class TestLibrarianServicerCreate:
    """create_servicer() / __init__() の基本動作を検証する。"""

    def test_create_servicerがLibrarianServicerを返す(self) -> None:
        cfg = _make_config()
        svc = create_servicer(cfg)
        assert isinstance(svc, LibrarianServicer)

    def test_LibrarianServicerがConfigを保持する(self) -> None:
        cfg = _make_config(max_loops=3)
        svc = LibrarianServicer(cfg)
        assert svc._cfg.max_loops == 3


class TestLibrarianServicerThink:
    """Think() の基本フローを検証する。"""

    def _run_think(
        self,
        servicer: LibrarianServicer,
        requests: list[MagicMock],
        pb2: MagicMock,
        mock_graph: MagicMock | None = None,
    ) -> list[MagicMock]:
        """
        Think() の出力を全収集して返す。

        - librarian.v1 は常にモック化（proto stubs 未生成環境でも動作）
        - mock_graph が渡された場合は librarian.server.get_graph をモック化
        """
        ctx = MagicMock()
        patches = {
            "librarian.v1": MagicMock(librarian_pb2=pb2),
        }
        with patch.dict("sys.modules", patches):
            if mock_graph is not None:
                with patch("librarian.server.get_graph", return_value=mock_graph):
                    responses = list(servicer.Think(iter(requests), ctx))
            else:
                responses = list(servicer.Think(iter(requests), ctx))
        return responses

    # ── SearchAction 送信テスト ────────────────────────────────────────

    def test_初回リクエストでSearchActionが送信される(self) -> None:
        """1回の interrupt ループで SearchAction が正しく送信されること。"""
        pb2, _ = _make_proto_mock()
        cfg = _make_config(max_loops=2)
        servicer = LibrarianServicer(cfg)

        req1 = _make_initial_request()
        req2 = _make_search_result_request(
            search_results=[{"chunk_id": "chunk-001", "content": "量子力学の説明", "score": 0.9}],
            new_chunk_ids=["chunk-001"],
        )

        mock_graph = _make_graph_mock()

        responses = self._run_think(servicer, [req1, req2], pb2, mock_graph)

        # 少なくとも SearchAction + CompleteAction の 2 レスポンス
        assert len(responses) >= 1
        # ThinkResponse が 2 回以上呼ばれていること（SearchAction + CompleteAction）
        assert pb2.ThinkResponse.call_count >= 2
        # SearchAction が 1 回呼ばれていること
        pb2.SearchAction.assert_called_once()

    def test_SearchActionにinterruptのpayloadが反映される(self) -> None:
        """interrupt payload の queries が SearchAction に正しく渡されること。"""
        pb2, _ = _make_proto_mock()
        cfg = _make_config(max_loops=2)
        servicer = LibrarianServicer(cfg)

        expected_queries = ["量子もつれ 解説", "量子力学 歴史"]
        req1 = _make_initial_request()
        req2 = _make_search_result_request(
            search_results=[{"chunk_id": "chunk-001", "content": "量子力学", "score": 0.9}],
            new_chunk_ids=["chunk-001"],
        )

        mock_graph = _make_graph_mock(
            interrupt_payload={
                "queries": expected_queries,
                "rationale": "量子力学について調査中",
                "exclude_chunk_ids": ["old-chunk-001"],
            }
        )

        self._run_think(servicer, [req1, req2], pb2, mock_graph)

        # SearchAction に正しい queries と exclude_chunk_ids が渡されていること
        pb2.SearchAction.assert_called_once_with(
            queries_text=expected_queries,
            queries_vector=expected_queries,
            rationale="量子力学について調査中",
            exclude_chunk_ids=["old-chunk-001"],
        )

    # ── CompleteAction テスト ─────────────────────────────────────────

    def test_グラフ完了後にCompleteActionが返される(self) -> None:
        """LangGraph 完了後に CompleteAction を含む ThinkResponse が返されること。"""
        pb2, _ = _make_proto_mock()
        cfg = _make_config(max_loops=2)
        servicer = LibrarianServicer(cfg)

        req1 = _make_initial_request(
            completion_criteria=["量子力学の定義", "歴史的背景"]
        )
        req2 = _make_search_result_request(
            search_results=[{"chunk_id": "chunk-001", "content": "量子力学の説明", "score": 0.9}],
            new_chunk_ids=["chunk-001"],
        )

        mock_graph = _make_graph_mock(
            final_kept_chunks=[{"chunk_id": "chunk-001", "content": "量子力学の説明"}],
            satisfied_ratio=0.9,
        )

        responses = self._run_think(servicer, [req1, req2], pb2, mock_graph)

        # CompleteAction が呼ばれていること
        pb2.CompleteAction.assert_called_once()
        # Evidence が kept_chunks の chunk_id から生成されていること
        pb2.Evidence.assert_called_once_with(
            chunk_id="chunk-001",
            why_relevant=pb2.Evidence.call_args.kwargs["why_relevant"],
        )

    def test_CompleteActionのevidence_countが正しい(self) -> None:
        """複数の kept_chunks から Evidence が正しい数だけ生成されること。"""
        pb2, _ = _make_proto_mock()
        cfg = _make_config(max_loops=2)
        servicer = LibrarianServicer(cfg)

        req1 = _make_initial_request()
        req2 = _make_search_result_request(
            search_results=[
                {"chunk_id": f"chunk-{i:03d}", "content": f"内容{i}", "score": 0.8}
                for i in range(3)
            ],
            new_chunk_ids=[f"chunk-{i:03d}" for i in range(3)],
        )

        mock_graph = _make_graph_mock(
            final_kept_chunks=[
                {"chunk_id": "chunk-000", "content": "内容0"},
                {"chunk_id": "chunk-001", "content": "内容1"},
                {"chunk_id": "chunk-002", "content": "内容2"},
            ],
            satisfied_ratio=1.0,
        )

        self._run_think(servicer, [req1, req2], pb2, mock_graph)

        # Evidence が 3 件生成されていること
        assert pb2.Evidence.call_count == 3

    # ── ErrorAction テスト ────────────────────────────────────────────

    def test_エビデンス空のときErrorActionが返される(self) -> None:
        """kept_chunks が空の場合は LOOP_LIMIT ErrorAction が返されること。"""
        pb2, _ = _make_proto_mock()
        cfg = _make_config(max_loops=2)
        servicer = LibrarianServicer(cfg)

        req1 = _make_initial_request()
        req2 = _make_search_result_request(search_results=[])

        mock_graph = _make_graph_mock(
            final_kept_chunks=[],   # エビデンスなし
            satisfied_ratio=0.0,
        )

        responses = self._run_think(servicer, [req1, req2], pb2, mock_graph)

        # ErrorAction が呼ばれていること
        pb2.ErrorAction.assert_called_once()
        # error_type が "LOOP_LIMIT" であること
        call_kwargs = pb2.ErrorAction.call_args.kwargs
        assert call_kwargs["error_type"] == "LOOP_LIMIT"
        # CompleteAction は呼ばれないこと
        pb2.CompleteAction.assert_not_called()

    # ── エラーハンドリングテスト ─────────────────────────────────────

    def test_proto_stubsがないときにabortする(self) -> None:
        """proto stubs が import できない場合、context.abort が呼ばれること。"""
        cfg = _make_config()
        servicer = LibrarianServicer(cfg)

        ctx = MagicMock()
        req = _make_initial_request(request_id="req-002")

        # librarian.v1 = None → ImportError をシミュレート
        with patch.dict("sys.modules", {"librarian.v1": None}):
            try:
                list(servicer.Think(iter([req]), ctx))
            except Exception:
                pass
            ctx.abort.assert_called_once()

    def test_get_graphがNoneのときにabortする(self) -> None:
        """get_graph() が None を返す場合、context.abort が呼ばれること。"""
        pb2, _ = _make_proto_mock()
        cfg = _make_config()
        servicer = LibrarianServicer(cfg)

        ctx = MagicMock()
        req = _make_initial_request(request_id="req-003")

        with patch.dict("sys.modules", {"librarian.v1": MagicMock(librarian_pb2=pb2)}):
            with patch("librarian.server.get_graph", return_value=None):
                try:
                    list(servicer.Think(iter([req]), ctx))
                except Exception:
                    pass
                ctx.abort.assert_called_once()

    # ── initial_state の内容検証 ─────────────────────────────────────

    def test_completion_criteriaがinitial_stateに反映される(self) -> None:
        """completion_criteria が initial_state に正しく渡されること。"""
        pb2, _ = _make_proto_mock()
        cfg = _make_config(max_loops=2)
        servicer = LibrarianServicer(cfg)

        criteria = ["研究目的 | 研究の背景", "実験方法"]
        req1 = _make_initial_request(
            completion_criteria=criteria,
            interpreted_query="この研究の目的を教えてください",
        )
        req2 = _make_search_result_request(
            search_results=[{"chunk_id": "chunk-001", "content": "研究の目的説明", "score": 0.8}],
            new_chunk_ids=["chunk-001"],
        )

        mock_graph = _make_graph_mock(
            final_kept_chunks=[{"chunk_id": "chunk-001", "content": "研究の目的説明"}],
        )

        # graph.stream が initial_state を受け取ることをキャプチャ
        captured_states: list[dict] = []

        def capture_stream(state_or_cmd, **kwargs: Any):
            if isinstance(state_or_cmd, dict):
                captured_states.append(state_or_cmd)
            return []

        mock_graph.stream.side_effect = capture_stream

        self._run_think(servicer, [req1, req2], pb2, mock_graph)

        # 初回 stream に completion_criteria が含まれていること
        assert len(captured_states) >= 1
        first_state = captured_states[0]
        assert first_state.get("completion_criteria") == criteria
        assert first_state.get("interpreted_query") == "この研究の目的を教えてください"

    def test_thinking_levelがinitial_stateに反映される(self) -> None:
        """thinking_level と max_loops が initial_state に反映されること。"""
        pb2, _ = _make_proto_mock()
        cfg = _make_config(max_loops=4)
        servicer = LibrarianServicer(cfg)

        req1 = _make_initial_request(
            thinking_level="flash-lite",
            max_loops=3,
        )
        req2 = _make_search_result_request(
            search_results=[{"chunk_id": "c1", "content": "内容", "score": 0.7}],
            new_chunk_ids=["c1"],
        )

        mock_graph = _make_graph_mock(
            final_kept_chunks=[{"chunk_id": "c1", "content": "内容"}],
        )

        captured_states: list[dict] = []

        def capture_stream(state_or_cmd, **kwargs: Any):
            if isinstance(state_or_cmd, dict):
                captured_states.append(state_or_cmd)
            return []

        mock_graph.stream.side_effect = capture_stream

        self._run_think(servicer, [req1, req2], pb2, mock_graph)

        assert len(captured_states) >= 1
        assert captured_states[0].get("thinking_level") == "flash-lite"
        assert captured_states[0].get("max_loops") == 3

    def test_request_idがinitial_stateに含まれる(self) -> None:
        """request_id / user_query / subject_id が initial_state に含まれること。"""
        pb2, _ = _make_proto_mock()
        cfg = _make_config()
        servicer = LibrarianServicer(cfg)

        req1 = _make_initial_request(
            request_id="req-unique-001",
            user_query="機械学習とは何か",
            subject_id="subj-ml-101",
        )
        req2 = _make_search_result_request(
            search_results=[{"chunk_id": "ml-001", "content": "機械学習の定義", "score": 0.95}],
            new_chunk_ids=["ml-001"],
        )

        mock_graph = _make_graph_mock(
            final_kept_chunks=[{"chunk_id": "ml-001", "content": "機械学習の定義"}],
        )

        captured_states: list[dict] = []

        def capture_stream(state_or_cmd, **kwargs: Any):
            if isinstance(state_or_cmd, dict):
                captured_states.append(state_or_cmd)
            return []

        mock_graph.stream.side_effect = capture_stream

        self._run_think(servicer, [req1, req2], pb2, mock_graph)

        assert len(captured_states) >= 1
        state = captured_states[0]
        assert state.get("request_id") == "req-unique-001"
        assert state.get("user_query") == "機械学習とは何か"
        assert state.get("subject_id") == "subj-ml-101"

    # ── graph.stream / get_state 呼び出し回数の検証 ─────────────────

    def test_graphのstream呼び出し回数が正しい(self) -> None:
        """
        1 ループ完了時の graph.stream 呼び出し回数を検証する。

        期待:
          1. graph.stream(initial_state, ...) → interrupt まで
          2. graph.stream(Command(resume=...), ...) → END まで
        計: 2 回
        """
        pb2, _ = _make_proto_mock()
        cfg = _make_config(max_loops=2)
        servicer = LibrarianServicer(cfg)

        req1 = _make_initial_request()
        req2 = _make_search_result_request(
            search_results=[{"chunk_id": "chunk-001", "content": "内容", "score": 0.9}],
            new_chunk_ids=["chunk-001"],
        )

        mock_graph = _make_graph_mock()

        self._run_think(servicer, [req1, req2], pb2, mock_graph)

        # stream が 2 回呼ばれていること
        assert mock_graph.stream.call_count == 2

    def test_graphのget_state呼び出し回数が正しい(self) -> None:
        """
        1 ループ完了時の graph.get_state 呼び出し回数を検証する。

        期待:
          1. interrupt 後の get_state（停止中確認）
          2. resume 後の get_state（終了確認）
        計: 2 回
        """
        pb2, _ = _make_proto_mock()
        cfg = _make_config(max_loops=2)
        servicer = LibrarianServicer(cfg)

        req1 = _make_initial_request()
        req2 = _make_search_result_request(
            search_results=[{"chunk_id": "chunk-001", "content": "内容", "score": 0.9}],
            new_chunk_ids=["chunk-001"],
        )

        mock_graph = _make_graph_mock()

        self._run_think(servicer, [req1, req2], pb2, mock_graph)

        # get_state が 2 回呼ばれていること（interrupt 確認 + 終了確認）
        assert mock_graph.get_state.call_count == 2
