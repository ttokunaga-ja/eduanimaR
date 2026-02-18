package usecases

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
)

const (
	chatSearchLimit   = 10 // 1クエリあたりの最大検索結果数
	fallbackEvidenceN = 5  // Librarian がエビデンスを返さない場合のフォールバック件数
	excerptMaxLen     = 300
)

// ChatUseCase は質問応答セッションのオーケストレーションを担う。
type ChatUseCase struct {
	subjectRepo   ports.SubjectRepository
	qaSessionRepo ports.QASessionRepository
	chunkRepo     ports.ChunkRepository
	llm           ports.LLMClient
	librarian     ports.LibrarianClient
}

// NewChatUseCase は ChatUseCase を生成する。
func NewChatUseCase(
	subjectRepo ports.SubjectRepository,
	qaSessionRepo ports.QASessionRepository,
	chunkRepo ports.ChunkRepository,
	llm ports.LLMClient,
	librarian ports.LibrarianClient,
) *ChatUseCase {
	return &ChatUseCase{
		subjectRepo:   subjectRepo,
		qaSessionRepo: qaSessionRepo,
		chunkRepo:     chunkRepo,
		llm:           llm,
		librarian:     librarian,
	}
}

// ─── Ask ─────────────────────────────────────────────────────────

// Ask は質問応答セッションを実行し、SSEイベントをコールバックに逐次渡す。
//
// フロー:
//  1. subject 所有権確認（subjectID + userID）
//  2. QASession 作成（DB永続化）
//  3. SSEEventThinking 送信
//  4. LibrarianClient.Think 呼び出し（双方向ストリーミング）
//     - onSearchRequest コールバックで全文検索・ベクトル検索を実行
//     - SSEEventSearching 送信
//  5. エビデンスチャンク選定 → SSEEventEvidence 送信
//  6. LLM 回答ストリーミング生成 → SSEEventAnswer 送信
//  7. QASession.Answer / Sources を永続化
//  8. SSEEventDone 送信
func (uc *ChatUseCase) Ask(
	ctx context.Context,
	subjectID, userID uuid.UUID,
	question string,
	onEvent func(eventType domain.SSEEventType, data any) error,
) (*domain.QASession, error) {
	// 1. subject 所有権確認
	if _, err := uc.subjectRepo.GetByIDAndUserID(ctx, subjectID, userID); err != nil {
		return nil, fmt.Errorf("get subject: %w", err)
	}

	// 2. QASession 作成
	session := &domain.QASession{
		ID:        uuid.New(),
		UserID:    userID,
		SubjectID: subjectID,
		Question:  question,
	}
	if err := uc.qaSessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create qa session: %w", err)
	}
	slog.Info("qa session created", "session_id", session.ID, "subject_id", subjectID)

	// 3. Librarian 推論開始通知
	if err := onEvent(domain.SSEEventThinking, map[string]any{
		"session_id": session.ID.String(),
		"message":    "Analyzing your question...",
	}); err != nil {
		return nil, err
	}

	// 累積検索結果（Librarian の TempIndex はこの配列のインデックスを指す）
	var allResults []domain.SearchResult
	seenChunks := make(map[uuid.UUID]struct{})

	// 4. Librarian Think（双方向ストリーミング）
	thinkResult, err := uc.librarian.Think(
		ctx,
		session.ID.String(),
		question,
		subjectID,
		userID,
		func(req ports.LibrarianSearchRequest) (*ports.LibrarianSearchResponse, error) {
			// 検索開始通知
			if evErr := onEvent(domain.SSEEventSearching, map[string]any{
				"queries_text":   req.QueriesText,
				"queries_vector": req.QueriesVector,
				"rationale":      req.Rationale,
			}); evErr != nil {
				return nil, evErr
			}

			// (A) 全文検索（text queries）
			for _, q := range req.QueriesText {
				if q == "" {
					continue
				}
				results, searchErr := uc.chunkRepo.SearchByText(ctx, subjectID, q, chatSearchLimit)
				if searchErr != nil {
					slog.Warn("text search error", "query", q, "error", searchErr)
					continue
				}
				for _, r := range results {
					if _, seen := seenChunks[r.ChunkID]; !seen {
						seenChunks[r.ChunkID] = struct{}{}
						allResults = append(allResults, *r)
					}
				}
			}

			// (B) ベクトル検索（vector queries: 各クエリを embed → HNSW 検索）
			for _, q := range req.QueriesVector {
				if q == "" {
					continue
				}
				emb, embErr := uc.llm.GenerateEmbedding(ctx, q)
				if embErr != nil {
					slog.Warn("embedding error", "query", q, "error", embErr)
					continue
				}
				vec := pgvector.NewVector(emb)
				results, searchErr := uc.chunkRepo.SearchByVector(ctx, subjectID, vec, chatSearchLimit)
				if searchErr != nil {
					slog.Warn("vector search error", "query", q, "error", searchErr)
					continue
				}
				for _, r := range results {
					if _, seen := seenChunks[r.ChunkID]; !seen {
						seenChunks[r.ChunkID] = struct{}{}
						allResults = append(allResults, *r)
					}
				}
			}

			slog.Info("search round completed",
				"text_queries", len(req.QueriesText),
				"vector_queries", len(req.QueriesVector),
				"total_accumulated", len(allResults),
			)

			// Librarian には累積された全結果を返す（TempIndex が安定する）
			return &ports.LibrarianSearchResponse{Results: allResults}, nil
		},
	)
	if err != nil {
		_ = onEvent(domain.SSEEventError, map[string]any{"message": err.Error()})
		return nil, fmt.Errorf("librarian think: %w", err)
	}

	// 5. エビデンス選定 & SSEEventEvidence 送信
	evidenceTexts := make([]string, 0, len(thinkResult.Evidences))
	sources := make([]domain.Source, 0, len(thinkResult.Evidences))

	for _, ev := range thinkResult.Evidences {
		if ev.TempIndex < 0 || ev.TempIndex >= len(allResults) {
			slog.Warn("evidence index out of range",
				"index", ev.TempIndex,
				"allResults_len", len(allResults),
			)
			continue
		}
		r := allResults[ev.TempIndex]
		evidenceTexts = append(evidenceTexts, r.Content)

		excerpt := r.Content
		if len([]rune(excerpt)) > excerptMaxLen {
			runes := []rune(excerpt)
			excerpt = string(runes[:excerptMaxLen])
		}

		sources = append(sources, domain.Source{
			FileID:     r.FileID,
			ChunkID:    r.ChunkID,
			FileName:   r.FileName,
			PageNumber: r.PageNumber,
			Excerpt:    excerpt,
		})

		_ = onEvent(domain.SSEEventEvidence, map[string]any{
			"chunk_id":     r.ChunkID.String(),
			"file_name":    r.FileName,
			"why_relevant": ev.WhyRelevant,
			"excerpt":      excerpt,
		})
	}

	// エビデンスが0件の場合: 累積検索結果の上位N件をフォールバック
	if len(evidenceTexts) == 0 && len(allResults) > 0 {
		slog.Warn("no evidences from librarian, using fallback",
			"fallback_n", fallbackEvidenceN,
			"available", len(allResults),
		)
		top := allResults
		if len(top) > fallbackEvidenceN {
			top = top[:fallbackEvidenceN]
		}
		for _, r := range top {
			evidenceTexts = append(evidenceTexts, r.Content)
		}
	}

	// 6. LLM 回答ストリーミング生成 → SSEEventAnswer
	var answerBuf strings.Builder
	streamErr := uc.llm.GenerateAnswerStream(ctx, question, evidenceTexts, func(text string) error {
		answerBuf.WriteString(text)
		return onEvent(domain.SSEEventAnswer, map[string]any{"text": text})
	})
	if streamErr != nil {
		_ = onEvent(domain.SSEEventError, map[string]any{"message": streamErr.Error()})
		return nil, fmt.Errorf("generate answer stream: %w", streamErr)
	}

	// 7. QASession.Answer / Sources を永続化
	updated, updateErr := uc.qaSessionRepo.UpdateAnswer(ctx, session.ID, answerBuf.String(), sources)
	if updateErr != nil {
		// 永続化失敗はログのみ（クライアントへのストリーミングは完了済み）
		slog.Error("failed to update qa session answer",
			"session_id", session.ID,
			"error", updateErr,
		)
	} else if updated != nil {
		session = updated
	}

	// 8. 完了通知
	_ = onEvent(domain.SSEEventDone, map[string]any{
		"session_id": session.ID.String(),
	})

	return session, nil
}

