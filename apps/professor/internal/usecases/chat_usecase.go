package usecases

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/ports"
)

const (
	fallbackEvidenceN = 5  // Librarian がエビデンスを返さない場合のフォールバック件数
	unanswerableTopN  = 10 // 回答不能時に返す収集ソース件数
	excerptMaxLen     = 300
	rrfK              = 60 // RRF 定数（標準値: 60）
)

// topKSchedule は Dynamic Top-K のループ回数に応じた検索件数スケジュール（B-1）。
//
// [ループ1: 5件, ループ2: 8件, ループ3: 8件, ループ4以降: 4件]
// - Loop 1 (Top 5): 簡単な質問をFast Pathで最速終了させる。
// - Loop 2-3 (Top 8): 難易度が高いと判断し、視野を広げて再検索。
// - Loop 4+ (Top 4): 最終確認のみ行い、なければ素早く「回答不能」へ。
var topKSchedule = []int{5, 8, 8, 4}

func dynamicTopK(round int) int {
	if round <= 0 {
		return topKSchedule[0]
	}
	idx := round - 1
	if idx >= len(topKSchedule) {
		return topKSchedule[len(topKSchedule)-1]
	}
	return topKSchedule[idx]
}

// rrfMerge は複数のランク付き検索結果リストを Reciprocal Rank Fusion でマージする。
//
// 各リストは上位順（rank=0 が最高スコア）で渡す。
// 同一 ChunkID が複数リストに現れた場合、スコアを加算して昇格させる。
// 戻り値は RRF スコア降順にソートされた重複なしスライス。
func rrfMerge(rankedLists [][]*domain.SearchResult) []domain.SearchResult {
	type entry struct {
		result *domain.SearchResult
		score  float64
	}
	scoreMap := make(map[uuid.UUID]float64)
	resultMap := make(map[uuid.UUID]*domain.SearchResult)

	for _, list := range rankedLists {
		for rank, r := range list {
			scoreMap[r.ChunkID] += 1.0 / float64(rrfK+rank+1)
			if _, ok := resultMap[r.ChunkID]; !ok {
				cp := *r
				resultMap[r.ChunkID] = &cp
			}
		}
	}

	entries := make([]entry, 0, len(scoreMap))
	for id, score := range scoreMap {
		entries = append(entries, entry{result: resultMap[id], score: score})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].score > entries[j].score
	})

	merged := make([]domain.SearchResult, len(entries))
	for i, e := range entries {
		merged[i] = *e.result
	}
	return merged
}

// ChatUseCase は質問応答セッションのオーケストレーションを担う。
type ChatUseCase struct {
	subjectRepo   ports.SubjectRepository
	qaSessionRepo ports.QASessionRepository
	chunkRepo     ports.ChunkRepository
	fileRepo      ports.FileRepository // PDF 原本取得のためのファイルメタデータ参照用
	storage       ports.ObjectStorage  // MinIO から PDF をダウンロードするためのストレージ
	llm           ports.LLMClient
	librarian     ports.LibrarianClient
}

// AskOptions は質問応答の実行オプション。
type AskOptions struct {
	// MaxLoops は Librarian 検索ループ上限。0 以下はデフォルト値を使用。
	MaxLoops int32
	// AnswerModelOverride は回答生成モデルの上書き名。空文字はデフォルト。
	AnswerModelOverride string
	// AnswerThinkingLevel は回答生成時の thinking_level（minimal/low/medium/high）。
	AnswerThinkingLevel string
	// ThinkingLevel は Librarian が使用するモデルを決定するレベル（C要件）。
	// "flash-lite" | "flash" | "" (空文字はflashとして扱う)
	ThinkingLevel string
}

