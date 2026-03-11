"""
LangGraph ベースの推論エージェント

Gemini Structured Output を使用してクエリ生成・検索結果評価・エビデンス選別を行う。

SDK: google-genai（新公式SDK）
  旧: google-generativeai（非推奨）→ 新: google-genai

Temperature 設計（gemini_use.md 準拠）:
  Temperature=1.0 MANDATORY for Gemini 3 (全関数共通)
  - Gemini 3 は Temperature=0.0 がアンチパターン（品質低下・過決定論の弊害）。
  - Seed=42 で決定論的再現性を確保する（ベストエフォート）。

SubAgent 設計（並列化）:
  - SubAgent-B (filter_useful_chunks): 新規チャンクのフィルタリングのみ実行
  - SubAgent-C (judge_sufficiency):    新規+保存済みチャンクで充足度判断
  - evaluate_parallel: ThreadPoolExecutor で B/C を並列実行（レイテンシ削減）

出力最小化設計（A-1）:
  評価結果は最小フィールドのみ。長文の reason/rationale は排除する。

蓄積型評価（A-2）:
  SubAgent-C に「今回の新規チャンク」＋「過去にKeepされた精鋭チャンク」の
  両方を渡し、累積情報量で充足度を判断することでEarly Exitを実現する。

Triaging（A-3）:
  SubAgent-B が新規チャンクの中から有用なものをIDで選択し、
  ゴミ情報がProfessorに渡らないようにする。

ループ上限: max_loops (ThinkingLevelに応じて 3〜5)
  - eduanima-flash: 3ループ（最速）
  - eduanima:       4ループ（バランス型・デフォルト）
  - eduanima-pro:   5ループ（最高品質）
"""

from __future__ import annotations

import json
from concurrent.futures import ThreadPoolExecutor
from typing import Any, TypedDict

import structlog

logger = structlog.get_logger(__name__)


# ─── Structured Output スキーマ定義 ──────────────────────────────────


class _QueryItem(TypedDict):
    """クエリ生成の Structured Output スキーマ。"""

    query: str  # 検索クエリ文字列（rationale フィールド削除: レイテンシ削減）


class _FilterResult(TypedDict):
    """
    SubAgent-B のフィルタリング結果スキーマ（A-1: 出力最小化）。

    新規チャンクの中から有用な chunk_id のみを選別する。
    """

    useful_chunk_ids: list[str]  # 今回の新規チャンクの中から有用な chunk_id のみ（A-3）


class _SufficiencyResult(TypedDict):
    """
    SubAgent-C の充足度判断スキーマ（A-1: 出力最小化）。

    新規＋保存済みチャンクの累積情報量で充足度を判断する。
    """

    is_sufficient: bool          # True: 充足（CompleteAction へ）、False: 再検索
    missing_keywords: list[str]  # 不足情報の短いキーワード（次のクエリ生成に使用）


# ─── Gemini クライアント（モジュールレベルシングルトン） ─────────────

_gemini_client = None                    # google.genai.Client シングルトン
_gemini_model_flash_lite: str | None = None  # flash-lite モデル名（eduanima-flash 用）
_gemini_model_flash: str | None = None   # flash モデル名（eduanima / eduanima-pro 用）


def init_gemini(api_key: str, model_flash_lite: str, model_flash: str) -> None:
    """
    Gemini クライアントを初期化する（起動時に一度だけ呼ぶ）。

    ThinkingLevel に応じて使用するモデルを動的に選択できるよう、
    2種類のモデル名を事前に保持する（C要件）。

    Args:
        api_key: GEMINI_API_KEY
        model_flash_lite: eduanima-flash レベル用モデル（最速）
        model_flash: eduanima / eduanima-pro レベル用モデル（バランス型）
    """
    global _gemini_client, _gemini_model_flash_lite, _gemini_model_flash
    try:
        from google import genai

        _gemini_client = genai.Client(api_key=api_key)
        _gemini_model_flash_lite = model_flash_lite
        _gemini_model_flash = model_flash
        logger.info(
            "Gemini クライアント初期化完了",
            model_flash_lite=model_flash_lite,
            model_flash=model_flash,
        )
    except Exception as exc:
        logger.error("Gemini クライアント初期化失敗", error=str(exc))
        _gemini_client = None
        _gemini_model_flash_lite = None
        _gemini_model_flash = None


