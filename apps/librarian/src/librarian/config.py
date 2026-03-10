"""
Librarian サービス設定
環境変数から読み込み、デフォルト値を適用する。

ThinkingLevel別モデル設定（C要件）:
  - eduanima-flash  (thinking_level="flash-lite"): LIBRARIAN_MODEL_FLASH_LITE
  - eduanima        (thinking_level="flash"):      LIBRARIAN_MODEL_FLASH
  - eduanima-pro    (thinking_level="flash"):      LIBRARIAN_MODEL_FLASH（ProもLibrarianはflash使用）
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

    # ─── ThinkingLevel別モデル設定（C要件） ─────────────────────────────
    # eduanima-flash レベル用（最速・thinking_level="flash-lite"）
    model_flash_lite: str = field(
        default_factory=lambda: os.getenv(
            "LIBRARIAN_MODEL_FLASH_LITE",
            "gemini-3.1-flash-lite-preview",
        )
    )
    # eduanima / eduanima-pro レベル用（thinking_level="flash"）
    # ※ LibrarianはProレベルでもflashモデルを使用する（設計方針）
    model_flash: str = field(
        default_factory=lambda: os.getenv(
            "LIBRARIAN_MODEL_FLASH",
            "gemini-3-flash-preview",
        )
    )

    # エージェント制約
    # max_loops: ThinkingLevelに応じて 3〜5 の範囲で可変
    #   - eduanima-flash: 3（最速）
    #   - eduanima:       4（デフォルト・バランス型）
    #   - eduanima-pro:   5（最高品質）
    # このデフォルト値は constraints.max_loops が未指定の場合のフォールバック
    max_loops: int = field(default_factory=lambda: int(os.getenv("LIBRARIAN_MAX_LOOPS", "4")))
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
