"""
LibrarianServicer: gRPC 双方向ストリーミング実装

プロトコル仕様:
  1. Professor → Librarian: ThinkRequest (user_query, subject_id, constraints)
     - 初回メッセージで request_id / user_query / subject_id / constraints を受け取る
     - constraints.thinking_level でLibrarianが使用するモデルを決定する（C要件）
  2. Librarian → Professor: SearchAction
     - Gemini で多様なクエリを生成して送信（Temperature=0.3）
     - rationale: UX表示用の日本語自然文（D要件）
     - exclude_chunk_ids: 既読チャンクIDリスト（B-2要件）
  3. Professor → Librarian: ThinkRequest (state=JSON with search_results + new_chunk_ids)
     - 検索結果を state フィールドの JSON に格納して送信
     - new_chunk_ids: 今回の新規チャンクIDリスト（B-2要件）
  4. Librarian: Gemini で蓄積型評価（A-2）とTriaging（A-3）を実行
     - 「今回の新規チャンク」＋「過去のKept精鋭チャンク」で累積評価
     - useful_chunk_ids でkept_chunksを更新
     - 充足: CompleteAction を送信してセッション終了
     - 不足: 2 に戻ってリファインクエリで再検索（最大 max_loops 回）
  5. Librarian → Professor: CompleteAction
     - chunk_idベースのエビデンスリストを送信してストリームクローズ（proto変更済み）

エラー時:
  - ループ上限超過 → ErrorAction("LOOP_LIMIT", ...) を送信してストリームクローズ
  - gRPC キャンセル → ErrorAction("TIMEOUT", ...) を送信してストリームクローズ

ThinkingLevelとループ上限:
  - eduanima-flash  (thinking_level="flash-lite"): max_loops=3（最速）
  - eduanima        (thinking_level="flash"):      max_loops=4（デフォルト）
  - eduanima-pro    (thinking_level="flash"):      max_loops=5（高品質）
  ※ LibrarianはProレベルでも flash モデルを使用する（C要件の設計方針）
"""

from __future__ import annotations

from collections.abc import Iterator

import grpc
import structlog

from librarian.config import Config
from librarian.graph import (
    build_search_queries,
    deserialize_state,
    evaluate_and_triage,
    init_gemini,
)

logger = structlog.get_logger(__name__)


def obs_fields(request_id: str, user_id: str = "unknown") -> dict[str, str]:
    """全ログで共通化する観測キーを返す。"""
    return {
        "request_id": request_id,
        "trace_id": request_id,
        "user_id": user_id,
    }