def get_model_for_level(thinking_level: str) -> str | None:
    """
    ThinkingLevel に応じて使用するモデル名を返す（C要件）。

    Args:
        thinking_level:
            "flash-lite" → flash-lite モデル（eduanima-flash 用、Minimal思考）
            "flash-low"  → flash モデル（eduanima-pro 用、Low思考）
            "flash" | "" → flash モデル（eduanima 用、Minimal思考）

    Returns:
        使用するモデル名。クライアント未初期化時は None。
    """
    if _gemini_client is None:
        return None
    if thinking_level == "flash-lite":
        return _gemini_model_flash_lite
    # "flash", "flash-low", またはデフォルト → flash モデルを使用
    return _gemini_model_flash


def get_thinking_config_for_level(thinking_level: str):
    """
    ThinkingLevel に応じた ThinkingConfig を返す（C要件）。

    Gemini 3 の thinking_level enum（thinking_budget は廃止済み）:
        ThinkingLevel.MINIMAL = 最小限の思考（最速、Flash-Lite デフォルト）
        ThinkingLevel.LOW     = 軽量な思考（精度と速度のバランス）
        ThinkingLevel.MEDIUM  = バランス型思考（3.1 Pro 向け）
        ThinkingLevel.HIGH    = 最大深度の思考（デフォルト: Pro / Flash）

    Level → Librarian Thinking 対応:
        "flash-lite" → None（flash-lite はデフォルトで MINIMAL 相当）
        "flash"      → ThinkingLevel.MINIMAL（eduanima 用）
        "flash-low"  → ThinkingLevel.LOW（eduanima-pro 用）

    Args:
        thinking_level: "flash-lite" | "flash" | "flash-low" | ""

    Returns:
        ThinkingConfig オブジェクト、または None（flash-lite デフォルト時）
    """
    try:
        from google.genai import types
        if thinking_level == "flash-lite":
            # flash-lite はデフォルト設定で MINIMAL 相当のため設定不要
            return None
        if thinking_level == "flash-low":
            # Low thinking（eduanima-pro: 精度と速度のバランス）
            return types.ThinkingConfig(thinking_level=types.ThinkingLevel.LOW)
        # "flash" またはデフォルト → MINIMAL（最速）
        return types.ThinkingConfig(thinking_level=types.ThinkingLevel.MINIMAL)
    except Exception:
        return None


# ─── グラフ状態 ─────────────────────────────────────────────────────


class AgentState(TypedDict):
    """LangGraph が管理する状態スキーマ。"""

    request_id: str
    user_query: str
    subject_id: str

    # 検索結果（Professor から受け取った生データ）
    search_results: list[dict[str, Any]]

    # 推論ループカウンタ
    loop_count: int

    # 最終的なエビデンスchunk_idリスト
    evidence_chunk_ids: list[str]

    # エラー情報
    error: str | None


# ─── クエリ生成 ──────────────────────────────────────────────────────


