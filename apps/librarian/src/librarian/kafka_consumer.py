"""Kafka consumer for librarian-side ingest event processing.

The consumer is optional and only starts when LIBRARIAN_KAFKA_ENABLED=true.
This keeps local quickstart lightweight while enabling event-driven operations.
"""

from __future__ import annotations

import json
import threading
import time
from collections.abc import Callable
from typing import Any

import structlog

logger = structlog.get_logger(__name__)


class IngestEventConsumer:
    """Consumes ingest-job events from Kafka in a background thread."""

    def __init__(
        self,
        *,
        brokers: str,
        topic: str,
        group_id: str,
        on_event: Callable[[dict[str, Any]], None],
    ) -> None:
        self._brokers = brokers
        self._topic = topic
        self._group_id = group_id
        self._on_event = on_event
        self._stop_event = threading.Event()
        self._thread: threading.Thread | None = None

    def start(self) -> None:
        if self._thread and self._thread.is_alive():
            return
        self._thread = threading.Thread(target=self._run, daemon=True, name="librarian-kafka-consumer")
        self._thread.start()
        logger.info(
            "librarian_kafka_consumer_started",
            brokers=self._brokers,
            topic=self._topic,
            group_id=self._group_id,
        )

    def stop(self, timeout: float = 5.0) -> None:
        self._stop_event.set()
        if self._thread and self._thread.is_alive():
            self._thread.join(timeout=timeout)
        logger.info("librarian_kafka_consumer_stopped")

    def _build_consumer(self) -> Any | None:
        try:
            from kafka import KafkaConsumer  # type: ignore[import-not-found]
        except Exception as exc:
            logger.warning(
                "kafka_python_not_available",
                error=str(exc),
                hint="Install kafka-python or disable LIBRARIAN_KAFKA_ENABLED",
            )
            return None

        return KafkaConsumer(
            self._topic,
            bootstrap_servers=[s.strip() for s in self._brokers.split(",") if s.strip()],
            group_id=self._group_id,
            auto_offset_reset="latest",
            enable_auto_commit=True,
            consumer_timeout_ms=1000,
            value_deserializer=lambda v: v.decode("utf-8") if isinstance(v, (bytes, bytearray)) else str(v),
        )

    def _run(self) -> None:
        consumer = self._build_consumer()
        if consumer is None:
            return

        try:
            while not self._stop_event.is_set():
                batches = consumer.poll(timeout_ms=1000)
                if not batches:
                    continue

                for messages in batches.values():
                    for message in messages:
                        try:
                            payload = json.loads(message.value)
                            if isinstance(payload, dict):
                                self._on_event(payload)
                                continue
                            logger.warning("kafka_event_not_object", payload_type=type(payload).__name__)
                        except json.JSONDecodeError:
                            logger.warning(
                                "kafka_event_invalid_json",
                                topic=self._topic,
                                partition=message.partition,
                                offset=message.offset,
                            )
                        except Exception as exc:
                            logger.exception("kafka_event_handler_failed", error=str(exc))
                time.sleep(0.01)
        finally:
            consumer.close()
