"""
config.py のユニットテスト

Config 読み込みとバリデーションの挙動を確認する。
外部依存なしで実行できる。
"""

from __future__ import annotations

import os
from unittest.mock import patch

from librarian.config import Config, load


class TestConfigDefaults:
    def test_デフォルトポートが50051(self) -> None:
        with patch.dict(os.environ, {}, clear=False):
            # LIBRARIAN_PORT 未設定の場合はデフォルト値
            cfg = Config()
            assert cfg.port == 50051

    def test_デフォルトヘルスポートが8081(self) -> None:
        cfg = Config()
        assert cfg.health_port == 8081

    def test_デフォルトmax_loopsが3(self) -> None:
        cfg = Config()
        assert cfg.max_loops == 3

    def test_デフォルトmax_resultsが10(self) -> None:
        cfg = Config()
        assert cfg.max_results == 10

    def test_デフォルトtimeout_msが30000(self) -> None:
        cfg = Config()
        assert cfg.timeout_ms == 30000

    def test_デフォルトlog_levelがINFO(self) -> None:
        # LOG_LEVEL を環境変数から除外して Config を生成する（アイソレーション）
        env = {k: v for k, v in os.environ.items() if k != "LOG_LEVEL"}
        with patch.dict(os.environ, env, clear=True):
            cfg = Config()
            assert cfg.log_level == "INFO"


class TestConfigEnvOverride:
    def test_環境変数でポートを上書きできる(self) -> None:
        with patch.dict(os.environ, {"LIBRARIAN_PORT": "9999"}):
            cfg = Config()
            assert cfg.port == 9999

    def test_環境変数でmax_loopsを上書きできる(self) -> None:
        with patch.dict(os.environ, {"LIBRARIAN_MAX_LOOPS": "5"}):
            cfg = Config()
            assert cfg.max_loops == 5

    def test_gemini_api_keyを環境変数から読む(self) -> None:
        with patch.dict(os.environ, {"GEMINI_API_KEY": "test-key-xyz"}):
            cfg = Config()
            assert cfg.gemini_api_key == "test-key-xyz"

    def test_gemini_api_key未設定の場合は空文字(self) -> None:
        with patch.dict(os.environ, {}, clear=False):
            # Config() 直接生成: 現在の env をそのまま使うので
            # 未設定かどうかは load() のほうで確認する
            env_without_key = {k: v for k, v in os.environ.items() if k != "GEMINI_API_KEY"}
            with patch.dict(os.environ, env_without_key, clear=True):
                cfg = Config()
                assert cfg.gemini_api_key == ""


class TestConfigLoad:
    def test_loadがConfigを返す(self) -> None:
        with patch.dict(os.environ, {"GEMINI_API_KEY": "test-key"}):
            cfg = load()
            assert isinstance(cfg, Config)

    def test_Configはイミュータブル(self) -> None:
        cfg = Config()
        try:
            cfg.port = 9999  # type: ignore[misc]
            assert False, "frozenである場合は TypeError が発生するはず"
        except (AttributeError, TypeError):
            pass  # frozen=True のため変更不可
