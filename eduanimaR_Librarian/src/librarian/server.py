"""
LibrarianServicer: gRPC 双方向ストリーミング実装

プロトコル仕様（Phase 1 簡略版）:
  1. Professor → Librarian: ThinkRequest (user_query, subject_id)
     - 初回メッセージで request_id / user_query / subject_id / constraints を受け取る
  2. Librarian → Professor: SearchAction
     - クエリを生成して送信（全文検索 + ベクトル検索クエリ）
  3. Professor → Librarian: ThinkRequest (state=JSON with search_results)
     - 検索結果を state フィールドの JSON に格納して送信
  4. Librarian → Professor: CompleteAction
     - エビデンスインデックスリストを送信してセッション終了

エラー時:
  - gRPC コンテキストのキャンセル → ErrorAction("TIMEOUT", ...) を送信してストリームクローズ
  - ループ上限超過 → ErrorAction("LOOP_LIMIT", ...) を送信してストリームクローズ
"""

from __future__ import annotations

import logging
from typing import Iterator

import grpc

from librarian.config import Config
from librarian.graph import build_search_queries, deserialize_state, select_evidence

logger = logging.getLogger(__name__)


class LibrarianServicer:
    """
    gRPC LibrarianService の実装クラス。

    Note:
        proto stubs（librarian_pb2, librarian_pb2_grpc）は `make proto` で生成される。
        生成前はインポートエラーになるため、メソッド内で遅延インポートする。
    """

    def __init__(self, cfg: Config) -> None:
        self._cfg = cfg

    def Think(
        self,
        request_iterator: Iterator,
        context: grpc.ServicerContext,
    ) -> Iterator:
        """
        双方向ストリーミング RPC 実装。

        Phase 1 フロー:
          1. 初回 ThinkRequest を受け取る（user_query / subject_id）
          2. SearchAction を送信する
          3. 次の ThinkRequest（state に search_results）を受け取る
          4. CompleteAction を送信してストリームを終了する
        """
        # proto stubs を遅延インポート（make proto 後に利用可能）
        try:
            from librarian.v1 import librarian_pb2  # type: ignore[import]
        except ImportError as e:
            logger.error(
                "proto stubs が見つかりません。`make proto` を実行してください: %s", e
            )
            context.abort(grpc.StatusCode.INTERNAL, "proto stubs not generated")
            return

        request_id: str = ""
        loop_count: int = 0

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
                        extra={
                            "request_id": request_id,
                            "subject_id": subject_id,
                            "max_loops": max_loops,
                        },
                    )

                    # ─── SearchAction を生成して送信 ──────────────────
                    queries = build_search_queries(user_query, loop_count, [])
                    search_action = librarian_pb2.SearchAction(
                        queries_text=queries,
                        queries_vector=queries,  # Phase 1: 同じクエリをベクトル検索にも使用
                        rationale=f"ユーザークエリ「{user_query}」に関連するチャンクを検索します",
                    )
                    response = librarian_pb2.ThinkResponse(
                        request_id=request_id,
                        search=search_action,
                    )
                    logger.info(
                        "SearchAction 送信",
                        extra={"request_id": request_id, "queries": queries},
                    )
                    yield response
                    loop_count += 1
                    continue

                # ─── 2回目以降: 検索結果を受け取る ──────────────────────
                search_results = deserialize_state(req.state)
                logger.info(
                    "検索結果受信",
                    extra={
                        "request_id": request_id,
                        "results_count": len(search_results),
                        "loop_count": loop_count,
                    },
                )

                # ループ上限チェック
                if loop_count >= max_loops:
                    error_action = librarian_pb2.ErrorAction(
                        error_type="LOOP_LIMIT",
                        message=f"最大ループ数 {max_loops} に達しました",
                    )
                    yield librarian_pb2.ThinkResponse(
                        request_id=request_id,
                        error=error_action,
                    )
                    logger.warning("LOOP_LIMIT 到達", extra={"request_id": request_id})
                    return

                # ─── CompleteAction を生成して送信 ────────────────────
                evidence_list = select_evidence(search_results, max_results)
                evidences = [
                    librarian_pb2.Evidence(
                        temp_index=e["temp_index"],
                        why_relevant=e["why_relevant"],
                    )
                    for e in evidence_list
                ]
                complete_action = librarian_pb2.CompleteAction(
                    evidence=evidences,
                    coverage_notes=(
                        f"{len(evidence_list)} 件のチャンクを選択しました。"
                        if evidence_list
                        else "関連するチャンクが見つかりませんでした。"
                    ),
                )
                yield librarian_pb2.ThinkResponse(
                    request_id=request_id,
                    complete=complete_action,
                )
                logger.info(
                    "CompleteAction 送信",
                    extra={
                        "request_id": request_id,
                        "evidence_count": len(evidence_list),
                    },
                )
                # Phase 1: 1回の検索で完了
                return

        except grpc.RpcError as rpc_err:
            logger.error(
                "gRPC エラー",
                extra={"request_id": request_id, "error": str(rpc_err)},
            )
            raise

        except Exception as exc:  # noqa: BLE001
            logger.exception("Think 内部エラー", extra={"request_id": request_id})
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
