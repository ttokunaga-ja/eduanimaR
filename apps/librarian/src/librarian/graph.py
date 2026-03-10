"""
LangGraph ベースの推論エージェント

Gemini Structured Output を使用してクエリ生成・検索結果評価・エビデンス選別を行う。

SDK: google-genai（新公式SDK）
  旧: google-generativeai（非推奨）→ 新: google-genai

Temperature 設計:
  - build_search_queries  (クエリ生成)    : Temperature=0.3
      多様なクエリバリエーションで検索カバレッジを向上させる。
  - evaluate_and_triage   (評価・選別)    : Temperature=0.0 + Seed=42
      決定論的な終了判断を強制する（ランダム性は誤判断の原因）。
      同時に有用なchunk_idを選別し、Professorへの情報ノイズを排除する。
      Seed=42 で再現性を確保する（Gemini 3 ベストエフォート決定論）。

出力最小化設計（A-1）:
  評価結果は is_sufficient / useful_chunk_ids / missing_keywords の3フィールドのみ。
  長文の reason/rationale は排除し、評価レイテンシを 2〜3秒 に短縮する。

蓄積型評価（A-2）:
  評価フェーズに「今回の新規チャンク」＋「過去にKeepされた精鋭チャンク」の
  両方を渡し、累積情報量で充足度を判断することでEarly Exitを実現する。

Triaging（A-3）:
  LLMが新規チャンクの中から有用なものをIDで選択し、
  ゴミ情報がProfessorに渡らないようにする。

ループ上限: max_loops (ThinkingLevelに応じて 3〜5)
  - eduanima-flash: 3ループ（最速）
  - eduanima:       4ループ（バランス型・デフォルト）
  - eduanima-pro:   5ループ（最高品質）
"""

from __future__ import annotations

import json
from typing import Any, TypedDict

import structlog

logger = structlog.get_logger(__name__)


# ─── Structured Output スキーマ定義 ──────────────────────────────────


class _QueryItem(TypedDict):
    """クエリ生成の Structured Output スキーマ。"""

    query: str       # 検索クエリ文字列
    rationale: str   # このクエリを選んだ理由（日本語自然文・UX表示用）


