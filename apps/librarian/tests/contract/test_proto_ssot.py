"""
contracts/proto の SSOT 整合性テスト

Python サービスが依存する proto ファイルが contracts/ に存在し、
期待されるサービス定義・メッセージを含むことを確認する。
Go 版の tests/contract/ssot_test.go に相当する。
"""

from __future__ import annotations

import os
from pathlib import Path


def repo_root() -> Path:
    """リポジトリルートを解決する。"""
    current = Path(__file__).resolve()
    for parent in current.parents:
        if (parent / "contracts").is_dir():
            return parent
    raise RuntimeError(f"Repository root not found from {__file__}")


class TestProtoSsot:
    """contracts/proto の存在と内容を検証する。"""

    def test_proto_ssotファイルが存在する(self) -> None:
        proto_path = repo_root() / "contracts" / "proto" / "librarian" / "v1" / "librarian.proto"
        assert proto_path.exists(), f"contracts/proto SSOT が見つかりません: {proto_path}"

    def test_LibrarianServiceが定義されている(self) -> None:
        proto_path = repo_root() / "contracts" / "proto" / "librarian" / "v1" / "librarian.proto"
        content = proto_path.read_text()
        assert "LibrarianService" in content, "LibrarianService 定義が contracts/proto に存在しません"

    def test_Think_RPCが定義されている(self) -> None:
        proto_path = repo_root() / "contracts" / "proto" / "librarian" / "v1" / "librarian.proto"
        content = proto_path.read_text()
        assert "rpc Think" in content, "Think RPC が contracts/proto に存在しません"

    def test_ThinkRequestメッセージが定義されている(self) -> None:
        proto_path = repo_root() / "contracts" / "proto" / "librarian" / "v1" / "librarian.proto"
        content = proto_path.read_text()
        assert "ThinkRequest" in content, "ThinkRequest メッセージが contracts/proto に存在しません"

    def test_ThinkResponseメッセージが定義されている(self) -> None:
        proto_path = repo_root() / "contracts" / "proto" / "librarian" / "v1" / "librarian.proto"
        content = proto_path.read_text()
        assert "ThinkResponse" in content, "ThinkResponse メッセージが contracts/proto に存在しません"

    def test_SearchActionが定義されている(self) -> None:
        proto_path = repo_root() / "contracts" / "proto" / "librarian" / "v1" / "librarian.proto"
        content = proto_path.read_text()
        assert "SearchAction" in content, "SearchAction が contracts/proto に存在しません"

    def test_CompleteActionが定義されている(self) -> None:
        proto_path = repo_root() / "contracts" / "proto" / "librarian" / "v1" / "librarian.proto"
        content = proto_path.read_text()
        assert "CompleteAction" in content, "CompleteAction が contracts/proto に存在しません"

    def test_ErrorActionが定義されている(self) -> None:
        proto_path = repo_root() / "contracts" / "proto" / "librarian" / "v1" / "librarian.proto"
        content = proto_path.read_text()
        assert "ErrorAction" in content, "ErrorAction が contracts/proto に存在しません"

    def test_双方向ストリーミングRPCである(self) -> None:
        proto_path = repo_root() / "contracts" / "proto" / "librarian" / "v1" / "librarian.proto"
        content = proto_path.read_text()
        # gRPC 双方向ストリーミングは "stream" が両側に必要
        assert "stream ThinkRequest" in content, "ThinkRequest はストリームである必要があります"
        assert "stream ThinkResponse" in content, "ThinkResponse はストリームである必要があります"

    def test_buf_yamlが存在する(self) -> None:
        buf_path = repo_root() / "contracts" / "proto" / "buf.yaml"
        assert buf_path.exists(), f"contracts/proto/buf.yaml が見つかりません: {buf_path}"


class TestGenProtoSsot:
    """gen/proto/python/ の state（make proto 後を想定）を検証する。"""

    def test_生成済みstubsが存在する(self) -> None:
        """
        gen/proto/python/librarian/v1/librarian_pb2.py が存在することを確認。
        存在しない場合は make proto を実行するよう示す。
        """
        # __file__ は apps/librarian/tests/contract/test_proto_ssot.py
        # parents[2] で apps/librarian/ に到達する
        gen_dir = Path(__file__).resolve().parents[2] / "gen" / "proto" / "python" / "librarian" / "v1"
        pb2_path = gen_dir / "librarian_pb2.py"
        assert pb2_path.exists(), (
            f"Proto stubs が見つかりません: {pb2_path}\n"
            "  → `make proto` を実行して生成してください"
        )

    def test_生成済みgrpc_stubsが存在する(self) -> None:
        gen_dir = Path(__file__).resolve().parents[2] / "gen" / "proto" / "python" / "librarian" / "v1"
        grpc_path = gen_dir / "librarian_pb2_grpc.py"
        assert grpc_path.exists(), (
            f"gRPC stubs が見つかりません: {grpc_path}\n"
            "  → `make proto` を実行して生成してください"
        )
