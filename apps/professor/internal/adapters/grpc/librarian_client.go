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

	librarianv1 "github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/gen/proto/librarian/v1"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
)

const (
	defaultMaxLoops   = 3
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
//  1. 初回 ThinkRequest（user_query, subject_id）を送信
//  2. SearchAction を受信 → onSearchRequest コールバックで検索実行
//  3. 検索結果を state JSON に詰めて次の ThinkRequest を送信
//  4. CompleteAction を受信 → LibrarianThinkResult を返す
func (c *librarianClient) Think(
	ctx context.Context,
	requestID string,
	userQuery string,
	subjectID uuid.UUID,
	userID uuid.UUID,
	onSearchRequest func(req ports.LibrarianSearchRequest) (*ports.LibrarianSearchResponse, error),
) (*ports.LibrarianThinkResult, error) {

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
			MaxLoops:   defaultMaxLoops,
			MaxResults: defaultMaxResults,
			TimeoutMs:  defaultTimeoutMs,
		},
	}); err != nil {
		return nil, fmt.Errorf("send initial ThinkRequest: %w", err)
	}

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
			)

			searchReq := ports.LibrarianSearchRequest{
				QueriesText:   action.Search.QueriesText,
				QueriesVector: action.Search.QueriesVector,
				Rationale:     action.Search.Rationale,
			}
			searchResp, err := onSearchRequest(searchReq)
			if err != nil {
				_ = stream.CloseSend()
				return nil, fmt.Errorf("onSearchRequest: %w", err)
			}

			// 検索結果を state JSON に直列化
			stateJSON, err := serializeSearchResults(searchResp.Results)
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
			evidences := make([]ports.LibrarianEvidence, len(action.Complete.Evidence))
			for i, e := range action.Complete.Evidence {
				evidences[i] = ports.LibrarianEvidence{
					TempIndex:   int(e.TempIndex),
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
// スキーマ:
//
//	{
//	  "search_results": [
//	    {"chunk_id": "...", "content": "...", "file_name": "..."},
//	    ...
//	  ]
//	}
func serializeSearchResults(results []domain.SearchResult) (string, error) {
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
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
