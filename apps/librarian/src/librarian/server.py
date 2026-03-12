"""
LibrarianServicer: gRPC 双方向ストリーミング実装（Phase 2: LangGraph Cyclic Graph）

Phase 2 改善点:
  - server.py は gRPC I/O アダプタのみを担う（ループ制御を LangGraph に移管）
  - LangGraph の interrupt() / Command(resume=...) で外部検索 I/O を外部化
  - ループ変数（kept_chunks, all_seen_chunk_ids, loop_count 等）を AgentState に統合
  - 70%ルールと OR条件対応は graph.py の SubAgent-C で処理

プロトコル仕様:
  1. Professor → Librarian: ThinkRequest (user_query, subject_id, constraints)
     - 初回メッセージで request_id / user_query / subject_id / constraints を受け取る
     - constraints.thinking_level でLibrarianが使用するモデルを決定する（C要件）
  2. Librarian → Professor: SearchAction
     - LangGraph の interrupt() payload から SearchAction を構築して送信
     - exclude_chunk_ids: 既読チャンクIDリスト（B-2要件）
  3. Professor → Librarian: ThinkRequest (state=JSON with search_results + new_chunk_ids)
     - 検索結果を state フィールドの JSON に格納して送信
     - Command(resume=...) でグラフを再開
  4. Librarian: LangGraph グラフが SubAgent-B/C を Fan-out/Fan-in で並列実行
     - 充足 or ループ上限: グラフが END に到達 → CompleteAction を送信
     - 不足: グラフが再び interrupt() → SearchAction を送信（次のループ）
  5. Librarian → Professor: CompleteAction
     - chunk_idベースのエビデンスリストを送信してストリームクローズ

エラー時:
  - グラフ完了後にエビデンスが空 → ErrorAction("LOOP_LIMIT", ...) を送信
  - gRPC キャンセル → エラーをログ記録してストリームクローズ

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
    AgentState,
    deserialize_state,
    get_graph,
    init_checkpointer,
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
    gRPC LibrarianService の実装クラス（Phase 2: LangGraph Cyclic Graph）。

    Note:
        proto stubs（librarian_pb2, librarian_pb2_grpc）は `make proto` で生成される。
        生成前はインポートエラーになるため、メソッド内で遅延インポートする。

    Phase 2 の設計:
        server.py は gRPC I/O アダプタのみを担う。
        ループ制御・状態管理・SubAgent 並列実行はすべて LangGraph グラフに委ねる。
        interrupt() / Command(resume=...) でグラフと外部 gRPC I/O を連携させる。
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
        # Phase 3: Checkpointer を初期化する（PostgreSQL or MemorySaver）。
        # LIBRARIAN_DATABASE_URL が設定されていれば PostgresSaver（耐障害性モード）を使用。
        # 未設定の場合は MemorySaver（開発モード）にフォールバックする。
        init_checkpointer(cfg.database_url)

    def Think(
        self,
        request_iterator: Iterator,
        context: grpc.ServicerContext,
    ) -> Iterator:
        """
        双方向ストリーミング RPC 実装（Phase 2: LangGraph Cyclic Graph）。

        フロー:
          ① 初回 ThinkRequest を受け取る（user_query / subject_id / constraints）
          ② LangGraph グラフを起動 → node_generate_queries → node_wait_for_search(interrupt)
          ③ interrupt() payload から SearchAction を構築 → Professor へ送信
          ④ Professor の ThinkRequest（search_results）を受け取り Command(resume=...) で再開
          ⑤ LangGraph が Fan-out/Fan-in で SubAgent-B/C を並列実行
             - 充足（is_sufficient or 70%ルール） → CompleteAction を送信 → 終了
             - 不足 → ③ に戻ってリファインクエリで再検索（ループ上限まで）
          ⑥ グラフ完了後に CompleteAction / ErrorAction を送信

        state_snapshot.next が非空 → グラフが interrupt で一時停止中
        state_snapshot.next が空  → グラフが END に到達（完了）
        """
        # proto stubs を遅延インポート（make proto 後に利用可能）
        try:
            from langgraph.types import Command
            from librarian.v1 import librarian_pb2  # type: ignore[import]
        except ImportError as e:
            logger.error("proto stubs または langgraph が見つかりません。`make proto` を実行してください: %s", e)
            context.abort(grpc.StatusCode.INTERNAL, "proto stubs not generated")
            return

        request_id: str = ""
        max_loops: int = self._cfg.max_loops
        thinking_level: str = "flash"

        try:
            # ─── 初回メッセージ受信 ───────────────────────────────────────
            req = next(request_iterator)
            request_id = req.request_id
            user_query = req.user_query
            subject_id = req.subject_id
            max_loops = req.constraints.max_loops or self._cfg.max_loops
            thinking_level = req.constraints.thinking_level or "flash"
            interpreted_query = req.constraints.interpreted_query or ""
            completion_criteria = list(req.constraints.completion_criteria)

            logger.info(
                "Think セッション開始",
                **obs_fields(request_id),
                subject_id=subject_id,
                max_loops=max_loops,
                thinking_level=thinking_level,
                has_interpreted_query=bool(interpreted_query),
                completion_criteria_count=len(completion_criteria),
            )

            # ─── LangGraph グラフ取得 ─────────────────────────────────────
            graph = get_graph()
            if graph is None:
                logger.error("LangGraph グラフが利用できません", **obs_fields(request_id))
                context.abort(grpc.StatusCode.INTERNAL, "LangGraph graph unavailable")
                return

            # スレッドIDはリクエストIDを使用（セッション内で一意）
            config = {"configurable": {"thread_id": request_id}}

            # AgentState の初期値（server.py のループ変数を統合）
            initial_state: AgentState = {
                "request_id": request_id,
                "user_query": user_query,
                "subject_id": subject_id,
                "interpreted_query": interpreted_query,
                "completion_criteria": completion_criteria,
                "loop_count": 0,
                "max_loops": max_loops,
                "thinking_level": thinking_level,
                "kept_chunks": [],
                "all_seen_chunk_ids": [],
                "missing_keywords": [],
                "search_directives": [],   # Phase 4: SubAgent-C の検索戦略指示
                "tried_queries": [],       # Phase 4: 全ループを通じて試したクエリ履歴
                "current_queries": [],
                "current_rationale": "",
                "search_results": [],
                "new_chunk_ids": [],
                "useful_chunk_ids": [],
                "is_sufficient": False,
                "satisfied_ratio": 0.0,
                "evidence_chunk_ids": [],
                "error": None,
            }

            # ─── グラフを最初の interrupt まで実行 ───────────────────────
            for _ in graph.stream(initial_state, config=config, stream_mode="updates"):
                pass  # イベントを消費（interrupt に到達するまで実行）

            # ─── interrupt ループ（gRPC I/O ← → LangGraph 連携） ─────────
            state_snapshot = graph.get_state(config)

            while state_snapshot.next:
                # interrupt payload を取得（node_wait_for_search が設定）
                interrupt_payload: dict = {}
                for task in state_snapshot.tasks:
                    for interrupt_item in task.interrupts:
                        interrupt_payload = interrupt_item.value
                        break
                    if interrupt_payload:
                        break

                queries: list[str] = interrupt_payload.get("queries", [])
                rationale: str = interrupt_payload.get("rationale", "")
                exclude_chunk_ids: list[str] = interrupt_payload.get("exclude_chunk_ids", [])
                current_loop = state_snapshot.values.get("loop_count", 1)

                logger.info(
                    "SearchAction 送信",
                    **obs_fields(request_id),
                    queries=queries,
                    loop=current_loop,
                    exclude_count=len(exclude_chunk_ids),
                )

                # SearchAction を Professor へ送信
                search_action = librarian_pb2.SearchAction(
                    queries_text=queries,
                    queries_vector=queries,  # ベクトル検索も同じクエリを使用
                    rationale=rationale,
                    exclude_chunk_ids=exclude_chunk_ids,  # B-2: 既読チャンクを除外
                )
                yield librarian_pb2.ThinkResponse(
                    request_id=request_id,
                    search=search_action,
                )

                # Professor からの検索結果を受信
                req = next(request_iterator)
                search_results, new_chunk_ids = deserialize_state(req.state)

                # backward compat: new_chunk_ids が空の場合は全体を新規とみなす
                if not new_chunk_ids and search_results:
                    new_chunk_ids = [
                        r.get("chunk_id", "") for r in search_results if r.get("chunk_id")
                    ]

                logger.info(
                    "検索結果受信",
                    **obs_fields(request_id),
                    total_results=len(search_results),
                    new_chunks=len(new_chunk_ids),
                    loop=current_loop,
                )

                # Command(resume=...) でグラフを再開
                # node_wait_for_search の interrupt() に search_results と new_chunk_ids を渡す
                for _ in graph.stream(
                    Command(resume={"results": search_results, "new_chunk_ids": new_chunk_ids}),
                    config=config,
                    stream_mode="updates",
                ):
                    pass  # イベントを消費（次の interrupt または END まで実行）

                state_snapshot = graph.get_state(config)

            # ─── グラフ完了 → CompleteAction / ErrorAction を生成 ──────────
            final_state = state_snapshot.values
            kept_chunks: list[dict] = final_state.get("kept_chunks", [])
            loop_count: int = final_state.get("loop_count", 0)
            satisfied_ratio: float = final_state.get("satisfied_ratio", 0.0)

            if not kept_chunks:
                # エビデンスが空 → LOOP_LIMIT エラー
                error_action = librarian_pb2.ErrorAction(
                    error_type="LOOP_LIMIT",
                    message=(
                        f"最大ループ数 {max_loops} に達しましたが、"
                        "十分なエビデンスが見つかりませんでした"
                    ),
                )
                yield librarian_pb2.ThinkResponse(
                    request_id=request_id,
                    error=error_action,
                )
                logger.warning(
                    "LOOP_LIMIT 到達",
                    **obs_fields(request_id),
                    max_loops=max_loops,
                    total_loops=loop_count,
                )
                return

            # CompleteAction: kept_chunks から chunk_id ベースのエビデンスを生成
            evidences = []
            for i, chunk in enumerate(kept_chunks):
                chunk_id = chunk.get("chunk_id", "")
                if not chunk_id:
                    continue
                evidences.append(
                    librarian_pb2.Evidence(
                        chunk_id=chunk_id,
                        why_relevant=(
                            f"ループ {loop_count} 回の探索で選別された精鋭チャンク（#{i + 1}）"
                        ),
                    )
                )

            coverage_notes = (
                f"{len(evidences)} 件のチャンクを選択しました"
                f"（{loop_count} 回の検索ループ、"
                f"satisfied_ratio={satisfied_ratio:.0%}、"
                f"thinking_level={thinking_level}）。"
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
                satisfied_ratio=satisfied_ratio,
                thinking_level=thinking_level,
            )

        except grpc.RpcError as rpc_err:
            logger.error(
                "gRPC エラー",
                **obs_fields(request_id),
                error=str(rpc_err),
            )
            raise

        except StopIteration:
            # request_iterator が予期せず終了した場合（Professor が切断）
            logger.warning(
                "リクエストストリーム予期せず終了",
                **obs_fields(request_id),
            )

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
