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

    # Gemini API
    gemini_api_key: str = field(default_factory=lambda: os.getenv("GEMINI_API_KEY", ""))

    # エージェント制約
    max_loops: int = field(default_factory=lambda: int(os.getenv("LIBRARIAN_MAX_LOOPS", "3")))
    max_results: int = field(default_factory=lambda: int(os.getenv("LIBRARIAN_MAX_RESULTS", "10")))
    timeout_ms: int = field(default_factory=lambda: int(os.getenv("LIBRARIAN_TIMEOUT_MS", "30000")))

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
