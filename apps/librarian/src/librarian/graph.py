"""
LangGraph ベースの推論エージェント（Phase 2: Full LangGraph Cyclic Graph）

Gemini Structured Output を使用してクエリ生成・検索結果評価・エビデンス選別を行う。

SDK: google-genai（新公式SDK）
  旧: google-generativeai（非推奨）→ 新: google-genai

Temperature 設計（gemini_use.md 準拠）:
  Temperature=1.0 MANDATORY for Gemini 3 (全関数共通)
  - Gemini 3 は Temperature=0.0 がアンチパターン（品質低下・過決定論の弊害）。
  - Seed=42 で決定論的再現性を確保する（ベストエフォート）。

SubAgent 設計（並列化 → LangGraph Fan-out/Fan-in）:
  - SubAgent-B (node_subagent_b / filter_useful_chunks):  新規チャンクのフィルタリング
  - SubAgent-C (node_subagent_c / judge_sufficiency):     新規+保存済みチャンクで充足度判断
  - LangGraph Fan-out/Fan-in で B/C を並列実行（レイテンシ削減）

充足度ルール（Phase 1改善）:
  - OR条件グループ対応: completion_criteria 内を " | " で区切るとOR条件として扱う
    例: "研究目的 | 研究の背景" → どちらか一方の情報があれば充足とみなす
  - 70%ルール: satisfied_ratio >= 0.7 の場合は is_sufficient=True（早期終了）
  - completion_criteria が空の場合は累積エビデンスで総合判断

LangGraph Cyclic Graph 設計（Phase 2改善）:
  - グラフがループ制御の主体（server.py は I/O アダプタのみ）
  - node_generate_queries → node_wait_for_search（interrupt）
    → Fan-out: node_subagent_b + node_subagent_c
    → Fan-in: node_update_evidence → should_continue_node
    → "continue": node_generate_queries（サイクル）
    → "complete": node_complete → END
  - MemorySaver で gRPC セッション間の状態を保持

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
import re
from typing import Any, TypedDict

import structlog

logger = structlog.get_logger(__name__)


# ─── Structured Output スキーマ定義 ──────────────────────────────────


class _QueryItem(TypedDict):
    """クエリ生成の Structured Output スキーマ。"""

    query: str  # 検索クエリ文字列（rationale フィールド削除: レイテンシ削減）


class _QueryBundleResult(TypedDict):
    """全文検索用/ベクトル検索用を分離したクエリ生成結果。"""

    text_queries: list[str]    # 全文検索用（短いキーワード句）
    vector_queries: list[str]  # ベクトル検索用（自然言語クエリ）


class _FilterResult(TypedDict):
    """
    SubAgent-B のフィルタリング結果スキーマ（A-1: 出力最小化）。

    新規チャンクの中から有用な chunk_id のみを選別する。
    """

    useful_chunk_ids: list[str]  # 今回の新規チャンクの中から有用な chunk_id のみ（A-3）


class _SufficiencyResult(TypedDict):
    """
    SubAgent-C の充足度判断スキーマ（Phase 4改善: 研究コーディネーター化）。

    新規＋保存済みチャンクの累積情報量で充足度を判断し、
    さらに次の検索戦略と completion_criteria の修正まで担う。
    OR条件グループ + 70%ルール対応。
    """

    is_sufficient: bool           # True: 充足（CompleteAction へ）、False: 再検索
    satisfied_ratio: float        # 充足割合（0.0〜1.0）。70%以上で早期終了を許可
    missing_keywords: list[str]   # 不足情報の短いキーワード（後方互換・補助用）
    search_directives: list[str]  # 次のループで何をどう探すかの自然言語指示
                                  # 例: ["Look for Table 3 comparing BiGRU vs BERT",
                                  #      "Find the evaluation metrics in Section 4"]
    revised_criteria: list[str]   # completion_criteria の修正版
                                  # 空の場合は現在の criteria を維持
    query_languages: list[str]    # 次ループで使う検索クエリ言語（例: ["en"], ["en", "ja"]）


# ─── Gemini クライアント（モジュールレベルシングルトン） ─────────────

_gemini_client = None                    # google.genai.Client シングルトン
_gemini_model_flash_lite: str | None = None  # flash-lite モデル名（eduanima-flash 用）
_gemini_model_flash: str | None = None   # flash モデル名（eduanima / eduanima-pro 用）

# ─── Checkpointer（モジュールレベルシングルトン） ─────────────────────

_checkpointer = None  # MemorySaver または PostgresSaver（init_checkpointer() で初期化）


def init_checkpointer(database_url: str = "") -> None:
    """
    Checkpointer を初期化する（起動時に一度だけ呼ぶ）。

    Phase 3: PostgreSQL Checkpointer（耐障害性）
      - LIBRARIAN_DATABASE_URL が設定されている場合は PostgresSaver を使用
      - PostgreSQL に LangGraph のループ状態を永続化することで
        サービス障害時もチェックポイントから再開可能になる
      - langgraph-checkpoint-postgres パッケージが必要
        （pip install "eduanima-librarian[postgres]"）

    フォールバック設計:
      - database_url が空 → MemorySaver（開発・テスト用）
      - PostgresSaver のインポート失敗 → MemorySaver にフォールバック
      - PostgresSaver の接続失敗 → MemorySaver にフォールバック

    LangGraph Studio との互換性:
      - build_graph(checkpointer=None) は常に MemorySaver を使用
      - Studio はこのパスで呼び出すため問題なし
      - 本番サーバーは init_checkpointer() → get_graph() の順で呼び出す

    Args:
        database_url: PostgreSQL 接続文字列
            例: postgresql://eduanima:pass@localhost:5432/eduanima_professor
            空文字の場合は MemorySaver を使用
    """
    global _checkpointer, _graph

    if database_url:
        try:
            from langgraph.checkpoint.postgres import PostgresSaver  # type: ignore[import]

            saver = PostgresSaver.from_conn_string(database_url)
            saver.setup()  # langgraph_checkpoint テーブルを作成（冪等）
            _checkpointer = saver
            _graph = None  # グラフを再構築させる（新しい checkpointer を使用）
            logger.info(
                "PostgresSaver 初期化完了（耐障害性モード）",
                url_prefix=database_url[:30] + "...",
            )
            return
        except ImportError:
            logger.warning(
                "langgraph-checkpoint-postgres が未インストール。"
                "MemorySaver にフォールバック。"
                "耐障害性が必要な場合: pip install 'eduanima-librarian[postgres]'"
            )
        except Exception as exc:
            logger.warning(
                "PostgresSaver 初期化失敗。MemorySaver にフォールバック",
                error=str(exc),
            )

    from langgraph.checkpoint.memory import MemorySaver

    _checkpointer = MemorySaver()
    _graph = None  # グラフを再構築させる
    logger.info("MemorySaver 使用（開発モード）")


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


# ─── グラフ状態（Phase 2: 拡張版） ──────────────────────────────────


class AgentState(TypedDict):
    """
    LangGraph が管理する状態スキーマ（Phase 2: server.py のループ変数を統合）。

    Phase 2 で追加したフィールド:
      - max_loops, thinking_level: ループ制御（server.py から移動）
      - kept_chunks, all_seen_chunk_ids: エビデンス蓄積（server.py から移動）
      - current_queries, current_rationale: 現在ループのクエリ情報
      - new_chunk_ids: 今回の検索で得られた新規チャンク ID
      - useful_chunk_ids: SubAgent-B 結果
      - is_sufficient, satisfied_ratio: SubAgent-C 結果（satisfied_ratio は Phase 1 追加）
      - missing_keywords: 不足キーワード（次クエリ生成用）
    """

    request_id: str
    user_query: str
    subject_id: str

    # Pre-search Step1 で生成された解釈済み質問と終了基準
    interpreted_query: str
    completion_criteria: list[str]

    # ループ制御（server.py から移動）
    loop_count: int
    max_loops: int
    thinking_level: str

    # 蓄積エビデンス（server.py から移動）
    kept_chunks: list[dict]
    all_seen_chunk_ids: list[str]   # set → list（JSON 直列化対応）

    # 現在ループのクエリ情報（node_generate_queries → node_wait_for_search へ）
    current_text_queries: list[str]
    current_vector_queries: list[str]
    current_rationale: str
    search_languages: list[str]   # クエリ生成に使う言語リスト（例: ["en"], ["en", "ja"]）

    # 検索結果（interrupt 後に Command(resume=...) で注入）
    search_results: list[dict]
    new_chunk_ids: list[str]

    # SubAgent 結果
    useful_chunk_ids: list[str]    # SubAgent-B
    is_sufficient: bool            # SubAgent-C
    satisfied_ratio: float         # SubAgent-C (Phase 1 追加: 70%ルール用)
    missing_keywords: list[str]    # SubAgent-C（後方互換・補助用キーワード）
    search_directives: list[str]   # SubAgent-C（Phase 4 追加: 次の検索戦略指示）
    tried_queries: list[str]       # Phase 4 追加: 全ループを通じて試したクエリ履歴

    # 最終エビデンス chunk_id リスト（node_complete で設定）
    evidence_chunk_ids: list[str]

    # エラー情報
    error: str | None


# ─── クエリ生成 ──────────────────────────────────────────────────────


def build_search_queries(
    user_query: str,
    loop_count: int,
    missing_keywords: list[str] | None = None,
    thinking_level: str = "flash",
    interpreted_query: str = "",
    search_directives: list[str] | None = None,
    tried_queries: list[str] | None = None,
    search_languages: list[str] | None = None,
) -> tuple[list[str], list[str]]:
    """
    Gemini Structured Output でユーザークエリから検索クエリ群を生成する（Phase 4改善）。

    Phase 4 改善:
      - search_directives: SubAgent-C が提案した検索指示を優先使用
      - tried_queries: 既に試みたクエリを「避けるクエリ」として明示
      - Seed=ループ依存: ループごとに Seed を変えて多様性を確保

    Temperature=1.0 + Seed=ループ依存（gemini_use.md 準拠）。
    rationale フィールドは削除済み（レイテンシ削減）。

    フォールバック: Gemini 利用不可の場合はユーザークエリをそのまま返す。

    Args:
        user_query: ユーザーの質問
        loop_count: 現在のループ番号（0 始まり）
        missing_keywords: judge_sufficiency で検出された不足キーワード（後方互換）
        thinking_level: 使用するモデルレベル
        interpreted_query: Pre-search Step1 で解釈した質問（初回クエリ生成に使用）
        search_directives: SubAgent-C（研究コーディネーター）が提案した検索指示
            ループ2以降で優先使用する
        tried_queries: 全ループで試みたクエリ履歴（重複防止用）

    Returns:
        (text_queries, vector_queries)
        - text_queries: 全文検索向けの短いキーワード句
        - vector_queries: ベクトル検索向けの自然言語クエリ
    """
    model_name = get_model_for_level(thinking_level)
    default_languages = search_languages or ["en"]
    if _gemini_client is None or model_name is None:
        # フォールバック: LLM なし（Gemini 未初期化時）
        return [user_query], [user_query]

    languages_text = ", ".join(default_languages)

    # プロンプト構築
    if loop_count == 0:
        # 初回: interpreted_query が提供されている場合はそちらを優先
        query_text = interpreted_query if interpreted_query else user_query
        prompt = f"""You are an academic search query generator.
