// Package grpc は gRPC アダプターを提供する。
package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	librarianv1 "github.com/ttokunaga-ja/eduanimaR/apps/professor/gen/proto/librarian/v1"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/ports"
)

const (
	defaultMaxLoops   = 4
	defaultMaxResults = 10
	defaultTimeoutMs  = 30000
)

// librarianClient は ports.LibrarianClient の gRPC 実装。
type librarianClient struct {
	client librarianv1.LibrarianServiceClient
}

// NewLibrarianClient は Librarian サービスへの gRPC 接続を確立して ports.LibrarianClient を返す。
func NewLibrarianClient(addr string) (ports.LibrarianClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("librarian gRPC dial: %w", err)
	}
	return &librarianClient{
		client: librarianv1.NewLibrarianServiceClient(conn),
	}, nil
}

// Think は双方向ストリーミング RPC を使って Librarian に推論を依頼する。
//
// フロー:
//  1. 初回 ThinkRequest（user_query, subject_id, thinking_level）を送信
//  2. SearchAction を受信 → onSearchRequest コールバックで検索実行
//     - ExcludeChunkIDs を使って既読チャンクをDBフィルタリング（B-2）
//  3. 検索結果を state JSON に詰めて次の ThinkRequest を送信
//     - new_chunk_ids: 今回の新規チャンクIDリスト（B-2: 蓄積型評価に使用）
//  4. CompleteAction を受信 → LibrarianThinkResult を返す
//     - Evidence は chunk_id ベース（temp_index廃止）
func (c *librarianClient) Think(
	ctx context.Context,
	requestID string,
	userQuery string,
	subjectID uuid.UUID,
	userID uuid.UUID,
	maxLoops int32,
	thinkingLevel string,
	interpretedQuery string,
	completionCriteria []string,
	onSearchRequest func(req ports.LibrarianSearchRequest) (*ports.LibrarianSearchResponse, error),
) (*ports.LibrarianThinkResult, error) {
	if maxLoops <= 0 {
		maxLoops = defaultMaxLoops
	}

	stream, err := c.client.Think(ctx)
	if err != nil {
		return nil, fmt.Errorf("open Think stream: %w", err)
	}

	// 初回リクエスト送信
	if err := stream.Send(&librarianv1.ThinkRequest{
		RequestId: requestID,
		UserQuery: userQuery,
		SubjectId: subjectID.String(),
		Constraints: &librarianv1.Constraints{
			MaxLoops:           maxLoops,
			MaxResults:         defaultMaxResults,
			TimeoutMs:          defaultTimeoutMs,
			ThinkingLevel:      thinkingLevel,      // C要件: Librarianのモデル選択に使用
			InterpretedQuery:   interpretedQuery,   // Pre-search Step1 で解釈した質問
			CompletionCriteria: completionCriteria, // judge_sufficiency に渡す終了基準
		},
	}); err != nil {
		return nil, fmt.Errorf("send initial ThinkRequest: %w", err)
	}

	// 前回ループまでの全累積チャンクIDを追跡（TempIndex廃止後のlookup用）
	allResults := make([]domain.SearchResult, 0)
	seenChunkIDs := make(map[string]struct{})

	// レスポンスループ
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("recv ThinkResponse: %w", err)
		}

		switch action := resp.Action.(type) {

		// ─── (A) 検索要求 ────────────────────────────────────────────
		case *librarianv1.ThinkResponse_Search:
			slog.Info("librarian SearchAction received",
				"request_id", requestID,
				"queries_count", len(action.Search.QueriesText),
				"rationale", action.Search.Rationale,
				"exclude_chunk_ids_count", len(action.Search.ExcludeChunkIds),
			)

			searchReq := ports.LibrarianSearchRequest{
				QueriesText:     action.Search.QueriesText,
				QueriesVector:   action.Search.QueriesVector,
				Rationale:       action.Search.Rationale,
				ExcludeChunkIDs: action.Search.ExcludeChunkIds, // B-2: 既読チャンク除外
			}
			searchResp, err := onSearchRequest(searchReq)
			if err != nil {
				_ = stream.CloseSend()
				return nil, fmt.Errorf("onSearchRequest: %w", err)
			}

			// 今回の新規チャンクIDリストを収集（B-2: seenChunkIDsで差分検出）
			newChunkIDs := make([]string, 0, len(searchResp.Results))
			for _, r := range searchResp.Results {
				chunkIDStr := r.ChunkID.String()
				if _, seen := seenChunkIDs[chunkIDStr]; !seen {
					newChunkIDs = append(newChunkIDs, chunkIDStr)
					seenChunkIDs[chunkIDStr] = struct{}{}
					allResults = append(allResults, r)
				}
			}

			// 検索結果を state JSON に直列化（new_chunk_ids を含める）
			stateJSON, err := serializeSearchResults(allResults, newChunkIDs)
			if err != nil {
				_ = stream.CloseSend()
				return nil, fmt.Errorf("serialize search results: %w", err)
			}

			// 結果を Librarian に送信
			if err := stream.Send(&librarianv1.ThinkRequest{
				RequestId: requestID,
				State:     stateJSON,
			}); err != nil {
				return nil, fmt.Errorf("send search results: %w", err)
			}

		// ─── (B) 完了 ────────────────────────────────────────────────
		case *librarianv1.ThinkResponse_Complete:
			_ = stream.CloseSend()
			slog.Info("librarian CompleteAction received",
				"request_id", requestID,
				"evidence_count", len(action.Complete.Evidence),
			)
			// chunk_id ベースのエビデンス（temp_index廃止）
			evidences := make([]ports.LibrarianEvidence, len(action.Complete.Evidence))
			for i, e := range action.Complete.Evidence {
				evidences[i] = ports.LibrarianEvidence{
					ChunkID:     e.ChunkId,
					WhyRelevant: e.WhyRelevant,
				}
			}
			return &ports.LibrarianThinkResult{
				Evidences:     evidences,
				CoverageNotes: action.Complete.CoverageNotes,
			}, nil

		// ─── (C) エラー ──────────────────────────────────────────────
		case *librarianv1.ThinkResponse_Error:
			_ = stream.CloseSend()
			slog.Warn("librarian ErrorAction received",
				"request_id", requestID,
				"error_type", action.Error.ErrorType,
				"message", action.Error.Message,
			)
			return &ports.LibrarianThinkResult{
				ErrorType:     action.Error.ErrorType,
				IsPartial:     action.Error.ErrorType == "LOOP_LIMIT",
				CoverageNotes: action.Error.Message,
			}, nil
		}
	}

	_ = stream.CloseSend()
	return &ports.LibrarianThinkResult{}, nil
}

// serializeSearchResults は検索結果を Librarian が期待する state JSON 文字列に変換する。
//
// スキーマ（拡張版）:
//
//	{
//	  "search_results": [
//	    {"chunk_id": "...", "content": "...", "file_name": "..."},
//	    ...
//	  ],
//	  "new_chunk_ids": ["uuid1", "uuid2", ...]  // 今回ループで初めて取得した新規ID
//	}
func serializeSearchResults(results []domain.SearchResult, newChunkIDs []string) (string, error) {
	type resultItem struct {
		ChunkID    string `json:"chunk_id"`
		FileID     string `json:"file_id"`
		Content    string `json:"content"`
		FileName   string `json:"file_name"`
		ChunkIndex int    `json:"chunk_index"`
	}
	items := make([]resultItem, len(results))
	for i, r := range results {
		items[i] = resultItem{
			ChunkID:    r.ChunkID.String(),
			FileID:     r.FileID.String(),
			Content:    r.Content,
			FileName:   r.FileName,
			ChunkIndex: r.ChunkIndex,
		}
	}
	b, err := json.Marshal(map[string]interface{}{
		"search_results": items,
		"new_chunk_ids":  newChunkIDs, // B-2: 蓄積型評価のための新規IDリスト
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