class LibrarianServicer:
    """
    gRPC LibrarianService の実装クラス。

    Note:
        proto stubs（librarian_pb2, librarian_pb2_grpc）は `make proto` で生成される。
        生成前はインポートエラーになるため、メソッド内で遅延インポートする。
    """

    def __init__(self, cfg: Config) -> None:
        self._cfg = cfg
        # 起動時に Gemini モデルを初期化する。
        # ThinkingLevel に応じてモデルを切り替えるため、2種類のモデルを初期化する。
        init_gemini(
            cfg.gemini_api_key,
            cfg.model_flash_lite,
            cfg.model_flash,
        )

    def Think(
        self,
        request_iterator: Iterator,
        context: grpc.ServicerContext,
    ) -> Iterator:
        """
        双方向ストリーミング RPC 実装。

        フロー:
          ① 初回 ThinkRequest を受け取る（user_query / subject_id / constraints）
          ② build_search_queries (Gemini) でクエリを生成 → SearchAction を送信
             rationale は日本語自然文（D要件: UX表示用）
             exclude_chunk_ids に既読IDを含める（B-2要件）
          ③ 次の ThinkRequest（state に search_results + new_chunk_ids）を受け取る
          ④ evaluate_and_triage (Gemini) で蓄積型評価とTriaging
             - 「今回の新規チャンク」＋「過去のKept精鋭チャンク」で累積評価（A-2）
             - useful_chunk_ids で kept_chunks を更新（A-3）
             - 充足: select_evidence → CompleteAction を送信してストリーム終了
             - 不足: ② に戻ってリファインクエリで再検索（ループ上限まで）
          ⑤ ループ上限超過時: ErrorAction("LOOP_LIMIT") を送信
        """
        # proto stubs を遅延インポート（make proto 後に利用可能）
        try:
            from librarian.v1 import librarian_pb2  # type: ignore[import]
        except ImportError as e:
            logger.error("proto stubs が見つかりません。`make proto` を実行してください: %s", e)
            context.abort(grpc.StatusCode.INTERNAL, "proto stubs not generated")
            return

        request_id: str = ""
        loop_count: int = 0
        user_query: str = ""
        max_loops: int = self._cfg.max_loops
        thinking_level: str = "flash"  # デフォルト: flash

        # A-2: 蓄積型エビデンス管理
        kept_chunks: list[dict] = []           # 過去のループでKeepされた精鋭チャンク
        all_seen_chunk_ids: set[str] = set()   # 全ループを通じて既読のchunk_idセット（B-2）

        # A-3: 現在のループの missing_keywords（次ループのクエリ生成に使用）
        current_missing_keywords: list[str] = []

        try:
            for req in request_iterator:
                # ─── 初回メッセージ処理 ─────────────────────────────────
                if loop_count == 0:
                    request_id = req.request_id
                    user_query = req.user_query
                    subject_id = req.subject_id

                    # constraints の読み取り（デフォルトフォールバック）
                    max_loops = req.constraints.max_loops or self._cfg.max_loops
                    # thinking_level: C要件 - Librarianが使用するモデルを決定する
                    thinking_level = req.constraints.thinking_level or "flash"

                    logger.info(
                        "Think セッション開始",
                        **obs_fields(request_id),
                        subject_id=subject_id,
                        max_loops=max_loops,
                        thinking_level=thinking_level,
                    )

                    # ─── SearchAction を生成して送信 ──────────────────
                    # Gemini でクエリを生成する。
                    # rationale は日本語自然文（D要件: UX表示）
                    queries, rationale = build_search_queries(
                        user_query,
                        loop_count,
                        missing_keywords=None,
                        thinking_level=thinking_level,
                    )
                    search_action = librarian_pb2.SearchAction(
                        queries_text=queries,
                        queries_vector=queries,  # ベクトル検索も同じクエリを使用
                        rationale=rationale,
                        exclude_chunk_ids=list(all_seen_chunk_ids),  # 初回は空
                    )
                    response = librarian_pb2.ThinkResponse(
                        request_id=request_id,
                        search=search_action,
                    )
                    logger.info(
                        "SearchAction 送信",
                        **obs_fields(request_id),
                        queries=queries,
                        rationale=rationale,
                        loop=loop_count + 1,
                    )
                    yield response
                    loop_count += 1
                    continue

                # ─── 2回目以降: 検索結果を受け取って評価 ────────────────
                # state JSON から全検索結果と今回の新規chunk_idリストを取得
                search_results, new_chunk_ids = deserialize_state(req.state)

                # new_chunk_ids が空の場合は search_results 全体を新規とみなす（後方互換）
                if not new_chunk_ids and search_results:
                    new_chunk_ids = [
                        r.get("chunk_id", "") for r in search_results if r.get("chunk_id")
                    ]

                # 今回の新規チャンクをフィルタ（B-2: 既読チェック）
                new_chunks = [
                    r for r in search_results
                    if r.get("chunk_id") in new_chunk_ids
                ]

                # 既読IDセットを更新
                for chunk_id in new_chunk_ids:
                    if chunk_id:
                        all_seen_chunk_ids.add(chunk_id)

                logger.info(
                    "検索結果受信",
                    **obs_fields(request_id),
                    total_results=len(search_results),
                    new_chunks=len(new_chunks),
                    kept_chunks=len(kept_chunks),
                    loop_count=loop_count,
                )

                # ─── Gemini で蓄積型評価・Triaging を実行 ───────────────
                # A-2: new_chunks + kept_chunks で累積評価
                # A-3: useful_chunk_ids で新規チャンクをTriaging
                is_sufficient, useful_chunk_ids, missing_keywords = evaluate_and_triage(
                    user_query,
                    new_chunks,
                    kept_chunks,
                    loop_count,
                    max_loops,
                    thinking_level=thinking_level,
                )

                # A-3: useful_chunk_ids のチャンクを kept_chunks に追加
                useful_id_set = set(useful_chunk_ids)
                for chunk in new_chunks:
                    chunk_id = chunk.get("chunk_id", "")
                    if chunk_id in useful_id_set:
                        kept_chunks.append(chunk)

                logger.info(
                    "評価・選別完了",
                    **obs_fields(request_id),
                    is_sufficient=is_sufficient,
                    useful_count=len(useful_chunk_ids),
                    kept_total=len(kept_chunks),
                    missing_count=len(missing_keywords),
                    loop_count=loop_count,
                )

                if not is_sufficient and loop_count < max_loops:
                    # ─── 不足: リファインクエリで再検索 ────────────────
                    current_missing_keywords = missing_keywords
                    queries, rationale = build_search_queries(
                        user_query,
                        loop_count,
                        missing_keywords=current_missing_keywords,
                        thinking_level=thinking_level,
                    )
                    search_action = librarian_pb2.SearchAction(
                        queries_text=queries,
                        queries_vector=queries,
                        rationale=rationale,
                        exclude_chunk_ids=list(all_seen_chunk_ids),  # B-2: 既読を除外
                    )
                    logger.info(
                        "SearchAction 送信（リファイン）",
                        **obs_fields(request_id),
                        queries=queries,
                        rationale=rationale,
                        loop=loop_count + 1,
                        missing_keywords=missing_keywords,
                        exclude_count=len(all_seen_chunk_ids),
                    )
                    yield librarian_pb2.ThinkResponse(
                        request_id=request_id,
                        search=search_action,
                    )
                    loop_count += 1
                    continue

                # ─── 充足 or ループ上限: CompleteAction を生成して送信 ──
                if len(kept_chunks) == 0 and loop_count >= max_loops:
                    # エビデンスが空でループ上限の場合 LOOP_LIMIT エラー
                    error_action = librarian_pb2.ErrorAction(
                        error_type="LOOP_LIMIT",
                        message=f"最大ループ数 {max_loops} に達しましたが、十分なエビデンスが見つかりませんでした",
                    )
                    yield librarian_pb2.ThinkResponse(
                        request_id=request_id,
                        error=error_action,
                    )
                    logger.warning("LOOP_LIMIT 到達", **obs_fields(request_id), max_loops=max_loops)
                    return

                # kept_chunks から chunk_id ベースのエビデンスを生成（A-3 + proto変更）
                evidences = []
                for i, chunk in enumerate(kept_chunks):
                    chunk_id = chunk.get("chunk_id", "")
                    if not chunk_id:
                        continue
                    evidences.append(
                        librarian_pb2.Evidence(
                            chunk_id=chunk_id,
                            why_relevant=f"ループ {loop_count} 回の探索で選別された精鋭チャンク（#{i + 1}）",
                        )
                    )

                coverage_notes = (
                    f"{len(evidences)} 件のチャンクを選択しました"
                    f"（{loop_count} 回の検索ループ、thinking_level={thinking_level}）。"
                    if evidences
                    else "関連するチャンクが見つかりませんでした。"
                )
                complete_action = librarian_pb2.CompleteAction(
                    evidence=evidences,
                    coverage_notes=coverage_notes,
                )
                yield librarian_pb2.ThinkResponse(
                    request_id=request_id,
                    complete=complete_action,
                )
                logger.info(
                    "CompleteAction 送信",
                    **obs_fields(request_id),
                    evidence_count=len(evidences),
                    total_loops=loop_count,
                    thinking_level=thinking_level,
                )
                return

        except grpc.RpcError as rpc_err:
            logger.error(
                "gRPC エラー",
                **obs_fields(request_id),
                error=str(rpc_err),
            )
            raise

        except Exception as exc:  # noqa: BLE001
            logger.exception("Think 内部エラー", **obs_fields(request_id))
            try:
                from librarian.v1 import librarian_pb2  # type: ignore[import]

                yield librarian_pb2.ThinkResponse(
                    request_id=request_id,
                    error=librarian_pb2.ErrorAction(
                        error_type="MODEL_FAILURE",
                        message=str(exc),
                    ),
                )
            except Exception:  # noqa: BLE001
                pass
            raise


def create_servicer(cfg: Config) -> LibrarianServicer:
    """LibrarianServicer インスタンスを生成して返す。"""
    return LibrarianServicer(cfg)
