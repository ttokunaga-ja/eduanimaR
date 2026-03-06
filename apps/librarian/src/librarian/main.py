"""
Librarian gRPC サーバーエントリーポイント

起動方法:
  # 開発環境（venv セットアップ後）
  make run

  # または直接実行
  PYTHONPATH=src python -m librarian.main

Docker:
  docker build -t eduanima-librarian .
  docker run -p 50051:50051 --env-file .env eduanima-librarian
"""

from __future__ import annotations

import logging
import signal
import sys
from concurrent import futures

import grpc

from librarian.config import load as load_config
from librarian.server import create_servicer

logger = logging.getLogger(__name__)


def setup_logging(log_level: str) -> None:
    """構造化ロギングのセットアップ。"""
    level = getattr(logging, log_level.upper(), logging.INFO)
    logging.basicConfig(
        level=level,
        format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
        stream=sys.stdout,
    )


def serve() -> None:
    """gRPC サーバーを起動してシグナルを待機する。"""
    cfg = load_config()
    setup_logging(cfg.log_level)

    # proto stubs を遅延インポート（make proto 後に利用可能）
    try:
        from librarian.v1 import librarian_pb2_grpc  # type: ignore[import]
    except ImportError as e:
        logger.error(
            "proto stubs が見つかりません。`make proto` を実行してください: %s",
            e,
        )
        sys.exit(1)

    # gRPC サーバー構築
    # スレッドプールサイズ: 同時ストリーミングセッション数の上限
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10),
        options=[
            # メッセージサイズ上限: 64MB（大きい検索結果対応）
            ("grpc.max_receive_message_length", 64 * 1024 * 1024),
            ("grpc.max_send_message_length", 64 * 1024 * 1024),
            # キープアライブ設定
            ("grpc.keepalive_time_ms", 10_000),
            ("grpc.keepalive_timeout_ms", 5_000),
        ],
    )

    # Servicer を登録
    servicer = create_servicer(cfg)
    librarian_pb2_grpc.add_LibrarianServiceServicer_to_server(servicer, server)

    # gRPC Reflection を有効化（grpcurl 等のデバッグツール用）
    try:
        from grpc_reflection.v1alpha import reflection  # type: ignore[import]
        from librarian.v1 import librarian_pb2  # type: ignore[import]

        service_names = (
            librarian_pb2.DESCRIPTOR.services_by_name["LibrarianService"].full_name,
            reflection.SERVICE_NAME,
        )
        reflection.enable_server_reflection(service_names, server)
        logger.info("gRPC Reflection を有効化しました")
    except ImportError:
        logger.debug("grpcio-reflection が利用できません。Reflection は無効です。")

    # アドレスをバインド
    listen_addr = f"[::]:{cfg.port}"
    server.add_insecure_port(listen_addr)

    logger.info("Librarian gRPC サーバーを起動します: %s", listen_addr)
    server.start()

    # ─── シグナルハンドリング ────────────────────────────────────────
    def _graceful_shutdown(signum: int, frame: object) -> None:
        logger.info("シャットダウンシグナルを受信しました (signum=%d)。グレースフルシャットダウン開始...", signum)
        # 5秒以内に既存ストリームを完了させてから停止
        stopped = server.stop(grace=5)
        stopped.wait()
        logger.info("サーバーを停止しました")

    signal.signal(signal.SIGTERM, _graceful_shutdown)
    signal.signal(signal.SIGINT, _graceful_shutdown)

    # メインスレッドをブロック
    logger.info("接続を待機中... (Ctrl+C で停止)")
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