def build_search_queries(
    user_query: str,
    loop_count: int,
    missing_keywords: list[str] | None = None,
    thinking_level: str = "flash",
) -> list[str]:
    """
    Gemini Structured Output でユーザークエリから検索クエリ群を生成する。

    Temperature=1.0 + Seed=42（gemini_use.md 準拠）。
    Seed=42 でクエリの再現性を確保しつつ、Temperature=1.0 で品質を維持する。
    rationale フィールドは削除済み（レイテンシ削減）。

    フォールバック: Gemini 利用不可の場合はユーザークエリをそのまま返す。

    Args:
        user_query: ユーザーの質問
        loop_count: 現在のループ番号（0 始まり）
        missing_keywords: judge_sufficiency で検出された不足キーワード
        thinking_level: 使用するモデルレベル

    Returns:
        queries: 検索クエリのリスト
    """
    model_name = get_model_for_level(thinking_level)
    if _gemini_client is None or model_name is None:
        # フォールバック: LLM なし（Gemini 未初期化時）
        return [user_query]

    # プロンプト構築
    if loop_count == 0 or not missing_keywords:
        prompt = f"""You are an academic search query generator.
Generate diverse search queries to find relevant course material chunks for the user's question.

User question: {user_query}

Generate 2-3 distinct search queries that approach the question from different angles.
Queries should be concise and use academic terminology."""
    else:
        missing_str = "\n".join(f"- {kw}" for kw in missing_keywords)
        prompt = f"""You are an academic search query generator.
The previous search was insufficient. Generate refined queries to find the missing information.

User question: {user_query}

Missing information keywords:
{missing_str}

Generate 2-3 refined search queries that specifically target these missing keywords."""

    try:
        from google.genai import types

        thinking_config = get_thinking_config_for_level(thinking_level)
        gen_config_kwargs: dict = {
            "response_mime_type": "application/json",
            "response_schema": list[_QueryItem],
            # gemini_use.md: Temperature=1.0 MANDATORY for Gemini 3
            # Seed=42 でクエリの再現性を確保する
            "temperature": 1.0,
            "seed": 42,
        }
        if thinking_config is not None:
            gen_config_kwargs["thinking_config"] = thinking_config

        response = _gemini_client.models.generate_content(
            model=model_name,
            contents=prompt,
            config=types.GenerateContentConfig(**gen_config_kwargs),
        )
        raw_text = response.text or ""
        items: list[_QueryItem] = json.loads(raw_text)
        queries = [item["query"] for item in items if item.get("query")]
        if not queries:
            return [user_query]
        logger.debug("クエリ生成完了", queries_count=len(queries), loop_count=loop_count)
        return queries
    except Exception as exc:
        logger.warning("クエリ生成失敗、フォールバック使用", error=str(exc))
        return [user_query]


# ─── SubAgent-B: 新規チャンクフィルタリング ──────────────────────────


def filter_useful_chunks(
    user_query: str,
    new_chunks: list[dict[str, Any]],
    thinking_level: str = "flash",
) -> list[str]:
    """
    SubAgent-B: 今回の新規チャンクのみをフィルタリングして有用な chunk_id を返す。

    新規チャンクのみを対象とするため、処理量が少なく高速。
    evaluate_parallel で SubAgent-C と並列実行される。

    Temperature=1.0 + Seed=42（gemini_use.md 準拠）。

    Args:
        user_query: ユーザーの質問
        new_chunks: 今回の検索で初めて取得した新規チャンク
        thinking_level: 使用するモデルレベル

    Returns:
        useful_chunk_ids: 今回の新規チャンクの中から有用な chunk_id のリスト
    """
    if not new_chunks:
        return []

    new_chunk_ids_list = [c.get("chunk_id", "") for c in new_chunks if c.get("chunk_id")]

    model_name = get_model_for_level(thinking_level)
    if _gemini_client is None or model_name is None:
        # フォールバック: 全新規チャンクを有用とみなす
        return new_chunk_ids_list

    # 新規チャンクのテキストを構築
    new_chunks_parts = []
    for c in new_chunks[:10]:  # 最大10件
        chunk_id = c.get("chunk_id", "unknown")
        content = c.get("content", "")[:300]
        new_chunks_parts.append(f"[ID:{chunk_id}]\n{content}")
    new_chunks_text = "\n\n".join(new_chunks_parts) if new_chunks_parts else "（なし）"

    prompt = f"""You are an academic research assistant.
Filter the new search results to identify which chunks are genuinely useful for answering the user's question.

User question: {user_query}

New chunks retrieved (select useful ones by ID):
{new_chunks_text}

Available chunk IDs: {new_chunk_ids_list}

Select only the chunk IDs that contain information directly relevant to the question.
It is acceptable to return an empty list if none are useful."""

    try:
        from google.genai import types

        thinking_config = get_thinking_config_for_level(thinking_level)
        config_kwargs: dict = {
            "response_mime_type": "application/json",
            "response_schema": _FilterResult,
            # gemini_use.md: Temperature=1.0 MANDATORY for Gemini 3
            # Seed=42 で再現性を確保する
            "temperature": 1.0,
            "seed": 42,
        }
        if thinking_config is not None:
            config_kwargs["thinking_config"] = thinking_config

        response = _gemini_client.models.generate_content(
            model=model_name,
            contents=prompt,
            config=types.GenerateContentConfig(**config_kwargs),
        )
        raw_text = response.text or ""
        result: _FilterResult = json.loads(raw_text)
        useful_chunk_ids = [
            cid for cid in result.get("useful_chunk_ids", [])
            if cid in new_chunk_ids_list  # 安全チェック: 本当に新規IDのみを許可
        ]
        logger.debug(
            "SubAgent-B フィルタ完了",
            new_count=len(new_chunks),
            useful_count=len(useful_chunk_ids),
        )
        return useful_chunk_ids
    except Exception as exc:
        logger.warning("SubAgent-B フィルタ失敗、フォールバック使用", error=str(exc))
        return new_chunk_ids_list