class _EvalResult(TypedDict):
    """
    評価・選別の Structured Output スキーマ（A-1: 出力最小化）。

    長文の reason/refined_query_hint は削除し、
    is_sufficient / useful_chunk_ids / missing_keywords のみを出力。
    これにより評価レイテンシを 2〜3秒 に短縮する。
    """

    is_sufficient: bool          # True: 充足（CompleteAction へ）、False: 再検索
    useful_chunk_ids: list[str]  # 今回の新規チャンクの中から有用なchunk_idを選別（A-3）
    missing_keywords: list[str]  # 不足している情報の短いキーワード（次のクエリ生成に使用）


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
) -> tuple[list[str], str]:
    """
    Gemini Structured Output でユーザークエリから検索クエリ群を生成する。
    Temperature=0.3: 多様なクエリバリエーションで検索カバレッジを向上させる。

    フォールバック: Gemini 利用不可の場合はユーザークエリをそのまま返す。

    Args:
        user_query: ユーザーの質問
        loop_count: 現在のループ番号（0 始まり）
        missing_keywords: evaluate_and_triage で検出された不足キーワード
        thinking_level: 使用するモデルレベル

    Returns:
        (queries, rationale)
        - queries: 検索クエリのリスト
        - rationale: 日本語の自然文（UX表示用）「〇〇について調査中...」形式
    """
    model_name = get_model_for_level(thinking_level)
    if _gemini_client is None or model_name is None:
        # フォールバック: LLM なし（Gemini 未初期化時）
        if loop_count == 0:
            return [user_query], f"「{user_query}」に関連する資料を検索しています..."
        kw = "、".join(missing_keywords[:2]) if missing_keywords else user_query
        return [user_query], f"「{kw}」に関する記述が見つからなかったため、別の角度で再調査しています..."

    # プロンプト構築
    if loop_count == 0 or not missing_keywords:
        prompt = f"""You are an academic search query generator.
Generate diverse search queries to find relevant course material chunks for the user's question.

User question: {user_query}

Generate 2-3 distinct search queries that approach the question from different angles.
Queries should be concise and use academic terminology.
The rationale field must be written in natural Japanese and will be shown to users as a progress message.
Example rationale: "決定係数の定義と計算方法について資料を検索しています。"
"""
    else:
        missing_str = "\n".join(f"- {kw}" for kw in missing_keywords)
        prompt = f"""You are an academic search query generator.
The previous search was insufficient. Generate refined queries to find the missing information.

User question: {user_query}

Missing information keywords:
{missing_str}

Generate 2-3 refined search queries that specifically target these missing keywords.
The rationale field must be written in natural Japanese and will be shown to users as a progress message.
Example rationale: "前回の検索では{missing_keywords[0]}に関する記述が見つかりませんでした。別の角度から再調査しています。"
"""

    try:
        from google.genai import types

        # thinking_config: ThinkingLevel に応じた思考バジェットを設定（C要件）
        thinking_config = get_thinking_config_for_level(thinking_level)
        gen_config_kwargs: dict = {
            "response_mime_type": "application/json",
            "response_schema": list[_QueryItem],
            # Temperature=0.3: 多様なクエリバリエーションで検索カバレッジを向上
            "temperature": 0.3,
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
        # rationale は最初のクエリのものを使用（UX表示用・日本語自然文）
        rationale = items[0].get("rationale", "") if items else ""
        if not queries:
            return [user_query], f"「{user_query}」に関連する資料を検索しています..."
        logger.debug("クエリ生成完了", queries_count=len(queries), loop_count=loop_count)
        return queries, rationale or f"「{user_query}」について資料を検索しています..."
    except Exception as exc:
        logger.warning("クエリ生成失敗、フォールバック使用", error=str(exc))
        return [user_query], f"「{user_query}」に関連する資料を検索しています..."


# ─── 評価・選別（蓄積型・A-1/A-2/A-3統合）───────────────────────────


def evaluate_and_triage(
    user_query: str,
    new_chunks: list[dict[str, Any]],
    kept_chunks: list[dict[str, Any]],
    loop_count: int,
    max_loops: int,
    thinking_level: str = "flash",
) -> tuple[bool, list[str], list[str]]:
    """
    蓄積型の充足度評価と新規チャンクのTriagingを行う（A-1/A-2/A-3統合）。

    - 「今回の新規チャンク」＋「過去のループでKeepされた精鋭チャンク」の
      両方を渡し、累積情報量で充足度を判断する（A-2: Early Exit実現）。
    - 同時に新規チャンクの中から有用なchunk_idをLLMに選別させる（A-3: Triaging）。
    - 出力は is_sufficient / useful_chunk_ids / missing_keywords のみ（A-1: 高速化）。
    - Temperature=0.0 + Seed=42: 決定論的な終了判断。

    Args:
        user_query: ユーザーの質問
        new_chunks: 今回の検索で初めて取得した新規チャンク
        kept_chunks: 過去のループでKeepされた精鋭チャンク（累積）
        loop_count: 現在のループ番号（1 以上）
        max_loops: ループ上限

    Returns:
        (is_sufficient, useful_new_chunk_ids, missing_keywords)
        - is_sufficient: True なら CompleteAction へ
        - useful_new_chunk_ids: 今回の新規から有用なchunk_idのリスト
        - missing_keywords: 不足情報の短いキーワード（次のクエリ生成に使用）
    """
    # ループ上限チェック（LLM 呼び出し前に判定）
    if loop_count >= max_loops:
        logger.info("ループ上限到達、強制終了", loop_count=loop_count, max_loops=max_loops)
        # 新規チャンクが全て有用とみなして返す（最後のチャンスなので全採用）
        all_new_ids = [c.get("chunk_id", "") for c in new_chunks if c.get("chunk_id")]
        return False, all_new_ids, []

    # 検索結果が空の場合はループ不要
    if not new_chunks and not kept_chunks:
        return False, [], []

    model_name = get_model_for_level(thinking_level)
    if _gemini_client is None or model_name is None:
        # フォールバック: 新規チャンク全採用で完了
        all_new_ids = [c.get("chunk_id", "") for c in new_chunks if c.get("chunk_id")]
        return False, all_new_ids, []

    # ─── プロンプト構築 ────────────────────────────────────────────────

    # 精鋭チャンク（過去のKeep済み）を構築（累積情報の概要）
    kept_summary_parts = []
    for i, c in enumerate(kept_chunks[:8]):  # 最大8件
        content = c.get("content", "")[:200]
        kept_summary_parts.append(f"[Kept-{i + 1}] {content}")
    kept_summary = "\n\n".join(kept_summary_parts) if kept_summary_parts else "（なし）"

    # 新規チャンク（今回取得）を構築
    new_chunks_parts = []
    for c in new_chunks[:10]:  # 最大10件
        chunk_id = c.get("chunk_id", "unknown")
        content = c.get("content", "")[:300]
        new_chunks_parts.append(f"[ID:{chunk_id}]\n{content}")
    new_chunks_text = "\n\n".join(new_chunks_parts) if new_chunks_parts else "（なし）"

    new_chunk_ids_list = [c.get("chunk_id", "") for c in new_chunks if c.get("chunk_id")]

    prompt = f"""You are an academic research evaluator.

User question: {user_query}

## Already collected evidence (from previous loops):
{kept_summary}

## New chunks retrieved this loop (select useful ones by ID):
{new_chunks_text}

## Available new chunk IDs: {new_chunk_ids_list}

Evaluate whether the combination of already collected evidence AND new chunks is sufficient to answer the user's question completely.

Rules:
- is_sufficient: true only if the combined evidence can fully answer the question
- useful_chunk_ids: select IDs from the new chunks that are genuinely relevant (can be empty if none are useful)
- missing_keywords: short keywords (1-3 words each) for information still missing (empty if sufficient)

Be strict: only mark sufficient if the question can be answered thoroughly."""

    try:
        from google.genai import types

        # thinking_config: ThinkingLevel に応じた思考バジェットを設定（C要件）
        thinking_config = get_thinking_config_for_level(thinking_level)
        eval_config_kwargs: dict = {
            "response_mime_type": "application/json",
            "response_schema": _EvalResult,
            # Temperature=0.0 + Seed=42:
            #   決定論的な終了判断を強制する（ランダム性は誤判断・無限ループの原因）。
            "temperature": 0.0,
            "seed": 42,
        }
        if thinking_config is not None:
            eval_config_kwargs["thinking_config"] = thinking_config

        response = _gemini_client.models.generate_content(
            model=model_name,
            contents=prompt,
            config=types.GenerateContentConfig(**eval_config_kwargs),
        )
        raw_text = response.text or ""
        result: _EvalResult = json.loads(raw_text)
        is_sufficient = bool(result.get("is_sufficient", False))
        useful_chunk_ids = [
            cid for cid in result.get("useful_chunk_ids", [])
            if cid in new_chunk_ids_list  # 安全チェック: 本当に新規IDのみを許可
        ]
        missing_keywords = list(result.get("missing_keywords", []))
        logger.debug(
            "評価・選別完了",
            is_sufficient=is_sufficient,
            useful_count=len(useful_chunk_ids),
            missing_count=len(missing_keywords),
            loop_count=loop_count,
        )
        return is_sufficient, useful_chunk_ids, missing_keywords
    except Exception as exc:
        logger.warning("評価・選別失敗、フォールバック使用", error=str(exc))
        # フォールバック: 新規チャンク全採用で完了
        all_new_ids = [c.get("chunk_id", "") for c in new_chunks if c.get("chunk_id")]
        return False, all_new_ids, []


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