Generate both keyword-style text-search queries and natural-language vector-search queries.

User question: {user_query}
Interpreted question: {query_text}
Target query languages: {languages_text}

Output requirements:
- text_queries: 2-3 short keyword phrases (2-6 words each), NOT full sentences
- vector_queries: 2-3 natural language queries for semantic search
Use academic terminology."""
    elif search_directives:
        # ループ2以降 + search_directives あり: SubAgent-C の指示を優先
        directives_str = "\n".join(f"- {d}" for d in search_directives)
        tried_str = ""
        if tried_queries:
            tried_str = "\n\nAlready tried queries (do NOT generate these again):\n" + \
                        "\n".join(f"- {q}" for q in tried_queries)
        prompt = f"""You are an academic search query generator.
The research coordinator has provided specific search directives for the next round.

User question: {user_query}
Target query languages: {languages_text}

Search directives from research coordinator (follow these closely):
{directives_str}{tried_str}

Generate both:
- text_queries: 2-3 short keyword phrases (2-6 words each), NOT full sentences
- vector_queries: 2-3 natural language queries
Convert each directive into concrete queries and be specific."""
    elif missing_keywords:
        # ループ2以降 + missing_keywords のみ: 不足キーワードベース
        missing_str = "\n".join(f"- {kw}" for kw in missing_keywords)
        tried_str = ""
        if tried_queries:
            tried_str = "\n\nAlready tried queries (do NOT generate these again):\n" + \
                        "\n".join(f"- {q}" for q in tried_queries)
        prompt = f"""You are an academic search query generator.
