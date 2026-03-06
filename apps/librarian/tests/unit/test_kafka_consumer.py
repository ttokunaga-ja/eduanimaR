"""kafka_consumer.py unit tests."""

from __future__ import annotations

from typing import Any
from unittest.mock import MagicMock

from librarian.kafka_consumer import IngestEventConsumer


class _FakeMessage:
    def __init__(self, value: str, partition: int = 0, offset: int = 0) -> None:
        self.value = value
        self.partition = partition
        self.offset = offset


class _FakeConsumer:
    def __init__(self, payloads: list[str]) -> None:
        self._payloads = payloads
        self._closed = False

    def poll(self, timeout_ms: int) -> dict[str, list[_FakeMessage]]:  # noqa: ARG002
        if not self._payloads:
            return {}
        payload = self._payloads.pop(0)
        return {"topic-0": [_FakeMessage(payload)]}

    def close(self) -> None:
        self._closed = True


def test_json_eventを処理できる() -> None:
    received: list[dict[str, Any]] = []
    fake = _FakeConsumer(['{"type":"ingest_done","job_id":"job-1"}'])

    c = IngestEventConsumer(
        brokers="localhost:9094",
        topic="eduanima.ingest.jobs",
        group_id="test-group",
        on_event=received.append,
    )

    c._build_consumer = MagicMock(return_value=fake)  # type: ignore[method-assign]
    c._stop_event.set()
    c._stop_event.clear()

    # _run は stop_event が立つまで回るため、1イベント受信後に停止させる
    def _on_event(payload: dict[str, Any]) -> None:
        received.append(payload)
        c._stop_event.set()

    c._on_event = _on_event  # type: ignore[assignment]
    c._run()

    assert len(received) == 1
    assert received[0]["type"] == "ingest_done"
    assert fake._closed is True


def test_invalid_jsonは無視される() -> None:
    received: list[dict[str, Any]] = []
    fake = _FakeConsumer(["not-json"])

    c = IngestEventConsumer(
        brokers="localhost:9094",
        topic="eduanima.ingest.jobs",
        group_id="test-group",
        on_event=received.append,
    )

    c._build_consumer = MagicMock(return_value=fake)  # type: ignore[method-assign]
    c._stop_event.set()
    c._stop_event.clear()

    # 1回 poll したら停止
    original_poll = fake.poll

    def _poll_once(timeout_ms: int) -> dict[str, list[_FakeMessage]]:
        out = original_poll(timeout_ms)
        c._stop_event.set()
        return out

    fake.poll = _poll_once  # type: ignore[method-assign]
    c._run()

    assert received == []
    assert fake._closed is True