# ─── SubAgent-C: 充足度判断 ─────────────────────────────────────────


def judge_sufficiency(
    user_query: str,
    new_chunks: list[dict[str, Any]],
    kept_chunks: list[dict[str, Any]],
    loop_count: int,
    max_loops: int,
    thinking_level: str = "flash",
) -> tuple[bool, list[str]]:
    """
    SubAgent-C: 新規＋保存済みチャンクの累積情報量で充足度を判断する（A-2）。

    evaluate_parallel で SubAgent-B と並列実行される。

    Temperature=1.0 + Seed=42（gemini_use.md 準拠）。

    Args:
        user_query: ユーザーの質問
        new_chunks: 今回の検索で初めて取得した新規チャンク
        kept_chunks: 過去のループでKeepされた精鋭チャンク（累積）
        loop_count: 現在のループ番号（1 以上）
        max_loops: ループ上限

    Returns:
        (is_sufficient, missing_keywords)
        - is_sufficient: True なら CompleteAction へ
        - missing_keywords: 不足情報の短いキーワード（次のクエリ生成に使用）
    """
    # ループ上限チェック（LLM 呼び出し前に判定）
    if loop_count >= max_loops:
        logger.info("ループ上限到達、強制終了", loop_count=loop_count, max_loops=max_loops)
        return False, []

    # 検索結果が空の場合はループ不要
    if not new_chunks and not kept_chunks:
        return False, []

    model_name = get_model_for_level(thinking_level)
    if _gemini_client is None or model_name is None:
        # フォールバック
        return False, []

    # 精鋭チャンク（過去のKeep済み）を構築（累積情報の概要）
    kept_summary_parts = []
    for i, c in enumerate(kept_chunks[:8]):  # 最大8件
        content = c.get("content", "")[:200]
        kept_summary_parts.append(f"[Kept-{i + 1}] {content}")
    kept_summary = "\n\n".join(kept_summary_parts) if kept_summary_parts else "（なし）"

    # 新規チャンクの概要（充足度判断用）
    new_chunks_parts = []
    for c in new_chunks[:10]:  # 最大10件
        content = c.get("content", "")[:200]
        new_chunks_parts.append(f"- {content}")
    new_chunks_text = "\n".join(new_chunks_parts) if new_chunks_parts else "（なし）"

    prompt = f"""You are an academic research evaluator.

User question: {user_query}

## Already collected evidence (from previous loops):
{kept_summary}

## New chunks retrieved this loop (summaries):
{new_chunks_text}

Evaluate whether the combination of already collected evidence AND new chunks is sufficient to answer the user's question completely.

Rules:
- is_sufficient: true only if the combined evidence can fully answer the question
- missing_keywords: short keywords (1-3 words each) for information still missing (empty if sufficient)

Be strict: only mark sufficient if the question can be answered thoroughly."""

    try:
        from google.genai import types

        thinking_config = get_thinking_config_for_level(thinking_level)
        config_kwargs: dict = {
            "response_mime_type": "application/json",
            "response_schema": _SufficiencyResult,
            # gemini_use.md: Temperature=1.0 MANDATORY for Gemini 3
            # Seed=42 で判断の再現性を確保する
            "temperature": 1.0,
            "seed": 42,
        }
        if thinking_config is not None:
            config_kwargs["thinking_config"] = thinking_config

        response = _gemini_client.models.generate_content(
            model=model_name,
            contents=prompt,
            config=types.GenerateContentConfig(**config_kwargs),
        )
        raw_text = response.text or ""
        result: _SufficiencyResult = json.loads(raw_text)
        is_sufficient = bool(result.get("is_sufficient", False))
        missing_keywords = list(result.get("missing_keywords", []))
        logger.debug(
            "SubAgent-C 充足度判断完了",
            is_sufficient=is_sufficient,
            missing_count=len(missing_keywords),
            loop_count=loop_count,
        )
        return is_sufficient, missing_keywords
    except Exception as exc:
        logger.warning("SubAgent-C 充足度判断失敗、フォールバック使用", error=str(exc))
        return False, []


