"""
共通テストフィクスチャ

使い方:
    pytest tests/
"""
from __future__ import annotations

import pytest


# ─── 共通フィクスチャ ───────────────────────────────────────────────


@pytest.fixture
def sample_request_id() -> str:
    return "test-request-00000000-0000-0000-0000-000000000001"


@pytest.fixture
def sample_user_query() -> str:
    return "量子力学のシュレーディンガー方程式について説明してください"


@pytest.fixture
def sample_subject_id() -> str:
    return "00000000-0000-0000-0000-000000000002"


@pytest.fixture
def sample_search_results() -> list[dict]:
    """Professor から受け取る典型的な検索結果リスト"""
    return [
        {
            "chunk_id": "00000000-0000-0000-0000-000000000010",
            "file_id": "00000000-0000-0000-0000-000000000003",
            "file_name": "lecture01.pdf",
            "content": "シュレーディンガー方程式は量子力学の基本方程式であり、時間発展を記述する。",
            "score": 0.95,
            "page_number": 1,
        },
        {
            "chunk_id": "00000000-0000-0000-0000-000000000011",
            "file_id": "00000000-0000-0000-0000-000000000003",
            "file_name": "lecture01.pdf",
            "content": "波動関数 ψ(x,t) は粒子の状態を確率振幅で表す。",
            "score": 0.87,
            "page_number": 2,
        },
        {
            "chunk_id": "00000000-0000-0000-0000-000000000012",
            "file_id": "00000000-0000-0000-0000-000000000003",
            "file_name": "lecture02.pdf",
            "content": "ハミルトニアン演算子はエネルギーの観測量に対応する。",
            "score": 0.81,
            "page_number": 5,
        },
    ]


@pytest.fixture
def empty_search_results() -> list[dict]:
    return []
