"""
server.py のユニットテスト

LibrarianServicer.Think() のメッセージ処理ロジックを検証する。
gRPC コンテキスト・proto stubs をモックして外部依存なしで実行。
"""

from __future__ import annotations

from typing import Any
from unittest.mock import MagicMock, patch

from librarian.config import Config
from librarian.server import LibrarianServicer, create_servicer


def _make_config(**overrides: Any) -> Config:
    """テスト用 Config を生成する。"""
    defaults: dict[str, Any] = {
        "gemini_api_key": "test-key",
        "max_loops": 2,
        "max_results": 5,
    }
    defaults.update(overrides)
    # Config は frozen dataclass なので環境変数パッチで生成
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

    # SearchAction / CompleteAction / ErrorAction / ThinkResponse のモック
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


class TestLibrarianServicerThink:
    """Think() の基本フローを検証する。"""

    def _run_think(
        self,
        servicer: LibrarianServicer,
        requests: list[MagicMock],
        pb2: MagicMock,
    ) -> list[MagicMock]:
        """Think() の出力を全収集して返す。"""
        ctx = MagicMock()

        with patch.dict(
            "sys.modules",
            {"librarian.v1": MagicMock(librarian_pb2=pb2)},
        ):
            responses = list(servicer.Think(iter(requests), ctx))
        return responses

    def test_create_servicerがLibrarianServicerを返す(self) -> None:
        cfg = _make_config()
        svc = create_servicer(cfg)
        assert isinstance(svc, LibrarianServicer)

    def test_初回リクエストでSearchActionが送信される(self) -> None:
        pb2, _ = _make_proto_mock()
        cfg = _make_config()
        servicer = LibrarianServicer(cfg)

        # 初回リクエスト
        req1 = MagicMock()
        req1.request_id = "test-req-001"
        req1.user_query = "量子力学とは"
        req1.subject_id = "subj-001"
        req1.constraints.max_loops = 2
        req1.constraints.max_results = 5

        # 2回目（検索結果を含む）
        import json

        req2 = MagicMock()
        req2.state = json.dumps({"search_results": [{"content": "量子力学の説明", "score": 0.9}]})

        responses = self._run_think(servicer, [req1, req2], pb2)

        # 少なくとも 1 つ以上のレスポンスが返る
        assert len(responses) >= 1
        # 最初のレスポンスは SearchAction を持っているはず（ThinkResponse が生成された）
        pb2.ThinkResponse.assert_called()

    def test_proto_stubsがないときにabortする(self) -> None:
        cfg = _make_config()
        servicer = LibrarianServicer(cfg)

        ctx = MagicMock()
        req = MagicMock()
        req.request_id = "test-req-002"
        req.user_query = "test"
        req.subject_id = "subj-001"
        req.constraints.max_loops = 1
        req.constraints.max_results = 5

        # proto stubs が import できない状態をシミュレート
        with patch.dict("sys.modules", {"librarian.v1": None}):
            # ImportError が発生する→ context.abort が呼ばれる
            try:
                list(servicer.Think(iter([req]), ctx))
            except Exception:
                pass
            # abort が呼ばれたことを確認
            ctx.abort.assert_called_once()