The previous search was insufficient. Generate refined queries to find the missing information.

User question: {user_query}
Target query languages: {languages_text}

Missing information keywords:
{missing_str}{tried_str}

Generate both:
- text_queries: 2-3 short keyword phrases (2-6 words each), NOT full sentences
- vector_queries: 2-3 natural language queries
Approach the topic from different angles than before."""
    else:
        # フォールバック: missing_keywords も search_directives もない場合
        tried_str = ""
        if tried_queries:
            tried_str = "\n\nAlready tried queries (do NOT generate these again):\n" + \
                        "\n".join(f"- {q}" for q in tried_queries)
        query_text = interpreted_query if interpreted_query else user_query
        prompt = f"""You are an academic search query generator.
Generate both keyword-style text-search queries and natural-language vector-search queries.

User question: {user_query}
Interpreted question: {query_text}
Target query languages: {languages_text}{tried_str}

Generate both:
- text_queries: 2-3 short keyword phrases (2-6 words each), NOT full sentences
- vector_queries: 2-3 natural language queries
Use academic terminology."""

    try:
        from google.genai import types

        thinking_config = get_thinking_config_for_level(thinking_level)
        # Seed をループごとに変えて多様なクエリを生成する（Phase 4改善）
        seed = 42 + loop_count * 7
        gen_config_kwargs: dict = {
            "response_mime_type": "application/json",
            "response_schema": _QueryBundleResult,
            # gemini_use.md: Temperature=1.0 MANDATORY for Gemini 3
            "temperature": 1.0,
            "seed": seed,
        }
        if thinking_config is not None:
            gen_config_kwargs["thinking_config"] = thinking_config

        response = _gemini_client.models.generate_content(
            model=model_name,
            contents=prompt,
            config=types.GenerateContentConfig(**gen_config_kwargs),
        )
        raw_text = response.text or ""

        result: _QueryBundleResult = json.loads(raw_text)
        text_queries = [q.strip() for q in result.get("text_queries", []) if q and q.strip()]
        vector_queries = [q.strip() for q in result.get("vector_queries", []) if q and q.strip()]

        # 全文検索向けクエリは文ではなくキーワード句に寄せる
        normalized_text_queries: list[str] = []
        for q in text_queries:
            compact = re.sub(r"[\s\n\t]+", " ", q).strip()
            compact = re.sub(r"[?!.。]+$", "", compact)
            normalized_text_queries.append(compact)

        text_queries = list(dict.fromkeys(normalized_text_queries))
        vector_queries = list(dict.fromkeys(vector_queries))

        if missing_keywords:
            # reasoning 問題で不足キーワードを明示した短句を追加し、再検索の多様性を高める
            keyword_phrase = " ".join(kw.strip() for kw in missing_keywords[:3] if kw.strip())
            if keyword_phrase:
                text_queries.append(keyword_phrase)
                vector_queries.append(
                    f"Find exact evidence for: {keyword_phrase}"
                )

        if not text_queries:
            text_queries = [user_query]
        if not vector_queries:
            vector_queries = [user_query]

        logger.debug(
            "クエリ生成完了",
            text_queries_count=len(text_queries),
            vector_queries_count=len(vector_queries),
            loop_count=loop_count,
            used_directives=bool(search_directives),
            search_languages=default_languages,
        )
        return text_queries, vector_queries
    except Exception as exc:
        logger.warning("クエリ生成失敗、フォールバック使用", error=str(exc))
        return [user_query], [user_query]


# ─── SubAgent-B: 新規チャンクフィルタリング ──────────────────────────


def filter_useful_chunks(
    user_query: str,
    new_chunks: list[dict[str, Any]],
    thinking_level: str = "flash",
) -> list[str]:
    """
    SubAgent-B: 今回の新規チャンクのみをフィルタリングして有用な chunk_id を返す。

    新規チャンクのみを対象とするため、処理量が少なく高速。
    LangGraph Fan-out で SubAgent-C と並列実行される（node_subagent_b）。

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