// ─── ListSessions ─────────────────────────────────────────────────

// ListSessions は指定 subject の QASession 一覧を返す。
func (uc *ChatUseCase) ListSessions(
	ctx context.Context,
	subjectID, userID uuid.UUID,
	limit, offset int,
) ([]*domain.QASession, error) {
	// subject 所有権確認
	if _, err := uc.subjectRepo.GetByIDAndUserID(ctx, subjectID, userID); err != nil {
		return nil, fmt.Errorf("get subject: %w", err)
	}
	return uc.qaSessionRepo.ListBySubjectID(ctx, subjectID, userID, limit, offset)
}

// CountSessions は指定 subject の QASession 件数を返す。
func (uc *ChatUseCase) CountSessions(
	ctx context.Context,
	subjectID, userID uuid.UUID,
) (int64, error) {
	return uc.qaSessionRepo.CountBySubjectID(ctx, subjectID, userID)
}

// ─── UpdateFeedback ───────────────────────────────────────────────

// UpdateFeedback は QASession にフィードバック（1: good / -1: bad）を記録する。
func (uc *ChatUseCase) UpdateFeedback(
	ctx context.Context,
	sessionID, userID uuid.UUID,
	feedback int,
) (*domain.QASession, error) {
	return uc.qaSessionRepo.UpdateFeedback(ctx, sessionID, userID, feedback)
}
