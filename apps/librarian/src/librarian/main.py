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
import threading
from concurrent import futures
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import grpc
import structlog

from librarian.config import load as load_config
from librarian.kafka_consumer import IngestEventConsumer
from librarian.server import create_servicer

logger = structlog.get_logger(__name__)


def start_health_server(port: int, ready_event: threading.Event) -> ThreadingHTTPServer:
    """HTTP /healthz /readyz エンドポイントを別スレッドで起動する。"""

    class HealthHandler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:  # noqa: N802
            if self.path == "/healthz":
                body = b'{"status":"ok","service":"librarian"}'
                self.send_response(200)
            elif self.path == "/readyz":
                status = b"ready" if ready_event.is_set() else b"starting"
                code = 200 if ready_event.is_set() else 503
                body = b'{"status":"' + status + b'","service":"librarian"}'
                self.send_response(code)
            else:
                body = b'{"error":"not_found"}'
                self.send_response(404)

            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, fmt: str, *args: object) -> None:
            return

    server = ThreadingHTTPServer(("0.0.0.0", port), HealthHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    logger.info("HTTP health server started", port=port)
    return server


def setup_logging(log_level: str) -> None:
    """構造化ロギングのセットアップ・ structlog を JSON レンダラーで設定する。"""
    level = getattr(logging, log_level.upper(), logging.INFO)
    # stdlib logging を structlog のバックエンドとして設定
    logging.basicConfig(
        format="%(message)s",
        stream=sys.stdout,
        level=level,
    )
    structlog.configure(
        processors=[
            structlog.stdlib.add_log_level,
            structlog.stdlib.add_logger_name,
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.StackInfoRenderer(),
            structlog.processors.format_exc_info,
            structlog.processors.JSONRenderer(),
        ],
        wrapper_class=structlog.make_filtering_bound_logger(level),
        logger_factory=structlog.PrintLoggerFactory(),
        cache_logger_on_first_use=True,
    )


def serve() -> None:
    """gRPC サーバーを起動してシグナルを待機する。"""
    cfg = load_config()
    setup_logging(cfg.log_level)

    kafka_consumer: IngestEventConsumer | None = None

    def on_ingest_event(payload: dict[str, object]) -> None:
        # 受信イベントは現時点ではログ記録のみに限定し、将来の
        # インデックスウォームアップや前処理フック拡張点として扱う。
        logger.info(
            "librarian_ingest_event_received",
            event_type=str(payload.get("type", "unknown")),
            job_id=str(payload.get("job_id", "")),
            file_id=str(payload.get("file_id", "")),
        )

    # 起動時必須設定のバリデーション
    if not cfg.gemini_api_key:
        logger.error("GEMINI_API_KEY is required but not set")
        sys.exit(1)

    # proto stubs を遅延インポート（make proto 後に利用可能）
    try:
        from librarian.v1 import librarian_pb2_grpc  # type: ignore[import]
    except ImportError as e:
        logger.error(
            "proto stubs が見つかりません。`make proto` を実行してください",
            error=str(e),
        )
        sys.exit(1)

    ready_event = threading.Event()
    health_server = start_health_server(cfg.health_port, ready_event)

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

    logger.info("Librarian gRPC サーバーを起動します", listen_addr=listen_addr)
    server.start()
    ready_event.set()

    if cfg.kafka_enabled:
        kafka_consumer = IngestEventConsumer(
            brokers=cfg.kafka_brokers,
            topic=cfg.kafka_topic_ingest,
            group_id=cfg.kafka_group_id,
            on_event=on_ingest_event,
        )
        kafka_consumer.start()
    else:
        logger.info("librarian_kafka_consumer_disabled")

    # ─── シグナルハンドリング ────────────────────────────────────────
    def _graceful_shutdown(signum: int, frame: object) -> None:
        logger.info("シャットダウンシグナルを受信。グレースフルシャットダウン開始", signum=signum)
        ready_event.clear()
        if kafka_consumer is not None:
            kafka_consumer.stop()
        health_server.shutdown()
        health_server.server_close()
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