# ─── SubAgent-C: 充足度判断（Phase 1改善） ──────────────────────────


def judge_sufficiency(
    user_query: str,
    new_chunks: list[dict[str, Any]],
    kept_chunks: list[dict[str, Any]],
    loop_count: int,
    max_loops: int,
    thinking_level: str = "flash",
    completion_criteria: list[str] | None = None,
    tried_queries: list[str] | None = None,
    search_languages: list[str] | None = None,
) -> tuple[bool, float, list[str], list[str], list[str], list[str]]:
    """
    SubAgent-C: 研究コーディネーターとして充足度判断 + 検索戦略見直し + criteria修正を行う（Phase 4改善）。

    Bug #1 修正: loop_count >= max_loops → loop_count > max_loops
      （最終ループでも LLM を実行し、充足判断を行えるようにする）

    Phase 4 改善:
      - 研究コーディネーター化: 充足度判断に加えて検索戦略と criteria 修正も担う
      - search_directives: 次のループで何をどう探すかの自然言語指示を生成
      - revised_criteria: completion_criteria が不適切な場合に修正版を提案
        （提案された場合は必ず採用する）
      - tried_queries: 既に試みたクエリを把握して重複を避ける

    Phase 1 改善（維持）:
      - OR条件グループ対応: completion_criteria 内の " | " 区切りをOR条件として扱う
      - 70%ルール: satisfied_ratio >= 0.7 の場合は is_sufficient=True

    LangGraph Fan-out で SubAgent-B と並列実行される（node_subagent_c）。

    Temperature=1.0 + Seed=ループ依存（gemini_use.md 準拠）。

    Args:
        user_query: ユーザーの質問
        new_chunks: 今回の検索で初めて取得した新規チャンク
        kept_chunks: 過去のループでKeepされた精鋭チャンク（累積）
        loop_count: 現在のループ番号（1 以上）
        max_loops: ループ上限
        thinking_level: 使用するモデルレベル
        completion_criteria: Pre-search Step1 で生成された終了基準リスト
            " | " 区切りのOR条件グループをサポート
        tried_queries: 全ループで試みた検索クエリ履歴（重複防止用）

    Returns:
        (is_sufficient, satisfied_ratio, missing_keywords, search_directives, revised_criteria, query_languages)
        - is_sufficient: True なら CompleteAction へ（70%ルール適用済み）
        - satisfied_ratio: 充足割合（0.0〜1.0）
        - missing_keywords: 不足情報の短いキーワード（後方互換・補助用）
        - search_directives: 次のループでどう探すかの自然言語指示リスト
        - revised_criteria: completion_criteria の修正版（空なら現在の criteria を維持）
        - query_languages: 次ループで使う検索言語（空なら現状維持）
    """
    current_languages = search_languages or ["en"]

    # Bug #1 修正: >= → > （最終ループでも LLM を実行）
    if loop_count > max_loops:
        logger.info("ループ上限超過、強制終了", loop_count=loop_count, max_loops=max_loops)
        return False, 0.0, [], [], [], []

    # 検索結果が空の場合はループ不要
    if not new_chunks and not kept_chunks:
        return False, 0.0, [], [], [], []

    model_name = get_model_for_level(thinking_level)
    if _gemini_client is None or model_name is None:
        # フォールバック
        return False, 0.0, [], [], [], []

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

    # 試みたクエリ履歴セクション（重複防止用）
    tried_queries_section = ""
    if tried_queries:
        tried_str = "\n".join(f"- {q}" for q in tried_queries)
        tried_queries_section = f"""

## Already tried queries (do NOT suggest these again in search_directives):
{tried_str}"""

    # completion_criteria をプロンプトに追加（Phase 1: OR条件グループ + 70%ルール）
    criteria_section = ""
    if completion_criteria:
        criteria_lines = "\n".join(f"- {c}" for c in completion_criteria)
        criteria_section = f"""

## Current completion criteria (OR groups supported):
Criteria separated by " | " mean: satisfying ANY ONE in the group counts as meeting that group.
**Scoring rule: is_sufficient=true if satisfied_ratio >= 0.7 (70%+ of groups met).**
Also set is_sufficient=true if the evidence clearly and thoroughly answers the question.

{criteria_lines}

Report satisfied_ratio = (number of satisfied groups) / (total groups).
If a criterion has " | " options, count it as 1 group satisfied if ANY option is met.

**revised_criteria**: If the current criteria are too strict, too broad, or misaligned with what the
question actually requires, provide an improved list. Otherwise return an empty list to keep them as-is."""
    else:
        criteria_section = """

## Sufficiency evaluation:
satisfied_ratio should reflect overall coverage of the question (0.0 = nothing, 1.0 = fully covered).
**Rule: is_sufficient=true if satisfied_ratio >= 0.7 or if evidence clearly answers the question.**

**revised_criteria**: If you can identify 2-5 specific sub-questions or aspects that would make a
complete answer, provide them as criteria. Otherwise return an empty list."""

    prompt = f"""You are a research coordinator managing an academic search process.
Your role is to (1) judge whether collected evidence is sufficient, (2) direct the next search if not,
and (3) refine the completion criteria if they are misaligned.

User question: {user_query}{criteria_section}{tried_queries_section}
Current search languages: {current_languages}

## Already collected evidence (from previous loops):
{kept_summary}

## New chunks retrieved this loop (summaries):
{new_chunks_text}

Evaluate the combined evidence and make THREE decisions:

**Decision 1 - Sufficiency**:
- is_sufficient: true if satisfied_ratio >= 0.7, OR if evidence thoroughly answers the question
- satisfied_ratio: proportion of criteria groups satisfied (0.0 to 1.0)
- missing_keywords: short keywords (1-3 words each) for gaps still remaining (empty if sufficient)

**Decision 2 - Search directives** (only if NOT sufficient):
- search_directives: 2-3 specific natural language instructions for the next search round
  Focus on what is MISSING. Be concrete: name specific sections, tables, figures, or concepts.
  Examples: "Find Table 3 comparing model accuracies", "Look for evaluation metrics in Section 4"
  Do NOT repeat already-tried queries.
  Return empty list if sufficient.

**Decision 3 - Criteria revision** (optional):
- revised_criteria: revised completion criteria if current ones are misaligned with the question.
  Return empty list to keep current criteria unchanged.

**Decision 4 - Query language update** (optional):
- query_languages: languages to use for next-round query generation, e.g. ["en"], ["ja"], ["en", "ja"]
  Infer from collected evidence language. If no change needed, return empty list.

Be pragmatic: 70%+ coverage is sufficient for a good answer."""

    try:
        from google.genai import types

        thinking_config = get_thinking_config_for_level(thinking_level)
        # Seed をループごとに変えて、毎回異なる視点での判断を促す
        seed = 42 + loop_count * 13
        config_kwargs: dict = {
            "response_mime_type": "application/json",
            "response_schema": _SufficiencyResult,
            # gemini_use.md: Temperature=1.0 MANDATORY for Gemini 3
            "temperature": 1.0,
            "seed": seed,
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
        satisfied_ratio = float(result.get("satisfied_ratio", 0.0))
        missing_keywords = list(result.get("missing_keywords", []))
        search_directives = list(result.get("search_directives", []))
        revised_criteria = list(result.get("revised_criteria", []))
        query_languages = list(result.get("query_languages", []))

        # 70%ルール: LLM が is_sufficient=False でも satisfied_ratio >= 0.7 なら充足
        if not is_sufficient and satisfied_ratio >= 0.7:
            is_sufficient = True
            search_directives = []  # 充足なので検索指示は不要
            query_languages = []
            logger.info(
                "70%ルール適用: 早期終了",
                satisfied_ratio=satisfied_ratio,
                loop_count=loop_count,
            )

        logger.debug(
            "SubAgent-C（研究コーディネーター）完了",
            is_sufficient=is_sufficient,
            satisfied_ratio=satisfied_ratio,
            missing_count=len(missing_keywords),
            directives_count=len(search_directives),
            criteria_revised=len(revised_criteria) > 0,
            query_languages=query_languages,
            loop_count=loop_count,
        )
        return (
            is_sufficient,
            satisfied_ratio,
            missing_keywords,
            search_directives,
            revised_criteria,
            query_languages,
        )
    except Exception as exc:
        logger.warning("SubAgent-C 充足度判断失敗、フォールバック使用", error=str(exc))
        return False, 0.0, [], [], [], []


# ─── 並列評価（後方互換ラッパー）────────────────────────────────────


def evaluate_parallel(
    user_query: str,
    new_chunks: list[dict[str, Any]],
    kept_chunks: list[dict[str, Any]],
    loop_count: int,
    max_loops: int,
    thinking_level: str = "flash",
    completion_criteria: list[str] | None = None,
) -> tuple[list[str], bool, float, list[str]]:
    """
    SubAgent-B と SubAgent-C を ThreadPoolExecutor で並列実行する（後方互換ラッパー）。

    Phase 2 では LangGraph Fan-out/Fan-in（node_subagent_b + node_subagent_c）が
    並列実行を担うため、このラッパーは直接呼び出されることは少ない。
    既存コードとの後方互換性のため維持する。

    Args:
        user_query: ユーザーの質問
        new_chunks: 今回の検索で初めて取得した新規チャンク
        kept_chunks: 過去のループでKeepされた精鋭チャンク（累積）
        loop_count: 現在のループ番号（1 以上）
        max_loops: ループ上限
        thinking_level: 使用するモデルレベル
        completion_criteria: Pre-search Step1 の終了基準リスト

    Returns:
        (useful_chunk_ids, is_sufficient, satisfied_ratio, missing_keywords)
        - useful_chunk_ids: 今回の新規から有用な chunk_id のリスト
        - is_sufficient: True なら CompleteAction へ（70%ルール適用済み）
        - satisfied_ratio: 充足割合（0.0〜1.0）
        - missing_keywords: 不足情報のキーワード（次のクエリ生成に使用）
    """
    from concurrent.futures import ThreadPoolExecutor

    # ループ上限の早期チェック（LLM 呼び出しを節約）
    if loop_count >= max_loops:
        logger.info("ループ上限到達、並列評価スキップ", loop_count=loop_count, max_loops=max_loops)
        all_new_ids = [c.get("chunk_id", "") for c in new_chunks if c.get("chunk_id")]
        return all_new_ids, False, 0.0, []

    if not new_chunks and not kept_chunks:
        return [], False, 0.0, []

    useful_chunk_ids: list[str] = []
    is_sufficient: bool = False
    satisfied_ratio: float = 0.0
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
                completion_criteria,
            )

            try:
                useful_chunk_ids = future_b.result()
            except Exception as exc:
                logger.warning("SubAgent-B 並列実行失敗", error=str(exc))
                useful_chunk_ids = [
                    c.get("chunk_id", "") for c in new_chunks if c.get("chunk_id")
                ]

            try:
                is_sufficient, satisfied_ratio, missing_keywords, _directives, _revised, _languages = future_c.result()
            except Exception as exc:
                logger.warning("SubAgent-C 並列実行失敗", error=str(exc))
                is_sufficient = False
                satisfied_ratio = 0.0
                missing_keywords = []

        logger.info(
            "並列評価完了",
            useful_count=len(useful_chunk_ids),
            is_sufficient=is_sufficient,
            satisfied_ratio=satisfied_ratio,
            missing_count=len(missing_keywords),
            loop_count=loop_count,
        )
    except Exception as exc:
        logger.warning("evaluate_parallel 失敗、フォールバック使用", error=str(exc))
        useful_chunk_ids = [c.get("chunk_id", "") for c in new_chunks if c.get("chunk_id")]
        is_sufficient = False
        satisfied_ratio = 0.0
        missing_keywords = []

    return useful_chunk_ids, is_sufficient, satisfied_ratio, missing_keywords