# ─── 並列評価（SubAgent B/C 同時実行）──────────────────────────────────


def evaluate_parallel(
    user_query: str,
    new_chunks: list[dict[str, Any]],
    kept_chunks: list[dict[str, Any]],
    loop_count: int,
    max_loops: int,
    thinking_level: str = "flash",
) -> tuple[list[str], bool, list[str]]:
    """
    SubAgent-B と SubAgent-C を ThreadPoolExecutor で並列実行する。

    - SubAgent-B (filter_useful_chunks): 新規チャンクのみをフィルタリング
    - SubAgent-C (judge_sufficiency):    新規+保存済みで充足度を判断

    両者は独立した入力を処理するため、並列化によりレイテンシを削減できる。

    Args:
        user_query: ユーザーの質問
        new_chunks: 今回の検索で初めて取得した新規チャンク
        kept_chunks: 過去のループでKeepされた精鋭チャンク（累積）
        loop_count: 現在のループ番号（1 以上）
        max_loops: ループ上限
        thinking_level: 使用するモデルレベル

    Returns:
        (useful_chunk_ids, is_sufficient, missing_keywords)
        - useful_chunk_ids: 今回の新規から有用な chunk_id のリスト
        - is_sufficient: True なら CompleteAction へ
        - missing_keywords: 不足情報のキーワード（次のクエリ生成に使用）
    """
    # ループ上限の早期チェック（LLM 呼び出しを節約）
    if loop_count >= max_loops:
        logger.info("ループ上限到達、並列評価スキップ", loop_count=loop_count, max_loops=max_loops)
        all_new_ids = [c.get("chunk_id", "") for c in new_chunks if c.get("chunk_id")]
        return all_new_ids, False, []

    if not new_chunks and not kept_chunks:
        return [], False, []

    useful_chunk_ids: list[str] = []
    is_sufficient: bool = False
    missing_keywords: list[str] = []

    try:
        with ThreadPoolExecutor(max_workers=2) as executor:
            future_b = executor.submit(
                filter_useful_chunks,
                user_query,
                new_chunks,
                thinking_level,
            )
            future_c = executor.submit(
                judge_sufficiency,
                user_query,
                new_chunks,
                kept_chunks,
                loop_count,
                max_loops,
                thinking_level,
            )

            # 各 Future の戻り型が異なるため、直接 .result() を呼ぶ
            # （as_completed は型推論を壊すため使わない。並列実行は submit の時点で完了済み）
            try:
                useful_chunk_ids = future_b.result()
            except Exception as exc:
                logger.warning("SubAgent-B 並列実行失敗", error=str(exc))
                useful_chunk_ids = [
                    c.get("chunk_id", "") for c in new_chunks if c.get("chunk_id")
                ]

            try:
                is_sufficient, missing_keywords = future_c.result()
            except Exception as exc:
                logger.warning("SubAgent-C 並列実行失敗", error=str(exc))
                is_sufficient = False
                missing_keywords = []

        logger.info(
            "並列評価完了",
            useful_count=len(useful_chunk_ids),
            is_sufficient=is_sufficient,
            missing_count=len(missing_keywords),
            loop_count=loop_count,
        )
    except Exception as exc:
        logger.warning("evaluate_parallel 失敗、フォールバック使用", error=str(exc))
        useful_chunk_ids = [c.get("chunk_id", "") for c in new_chunks if c.get("chunk_id")]
        is_sufficient = False
        missing_keywords = []

    return useful_chunk_ids, is_sufficient, missing_keywords


