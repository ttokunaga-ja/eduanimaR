"""
LibrarianServicer: gRPC 双方向ストリーミング実装

プロトコル仕様:
  1. Professor → Librarian: ThinkRequest (user_query, subject_id, constraints)
     - 初回メッセージで request_id / user_query / subject_id / constraints を受け取る
  2. Librarian → Professor: SearchAction
     - Gemini で多様なクエリを生成して送信（Temperature=0.3）
     - queries_text: 全文検索用クエリ群
     - queries_vector: ベクトル検索用クエリ群（同一クエリを使用）
  3. Professor → Librarian: ThinkRequest (state=JSON with search_results)
     - 検索結果を state フィールドの JSON に格納して送信
  4. Librarian: Gemini で充足度を評価（Temperature=0.0）
     - 充足: CompleteAction を送信してセッション終了
     - 不足: 2 に戻ってリファインクエリで再検索（最大 max_loops 回）
  5. Librarian → Professor: CompleteAction
     - エビデンスリストを送信してストリームクローズ

エラー時:
  - ループ上限超過 → ErrorAction("LOOP_LIMIT", ...) を送信してストリームクローズ
  - gRPC キャンセル → ErrorAction("TIMEOUT", ...) を送信してストリームクローズ

ループ上限: max_loops (デフォルト 5、平均 3 回収束想定)
"""

from __future__ import annotations

from collections.abc import Iterator

import grpc
import structlog

from librarian.config import Config
from librarian.graph import (
    build_search_queries,
    deserialize_state,
    evaluate_search_results,
    init_gemini,
    select_evidence,
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
        # クエリ生成 (Temperature=0.3) と充足度評価 (Temperature=0.0) の
        # 2 つのモデルインスタンスを作成する。
        init_gemini(cfg.gemini_api_key, cfg.gemini_model)

    def Think(
        self,
        request_iterator: Iterator,
        context: grpc.ServicerContext,
    ) -> Iterator:
        """
        双方向ストリーミング RPC 実装。

        フロー:
          ① 初回 ThinkRequest を受け取る（user_query / subject_id）
          ② build_search_queries (Gemini, T=0.3) でクエリを生成 → SearchAction を送信
          ③ 次の ThinkRequest（state に search_results）を受け取る
          ④ evaluate_search_results (Gemini, T=0.0) で充足度を評価
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
        max_results: int = self._cfg.max_results
        current_missing_aspects: list[str] = []

        try:
            for req in request_iterator:
                # ─── 初回メッセージ処理 ─────────────────────────────────
                if loop_count == 0:
                    request_id = req.request_id
                    user_query = req.user_query
                    subject_id = req.subject_id

                    # constraints の読み取り（デフォルトフォールバック）
                    max_loops = req.constraints.max_loops or self._cfg.max_loops
                    max_results = req.constraints.max_results or self._cfg.max_results

                    logger.info(
                        "Think セッション開始",
                        **obs_fields(request_id),
                        subject_id=subject_id,
                        max_loops=max_loops,
                    )

                    # ─── SearchAction を生成して送信 ──────────────────
                    # Gemini (Temperature=0.3) で多様なクエリを生成する。
                    queries, rationale = build_search_queries(user_query, loop_count, [], None)
                    search_action = librarian_pb2.SearchAction(
                        queries_text=queries,
                        queries_vector=queries,  # ベクトル検索も同じクエリを使用
                        rationale=rationale,
                    )
                    response = librarian_pb2.ThinkResponse(
                        request_id=request_id,
                        search=search_action,
                    )
                    logger.info(
                        "SearchAction 送信",
                        **obs_fields(request_id),
                        queries=queries,
                        loop=loop_count + 1,
                    )
                    yield response
                    loop_count += 1
                    continue

                # ─── 2回目以降: 検索結果を受け取って評価 ────────────────
                search_results = deserialize_state(req.state)
                logger.info(
                    "検索結果受信",
                    **obs_fields(request_id),
                    results_count=len(search_results),
                    loop_count=loop_count,
                )

                # ─── Gemini (Temperature=0.0) で充足度を評価 ────────────
                # loop_count >= max_loops の場合は evaluate_search_results 内で
                # should_continue=False が返されるため、ここでは直接評価を呼ぶだけでよい。
                should_continue, missing_aspects, eval_reason = evaluate_search_results(
                    user_query,
                    search_results,
                    loop_count,
                    max_loops,
                )
                logger.info(
                    "充足度評価完了",
                    **obs_fields(request_id),
                    should_continue=should_continue,
                    missing_count=len(missing_aspects),
                    reason=eval_reason,
                    loop_count=loop_count,
                )

                if should_continue:
                    # ─── 不足: リファインクエリで再検索 ────────────────
                    queries, rationale = build_search_queries(
                        user_query,
                        loop_count,
                        search_results,
                        missing_aspects,
                    )
                    search_action = librarian_pb2.SearchAction(
                        queries_text=queries,
                        queries_vector=queries,
                        rationale=rationale,
                    )
                    logger.info(
                        "SearchAction 送信（リファイン）",
                        **obs_fields(request_id),
                        queries=queries,
                        loop=loop_count + 1,
                        missing_aspects=missing_aspects,
                    )
                    yield librarian_pb2.ThinkResponse(
                        request_id=request_id,
                        search=search_action,
                    )
                    loop_count += 1
                    current_missing_aspects = missing_aspects
                    continue

                # ─── 充足 or ループ上限: CompleteAction を生成して送信 ──
                # loop_count >= max_loops の場合は LOOP_LIMIT エラーを返す。
                if loop_count >= max_loops and len(search_results) == 0:
                    error_action = librarian_pb2.ErrorAction(
                        error_type="LOOP_LIMIT",
                        message=f"最大ループ数 {max_loops} に達しました",
                    )
                    yield librarian_pb2.ThinkResponse(
                        request_id=request_id,
                        error=error_action,
                    )
                    logger.warning("LOOP_LIMIT 到達", **obs_fields(request_id), max_loops=max_loops)
                    return

                evidence_list = select_evidence(search_results, max_results)
                evidences = [
                    librarian_pb2.Evidence(
                        temp_index=e["temp_index"],
                        why_relevant=e["why_relevant"],
                    )
                    for e in evidence_list
                ]
                coverage_notes = (
                    f"{len(evidence_list)} 件のチャンクを選択しました"
                    f"（{loop_count} 回の検索ループ）。"
                    f" 評価: {eval_reason}"
                    if evidence_list
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
                    evidence_count=len(evidence_list),
                    total_loops=loop_count,
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