# ─── LangGraph ノード関数（Phase 2: 新規追加） ───────────────────────


def node_generate_queries(state: AgentState) -> dict:
    """
    クエリ生成ノード（Cyclic Graphの起点）。

    Phase 4 改善:
      - search_directives（研究コーディネーターの指示）を優先使用
      - tried_queries に今回のクエリを追記して重複防止

    loop_count をインクリメントし、next クエリを生成して
    node_wait_for_search へ渡す。
    """
    loop_count = state["loop_count"]
    missing_keywords = state.get("missing_keywords") or []
    search_directives = state.get("search_directives") or []
    tried_queries = state.get("tried_queries") or []
    search_languages = state.get("search_languages") or ["en"]

    text_queries, vector_queries = build_search_queries(
        user_query=state["user_query"],
        loop_count=loop_count,
        missing_keywords=missing_keywords if missing_keywords else None,
        thinking_level=state.get("thinking_level", "flash"),
        interpreted_query=state.get("interpreted_query", ""),
        search_directives=search_directives if search_directives else None,
        tried_queries=tried_queries if tried_queries else None,
        search_languages=search_languages,
    )

    # UX 向け rationale をローカルで生成（LLM 呼び出し不要）
    if loop_count == 0:
        rationale = (
            f"「{vector_queries[0][:30]}」に関連する資料を検索しています..."
            if vector_queries else "資料を検索しています..."
        )
    else:
        rationale = (
            f"「{vector_queries[0][:30]}」に関する追加情報を検索しています..."
            if vector_queries else "資料を追加検索しています..."
        )

    # tried_queries に今回のクエリを追記（重複防止）
    new_tried_queries = list(dict.fromkeys(tried_queries + text_queries + vector_queries))

    logger.debug(
        "node_generate_queries",
        loop_count=loop_count + 1,
        text_queries_count=len(text_queries),
        vector_queries_count=len(vector_queries),
        used_directives=bool(search_directives),
        search_languages=search_languages,
        request_id=state["request_id"],
    )

    return {
        "current_text_queries": text_queries,
        "current_vector_queries": vector_queries,
        "current_rationale": rationale,
        "loop_count": loop_count + 1,
        "tried_queries": new_tried_queries,
    }