# ─── LangGraph ノード関数 ─────────────────────────────────────────────


def should_continue_node(state: AgentState) -> str:
    """
    ループ終了条件を判定するエッジ関数（LangGraph グラフ用）。

    Note:
        実際のマルチループ制御は server.py の gRPC ストリームで行う。
        LangGraph グラフは補助的な状態管理ツールとして使用する。

    Returns:
        "search" | "complete"
    """
    if state.get("error"):
        return "complete"
    if state["loop_count"] >= 1 and len(state["search_results"]) > 0:
        return "complete"
    return "search"


def node_search(state: AgentState) -> AgentState:
    """SEARCH アクションの状態更新（クエリ生成のみ、実際の検索は Professor が行う）。"""
    logger.debug(
        "node_search",
        loop_count=state["loop_count"],
        request_id=state["request_id"],
    )
    return state


def node_complete(state: AgentState) -> AgentState:
    """COMPLETE アクションの状態更新（chunk_idベースのエビデンス設定）。"""
    logger.debug(
        "node_complete",
        results_count=len(state["search_results"]),
        request_id=state["request_id"],
    )
    # chunk_idベース: 全チャンクのIDを返す（実際の選別はserver.pyのkept_chunksで行う）
    chunk_ids = [
        r.get("chunk_id", "")
        for r in state["search_results"]
        if r.get("chunk_id")
    ]
    state["evidence_chunk_ids"] = chunk_ids
    return state


# ─── グラフ構築 ─────────────────────────────────────────────────────


def build_graph():
    """LangGraph StateGraph を構築して返す。"""
    try:
        from langgraph.graph import END, START, StateGraph

        builder = StateGraph(AgentState)
        builder.add_node("search", node_search)
        builder.add_node("complete", node_complete)

        builder.add_edge(START, "search")
        builder.add_conditional_edges(
            "search",
            should_continue_node,
            {"search": "search", "complete": "complete"},
        )
        builder.add_edge("complete", END)

        return builder.compile()

    except ImportError:
        logger.warning("langgraph が利用できません。フォールバックモードで動作します。")
        return None


_graph = None


def get_graph():
    """グラフのシングルトンを返す。"""
    global _graph
    if _graph is None:
        _graph = build_graph()
    return _graph


def deserialize_state(state_json: str) -> tuple[list[dict[str, Any]], list[str]]:
    """
    Professor から受け取った state JSON を検索結果リストと新規chunk_idリストに変換する。

    JSON スキーマ（拡張版）:
    {
      "search_results": [
        {"chunk_id": "...", "content": "...", "score": 0.9, ...},
        ...
      ],
      "new_chunk_ids": ["uuid1", "uuid2", ...]  // 今回ループで初めて取得した新規ID
    }

    Returns:
        (search_results, new_chunk_ids)
        - search_results: 全検索結果リスト
        - new_chunk_ids: 今回の新規チャンクIDリスト（空の場合はsearch_results全体を新規とみなす）
    """
    if not state_json:
        return [], []
    try:
        data = json.loads(state_json)
        search_results = data.get("search_results", [])
        new_chunk_ids = data.get("new_chunk_ids", [])
        return search_results, new_chunk_ids
    except json.JSONDecodeError:
        logger.warning("state JSON のデシリアライズに失敗しました: %s", state_json[:100])
        return [], []