// NewChatUseCase は ChatUseCase を生成する。
func NewChatUseCase(
	subjectRepo ports.SubjectRepository,
	qaSessionRepo ports.QASessionRepository,
	chunkRepo ports.ChunkRepository,
	fileRepo ports.FileRepository,
	storage ports.ObjectStorage,
	llm ports.LLMClient,
	librarian ports.LibrarianClient,
) *ChatUseCase {
	return &ChatUseCase{
		subjectRepo:   subjectRepo,
		qaSessionRepo: qaSessionRepo,
		chunkRepo:     chunkRepo,
		fileRepo:      fileRepo,
		storage:       storage,
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
	return uc.AskWithOptions(ctx, subjectID, userID, question, AskOptions{}, onEvent)
}

// AskWithOptions は Ask の拡張版で、品質レベルに応じたループ回数とモデル切替を受け付ける。
func (uc *ChatUseCase) AskWithOptions(
	ctx context.Context,
	subjectID, userID uuid.UUID,
	question string,
	opts AskOptions,
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

	// 累積検索結果（chunk_idベースのlookupで管理）
	var allResults []domain.SearchResult
	seenChunks := make(map[uuid.UUID]struct{})
	// chunk_id → SearchResult のlookupマップ（chunk_idベースevidence処理用）
	chunkByID := make(map[string]domain.SearchResult)
	searchRounds := 0

	// 4. Librarian Think（双方向ストリーミング）
	librarianStarted := time.Now()

	thinkResult, err := uc.librarian.Think(
		ctx,
		session.ID.String(),
		question,
		subjectID,
		userID,
		opts.MaxLoops,
		opts.ThinkingLevel, // C要件: Librarianのモデル選択に使用
		func(req ports.LibrarianSearchRequest) (*ports.LibrarianSearchResponse, error) {
			searchRounds++
			// 検索開始通知
			if evErr := onEvent(domain.SSEEventSearching, map[string]any{
				"queries_text":   req.QueriesText,
				"queries_vector": req.QueriesVector,
				"rationale":      req.Rationale,
			}); evErr != nil {
				return nil, evErr
			}

			// (A) & (B) 全クエリを goroutine で並列実行 → RRF でマージ
			//
			// 有効クエリ数を数え、チャネルで結果を収集する。
			// エラー時は nil を送信してチャネルのカウントを合わせる。
			var validQueries int
			for _, q := range req.QueriesText {
				if q != "" {
					validQueries++
				}
			}
			for _, q := range req.QueriesVector {
				if q != "" {
					validQueries++
				}
			}

			// B-1: Dynamic Top-K - ループ回数に応じて検索件数を動的に変える
			topK := dynamicTopK(searchRounds)
			slog.Info("dynamic top-k applied", "round", searchRounds, "top_k", topK)

			ch := make(chan []*domain.SearchResult, validQueries)

			// (A) 全文検索（goroutine）
			for _, q := range req.QueriesText {
				if q == "" {
					continue
				}
				q := q
				go func() {
					results, err := uc.chunkRepo.SearchByText(ctx, subjectID, q, topK)
					if err != nil {
						slog.Warn("text search error", "query", q, "error", err)
						ch <- nil
						return
					}
					ch <- results
				}()
			}

			// (B) ベクトル検索（embed → HNSW、goroutine）
			for _, q := range req.QueriesVector {
				if q == "" {
					continue
				}
				q := q
				go func() {
					emb, err := uc.llm.GenerateQueryEmbedding(ctx, q)
					if err != nil {
						slog.Warn("embedding error", "query", q, "error", err)
						ch <- nil
						return
					}
					vec := pgvector.NewVector(emb)
					results, err := uc.chunkRepo.SearchByVector(ctx, subjectID, vec, topK)
					if err != nil {
						slog.Warn("vector search error", "query", q, "error", err)
						ch <- nil
						return
					}
					ch <- results
				}()
			}

			// 全 goroutine の結果を収集
			var rankedLists [][]*domain.SearchResult
			for range validQueries {
				list := <-ch
				if len(list) > 0 {
					rankedLists = append(rankedLists, list)
				}
			}

			// RRF でマージ：このラウンドの結果を RRF スコア順に統合
			roundMerged := rrfMerge(rankedLists)

			// allResults に追加（chunk_id ベースで重複除去、lookupマップにも登録）
			for _, r := range roundMerged {
				r := r
				if _, seen := seenChunks[r.ChunkID]; !seen {
					seenChunks[r.ChunkID] = struct{}{}
					allResults = append(allResults, r)
					chunkByID[r.ChunkID.String()] = r
				}
			}

			slog.Info("search round completed (RRF)",
				"text_queries", len(req.QueriesText),
				"vector_queries", len(req.QueriesVector),
				"rrf_merged", len(roundMerged),
				"total_accumulated", len(allResults),
			)

			// Librarian には累積された全結果を返す（TempIndex が安定する）
			return &ports.LibrarianSearchResponse{Results: allResults}, nil
		},
	)
	librarianMS := int(time.Since(librarianStarted).Milliseconds())
	if err != nil {
		_ = onEvent(domain.SSEEventError, map[string]any{"message": err.Error()})
		return nil, fmt.Errorf("librarian think: %w", err)
	}

	// 5. エビデンス選定 & SSEEventEvidence 送信
	evidenceTexts := make([]string, 0, len(thinkResult.Evidences))
	sources := make([]domain.Source, 0, len(thinkResult.Evidences))

	for _, ev := range thinkResult.Evidences {
		// chunk_id ベースで SearchResult を逆引き（A-3 Triaging 結果）
		r, ok := chunkByID[ev.ChunkID]
		if !ok {
			slog.Warn("evidence chunk not found in accumulated results",
				"chunk_id", ev.ChunkID,
			)
			continue
		}
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
			"file_id":      r.FileID.String(),
			"file_name":    r.FileName,
			"page_number":  r.PageNumber,
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

	// 6. PDF 原本を MinIO から取得し、LLM 回答ストリーミング生成 → SSEEventAnswer
	//
	// sources に含まれる最初の FileID の PDF を取得する。
	// 取得失敗時はテキストエビデンスのみでフォールバックする（エラーは継続しない）。
	var pdfContent []byte
	var pdfMimeType string
	if len(sources) > 0 {
		fileID := sources[0].FileID
		file, fileErr := uc.fileRepo.GetByID(ctx, fileID)
		if fileErr != nil {
			slog.Warn("failed to get file metadata for PDF answer",
				"file_id", fileID, "error", fileErr)
		} else {
			reader, dlErr := uc.storage.Download(ctx, file.StoragePath)
			if dlErr != nil {
				slog.Warn("failed to download PDF for answer",
					"file_id", fileID, "storage_path", file.StoragePath, "error", dlErr)
			} else {
				defer reader.Close()
				b, readErr := io.ReadAll(reader)
				if readErr != nil {
					slog.Warn("failed to read PDF bytes for answer",
						"file_id", fileID, "error", readErr)
				} else {
					pdfContent = b
					pdfMimeType = file.MimeType
					slog.Info("PDF loaded for answer generation",
						"file_id", fileID, "size_bytes", len(b))
				}
			}
		}
	}

	var answerBuf strings.Builder
	answerGenStarted := time.Now()
	streamErr := uc.llm.GenerateAnswerStreamWithPDF(ctx, question, evidenceTexts, pdfContent, pdfMimeType, opts.AnswerModelOverride, opts.AnswerThinkingLevel, func(text string) error {
		answerBuf.WriteString(text)
		return onEvent(domain.SSEEventAnswer, map[string]any{"text": text})
	})
	answerGenMS := int(time.Since(answerGenStarted).Milliseconds())
	if streamErr != nil {
		_ = onEvent(domain.SSEEventError, map[string]any{"message": streamErr.Error()})
		return nil, fmt.Errorf("generate answer stream with pdf: %w", streamErr)
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
	isUnanswerable := len(evidenceTexts) == 0 || thinkResult.ErrorType != ""
	answerability := "answerable"
	if isUnanswerable {
		answerability = "unanswerable"
	}
	donePayload := map[string]any{
		"session_id":      session.ID.String(),
		"coverage_notes":  thinkResult.CoverageNotes,
		"is_partial":      thinkResult.IsPartial,
		"error_type":      thinkResult.ErrorType,
		"evidence_count":  len(evidenceTexts),
		"is_unanswerable": isUnanswerable,
		"answerability":   answerability,
		"loop_count":      searchRounds,
		"librarian_ms":    librarianMS,
		"answer_gen_ms":   answerGenMS,
	}
	if isUnanswerable {
		donePayload["unanswerable_reason"] = summarizeUnanswerableReason(thinkResult.CoverageNotes, thinkResult.ErrorType)
		collected := buildCollectedSources(allResults, unanswerableTopN)
		if len(collected) > 0 {
			donePayload["collected_sources"] = collected
		}
	}
	_ = onEvent(domain.SSEEventDone, donePayload)

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

func summarizeUnanswerableReason(coverageNotes, errorType string) string {
	if strings.TrimSpace(errorType) != "" {
		return strings.TrimSpace(errorType)
	}
	if strings.TrimSpace(coverageNotes) != "" {
		return strings.TrimSpace(coverageNotes)
	}
	return "insufficient_evidence"
}

func buildCollectedSources(results []domain.SearchResult, limit int) []map[string]any {
	if len(results) == 0 || limit <= 0 {
		return nil
	}
	if len(results) < limit {
		limit = len(results)
	}
	out := make([]map[string]any, 0, limit)
	for i, r := range results[:limit] {
		excerpt := r.Content
		if len([]rune(excerpt)) > excerptMaxLen {
			runes := []rune(excerpt)
			excerpt = string(runes[:excerptMaxLen])
		}
		item := map[string]any{
			"file_id":      r.FileID.String(),
			"file_name":    r.FileName,
			"excerpt":      excerpt,
			"why_relevant": fmt.Sprintf("retrieved candidate #%d", i+1),
		}
		if r.PageNumber != nil {
			item["page_number"] = *r.PageNumber
		}
		out = append(out, item)
	}
	return out
}