def node_wait_for_search(state: AgentState) -> dict:
    """
    外部検索待機ノード（interrupt() を使って gRPC I/O を外部化）。

    interrupt() で実行を一時停止し、server.py に SearchAction の送信を委ねる。
    server.py が Professor から検索結果を受け取り Command(resume=...) で再開すると、
    search_results と new_chunk_ids が状態に注入される。
    """
    try:
        from langgraph.types import interrupt
    except ImportError:
        logger.warning("langgraph.types.interrupt が利用できません")
        return {"search_results": [], "new_chunk_ids": []}

    # interrupt: ここで実行が一時停止し、server.py に制御が戻る
    # server.py はこの payload を見て SearchAction を構築・送信する
    search_data = interrupt({
        "queries_text": state["current_text_queries"],
        "queries_vector": state["current_vector_queries"],
        "queries": state["current_vector_queries"],
        "rationale": state["current_rationale"],
        "exclude_chunk_ids": state["all_seen_chunk_ids"],
    })

    # Command(resume={"results": [...], "new_chunk_ids": [...]}) で再開される
    results = search_data.get("results", [])
    new_chunk_ids = search_data.get("new_chunk_ids", [])

    logger.debug(
        "node_wait_for_search 再開",
        results_count=len(results),
        new_count=len(new_chunk_ids),
        request_id=state["request_id"],
    )

    return {
        "search_results": results,
        "new_chunk_ids": new_chunk_ids,
    }


