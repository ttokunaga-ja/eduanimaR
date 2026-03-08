"""
Librarian サービス設定
環境変数から読み込み、デフォルト値を適用する。
"""

from __future__ import annotations

import os
from dataclasses import dataclass, field

from dotenv import load_dotenv

load_dotenv()


@dataclass(frozen=True)
class Config:
    # gRPC サーバー設定
    port: int = field(default_factory=lambda: int(os.getenv("LIBRARIAN_PORT", "50051")))
    health_port: int = field(default_factory=lambda: int(os.getenv("LIBRARIAN_HEALTH_PORT", "8081")))

    # Gemini API
    gemini_api_key: str = field(default_factory=lambda: os.getenv("GEMINI_API_KEY", ""))
    # クエリ生成・充足度評価に使用するモデル名
    # LIBRARIAN_MODEL_SEARCH 環境変数で上書き可能（デフォルト: gemini-3-flash-preview）
    gemini_model: str = field(default_factory=lambda: os.getenv("LIBRARIAN_MODEL_SEARCH", "gemini-3-flash-preview"))

    # エージェント制約
    # max_loops: 上限 5 回（平均 3 回で収束する設計）
    max_loops: int = field(default_factory=lambda: int(os.getenv("LIBRARIAN_MAX_LOOPS", "5")))
    max_results: int = field(default_factory=lambda: int(os.getenv("LIBRARIAN_MAX_RESULTS", "10")))
    timeout_ms: int = field(default_factory=lambda: int(os.getenv("LIBRARIAN_TIMEOUT_MS", "30000")))

    # Kafka consumer (optional)
    kafka_enabled: bool = field(default_factory=lambda: os.getenv("LIBRARIAN_KAFKA_ENABLED", "false").lower() == "true")
    kafka_brokers: str = field(default_factory=lambda: os.getenv("KAFKA_BROKERS", "localhost:9094"))
    kafka_topic_ingest: str = field(default_factory=lambda: os.getenv("KAFKA_TOPIC_INGEST", "eduanima.ingest.jobs"))
    kafka_group_id: str = field(
        default_factory=lambda: os.getenv(
            "LIBRARIAN_KAFKA_GROUP_ID",
            "librarian-ingest-consumer",
        )
    )

    # ロギング
    log_level: str = field(default_factory=lambda: os.getenv("LOG_LEVEL", "INFO"))


def load() -> Config:
    """設定を読み込んで返す。"""
    cfg = Config()
    if not cfg.gemini_api_key:
        import warnings

        warnings.warn(
            "GEMINI_API_KEY が設定されていません。LLM呼び出しは失敗します。",
            stacklevel=2,
        )
    return cfg