def node_subagent_b(state: AgentState) -> dict:
    """
    SubAgent-B ノード（Fan-out: node_subagent_c と並列実行）。

    今回の新規チャンクをフィルタリングして有用な chunk_id を返す。
    """
    new_chunk_ids_set = set(state.get("new_chunk_ids", []))
    new_chunks = [
        r for r in state.get("search_results", [])
        if r.get("chunk_id") in new_chunk_ids_set
    ]

    useful_ids = filter_useful_chunks(
        user_query=state["user_query"],
        new_chunks=new_chunks,
        thinking_level=state.get("thinking_level", "flash"),
    )

    logger.debug(
        "node_subagent_b 完了",
        new_count=len(new_chunks),
        useful_count=len(useful_ids),
        request_id=state["request_id"],
    )

    return {"useful_chunk_ids": useful_ids}


def node_subagent_c(state: AgentState) -> dict:
    """
    SubAgent-C ノード（Fan-out: node_subagent_b と並列実行）。

    Phase 4 改善: 研究コーディネーターとして充足度判断 + 検索戦略 + criteria修正を行う。
    5-tuple を受け取り、search_directives と revised_criteria を状態に返す。
    """
    new_chunk_ids_set = set(state.get("new_chunk_ids", []))
    new_chunks = [
        r for r in state.get("search_results", [])
        if r.get("chunk_id") in new_chunk_ids_set
    ]

    is_suf, satisfied_ratio, missing_kw, directives, revised_crit, query_languages = judge_sufficiency(
        user_query=state["user_query"],
        new_chunks=new_chunks,
        kept_chunks=state.get("kept_chunks", []),
        loop_count=state["loop_count"],
        max_loops=state.get("max_loops", 4),
        thinking_level=state.get("thinking_level", "flash"),
        completion_criteria=state.get("completion_criteria") or None,
        tried_queries=state.get("tried_queries") or None,
        search_languages=state.get("search_languages") or None,
    )

    logger.debug(
        "node_subagent_c 完了",
        is_sufficient=is_suf,
        satisfied_ratio=satisfied_ratio,
        missing_count=len(missing_kw),
        directives_count=len(directives),
        criteria_revised=len(revised_crit) > 0,
        query_languages=query_languages,
        request_id=state["request_id"],
    )

    return {
        "is_sufficient": is_suf,
        "satisfied_ratio": satisfied_ratio,
        "missing_keywords": missing_kw,
        "search_directives": directives,          # Phase 4: 次の検索戦略指示
        "search_languages": query_languages if query_languages else state.get("search_languages", ["en"]),
        # revised_criteria は node_update_evidence で処理（completion_criteria 更新）
        # ここでは一時保存用に missing_keywords と並列保持する
        "completion_criteria": revised_crit if revised_crit else state.get("completion_criteria", []),
    }


def node_update_evidence(state: AgentState) -> dict:
    """
    エビデンス更新ノード（Fan-in: node_subagent_b + node_subagent_c の結果を集約）。

    Phase 4 改善:
      - node_subagent_c が completion_criteria を更新済み（revised_criteria採用）
      - no-progress detection: new_chunk_ids が空の場合は is_sufficient を上書きしない
        (should_continue_node 側の no-progress early exit と連動)

    - useful_chunk_ids のチャンクを kept_chunks に追加
    - all_seen_chunk_ids を更新
    - 70%ルールを final check（SubAgent-C が適用済みだが念のため）
    """
    new_chunk_ids_set = set(state.get("new_chunk_ids", []))
    new_chunks = [
        r for r in state.get("search_results", [])
        if r.get("chunk_id") in new_chunk_ids_set
    ]

    # 有用チャンクを kept_chunks に追加
    useful_id_set = set(state.get("useful_chunk_ids", []))
    new_kept = [c for c in new_chunks if c.get("chunk_id") in useful_id_set]

    # 既読 ID セットを更新
    new_seen = list(set(state.get("all_seen_chunk_ids", []) + list(new_chunk_ids_set)))

    # 70%ルール最終確認（SubAgent-C で未適用の場合も保護）
    is_sufficient = state.get("is_sufficient", False)
    satisfied_ratio = state.get("satisfied_ratio", 0.0)
    if not is_sufficient and satisfied_ratio >= 0.7:
        is_sufficient = True

    logger.info(
        "node_update_evidence 完了",
        new_kept_count=len(new_kept),
        total_kept=len(state.get("kept_chunks", [])) + len(new_kept),
        all_seen_count=len(new_seen),
        is_sufficient=is_sufficient,
        satisfied_ratio=satisfied_ratio,
        no_progress=len(new_chunk_ids_set) == 0,
        request_id=state["request_id"],
    )

    return {
        "kept_chunks": state.get("kept_chunks", []) + new_kept,
        "all_seen_chunk_ids": new_seen,
        "is_sufficient": is_sufficient,
    }


def node_complete(state: AgentState) -> dict:
    """
    完了ノード（chunk_id ベースのエビデンス設定）。

    kept_chunks から evidence_chunk_ids を生成する。
    """
    chunk_ids = [
        c.get("chunk_id", "")
        for c in state.get("kept_chunks", [])
        if c.get("chunk_id")
    ]
    logger.debug(
        "node_complete",
        evidence_count=len(chunk_ids),
        request_id=state["request_id"],
    )
    return {"evidence_chunk_ids": chunk_ids}


# ─── 終了条件判定（conditional edge）─────────────────────────────────


def should_continue_node(state: AgentState) -> str:
    """
    ループ終了条件を判定する conditional edge 関数（Phase 2: 実際のロジック）。

    終了条件（OR）:
      1. エラー発生
      2. ループ上限到達（loop_count >= max_loops）
      3. is_sufficient=True（SubAgent-C が充足判断 or 70%ルール適用済み）

    Returns:
        "generate_queries": 再検索（次のループへ）
        "complete": 終了（CompleteAction を生成）
    """
    if state.get("error"):
        logger.info("終了判定: エラー", request_id=state["request_id"])
        return "complete"

    # no-progress early exit: ループ1以降で新規チャンクが0件ならそれ以上の検索は無意味
    # 回答不能ケースの無駄ループを抑制する。
    if state["loop_count"] >= 1 and len(state.get("new_chunk_ids", [])) == 0:
        logger.info(
            "終了判定: no-progress（新規チャンク0件）",
            loop_count=state["loop_count"],
            request_id=state["request_id"],
        )
        return "complete"

    if state["loop_count"] >= state.get("max_loops", 4):
        logger.info(
            "終了判定: ループ上限到達",
            loop_count=state["loop_count"],
            max_loops=state.get("max_loops", 4),
            request_id=state["request_id"],
        )
        return "complete"

    if state.get("is_sufficient", False):
        logger.info(
            "終了判定: 充足（早期終了）",
            loop_count=state["loop_count"],
            satisfied_ratio=state.get("satisfied_ratio", 0.0),
            request_id=state["request_id"],
        )
        return "complete"

    logger.debug(
        "終了判定: 再検索",
        loop_count=state["loop_count"],
        satisfied_ratio=state.get("satisfied_ratio", 0.0),
        request_id=state["request_id"],
    )
    return "generate_queries"


# ─── グラフ構築（Phase 2: Cyclic + Fan-out/Fan-in + MemorySaver） ──


def build_graph(checkpointer=None):
    """
    LangGraph StateGraph を構築して返す（Phase 2: 完全なサイクルグラフ）。

    LangGraph Studio から引数なしで呼び出し可能。
    checkpointer=None の場合は MemorySaver を使用する。

    グラフ構造:
      START
        ↓
      generate_queries  (クエリ生成、loop_count インクリメント)
        ↓
      wait_for_search   (interrupt: gRPC SearchAction 送信を server.py に委ねる)
        ↓ ←──────────────── Fan-out（並列実行）
      subagent_b      subagent_c
      (filter_chunks) (judge_sufficiency)
        ↓ ←──────────────── Fan-in（両方完了待ち）
      update_evidence   (kept_chunks 更新、70%ルール最終確認)
        ↓
      should_continue_node (conditional edge)
        ├── "generate_queries" → generate_queries （サイクル）
        └── "complete"         → complete → END

    Args:
        checkpointer: LangGraph Checkpointer（None の場合は MemorySaver を使用）
            - None / MemorySaver: 開発・テスト用、LangGraph Studio との互換性を確保
            - PostgresSaver: 本番用（init_checkpointer() で初期化済みの場合）
    """
    try:
        from langgraph.checkpoint.memory import MemorySaver
        from langgraph.graph import END, START, StateGraph

        if checkpointer is None:
            checkpointer = MemorySaver()

        builder = StateGraph(AgentState)

        # ノード登録
        builder.add_node("generate_queries", node_generate_queries)
        builder.add_node("wait_for_search", node_wait_for_search)
        builder.add_node("subagent_b", node_subagent_b)
        builder.add_node("subagent_c", node_subagent_c)
        builder.add_node("update_evidence", node_update_evidence)
        builder.add_node("complete", node_complete)

        # エッジ定義
        builder.add_edge(START, "generate_queries")
        builder.add_edge("generate_queries", "wait_for_search")

        # Fan-out: wait_for_search → subagent_b + subagent_c（並列実行）
        builder.add_edge("wait_for_search", "subagent_b")
        builder.add_edge("wait_for_search", "subagent_c")

        # Fan-in: subagent_b + subagent_c → update_evidence（両方完了後に実行）
        builder.add_edge("subagent_b", "update_evidence")
        builder.add_edge("subagent_c", "update_evidence")

        # Conditional edge: 再検索 or 終了
        builder.add_conditional_edges(
            "update_evidence",
            should_continue_node,
            {
                "generate_queries": "generate_queries",
                "complete": "complete",
            },
        )

        builder.add_edge("complete", END)

        return builder.compile(checkpointer=checkpointer)

    except ImportError:
        logger.warning("langgraph が利用できません。フォールバックモードで動作します。")
        return None


_graph = None


def get_graph():
    """
    グラフのシングルトンを返す。

    init_checkpointer() 呼び出し後は _graph が None にリセットされるため、
    次の get_graph() 呼び出し時に新しい checkpointer でグラフを再構築する。
    """
    global _graph, _checkpointer
    if _graph is None:
        if _checkpointer is None:
            from langgraph.checkpoint.memory import MemorySaver
            _checkpointer = MemorySaver()
        _graph = build_graph(checkpointer=_checkpointer)
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
