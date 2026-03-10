package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	httpmw "github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/adapters/http/middleware"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/usecases"
)

// OpenAIChatHandler は OpenAI Chat Completions 互換エンドポイントを提供する。
//
// クライアント設定例:
//
//	client = OpenAI(
//	    base_url="https://host/api/v1/subjects/{subject_id}",
//	    api_key="<Firebase ID Token>"
//	)
//	client.chat.completions.create(model="professor", messages=[...])
//
// → POST /api/v1/subjects/:subject_id/chat/completions に転送される。
type OpenAIChatHandler struct {
	uc              *usecases.ChatUseCase
	modelAnswerPro  string
	maxLoopsFast    int32
	maxLoopsStd     int32
	maxLoopsPro     int32
	modelAnswerFast string
	thinkingFast    string
	thinkingStd     string
	thinkingPro     string
}

// NewOpenAIChatHandler は OpenAIChatHandler を生成する。
func NewOpenAIChatHandler(uc *usecases.ChatUseCase) *OpenAIChatHandler {
	return &OpenAIChatHandler{
		uc:              uc,
		modelAnswerPro:  strings.TrimSpace(os.Getenv("PROFESSOR_MODEL_ANSWER_PRO")),
		maxLoopsFast:    parseInt32Env("PROFESSOR_MAX_LOOPS_FAST", 3),
		maxLoopsStd:     parseInt32Env("PROFESSOR_MAX_LOOPS_DEFAULT", 4),
		maxLoopsPro:     parseInt32Env("PROFESSOR_MAX_LOOPS_PRO", 5),
		modelAnswerFast: parseStringEnv("PROFESSOR_MODEL_ANSWER_FAST", "gemini-3.1-flash-lite-preview"),
		thinkingFast:    parseStringEnv("PROFESSOR_THINKING_ANSWER_FAST", "minimal"),
		thinkingStd:     parseStringEnv("PROFESSOR_THINKING_ANSWER_DEFAULT", "low"),
		thinkingPro:     parseStringEnv("PROFESSOR_THINKING_ANSWER_PRO", "medium"),
	}
}

// Register は Echo グループにルートを登録する。
func (h *OpenAIChatHandler) Register(g *echo.Group) {
	g.POST("", h.ChatCompletions)
}

// ─── リクエスト / レスポンス型 ─────────────────────────────────────

// openaiChatRequest は OpenAI Chat Completions API のリクエストボディ。
type openaiChatRequest struct {
	// Model は professor では無視される（subject_id は URL パスから取得）。
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
	// Stream が true または nil の場合は SSE ストリーミング応答を返す。
	// false の場合はすべてのチャンクをバッファして JSON として返す。
	Stream *bool `json:"stream,omitempty"`
}

// openaiMessage は OpenAI メッセージオブジェクト。
type openaiMessage struct {
	Role    string `json:"role"`    // "user" | "assistant" | "system"
	Content string `json:"content"` // テキスト内容
}

// ─── ストリーミングレスポンス型 ─────────────────────────────────────

// openaiChunk は SSE ストリーミング時の delta chunk（chat.completion.chunk）。
type openaiChunk struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`  // "chat.completion.chunk"
	Created int64               `json:"created"` // Unix timestamp
	Model   string              `json:"model"`
	Choices []openaiChunkChoice `json:"choices"`
	// EduanimaEvent は eduanimaR 固有の中間イベント（thinking/searching/evidence）。
	// 標準 OpenAI クライアントはこのフィールドを無視する。
	EduanimaEvent *openaiEduanimaEvent `json:"eduanima_event,omitempty"`
}

type openaiChunkChoice struct {
	Index        int         `json:"index"`
	Delta        openaiDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// openaiDelta は delta chunk 内のコンテンツ差分。
// Content が nil の場合はコンテンツなし（中間イベント用チャンク）。
type openaiDelta struct {
	Role    string  `json:"role,omitempty"`
	Content *string `json:"content,omitempty"`
}

// openaiEduanimaEvent は eduanimaR 固有の中間イベント拡張フィールド。
type openaiEduanimaEvent struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// ─── 非ストリーミングレスポンス型 ─────────────────────────────────────

// openaiChatCompletion は非ストリーミング時の完全なレスポンス（chat.completion）。
type openaiChatCompletion struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`  // "chat.completion"
	Created int64               `json:"created"` // Unix timestamp
	Model   string              `json:"model"`
	Choices []openaiComplChoice `json:"choices"`
	Usage   openaiUsage         `json:"usage"`
	// EduanimaSources は eduanimaR 固有のソース情報。
	// 標準 OpenAI クライアントはこのフィールドを無視する。
	EduanimaSources          []openaiSource `json:"eduanima_sources,omitempty"`
	EduanimaCollectedSources []openaiSource `json:"eduanima_collected_sources,omitempty"`
	EduanimaMeta             *openaiMeta    `json:"eduanima_meta,omitempty"`
}

type openaiComplChoice struct {
	Index        int       `json:"index"`
	Message      openaiMsg `json:"message"`
	FinishReason string    `json:"finish_reason"` // "stop"
}

type openaiMsg struct {
	Role    string `json:"role"`    // "assistant"
	Content string `json:"content"` // 完全な回答テキスト
}

// openaiUsage はトークン使用量。professor では Gemini を使用するため 0 を返す。
type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// openaiSource は非ストリーミング時のソース情報（eduanima 拡張）。
type openaiSource struct {
	FileID      string `json:"file_id,omitempty"`
	FileName    string `json:"file_name"`
	PageNumber  *int   `json:"page_number,omitempty"`
	Excerpt     string `json:"excerpt"`
	WhyRelevant string `json:"why_relevant,omitempty"`
}

type openaiMeta struct {
	Answerability      string `json:"answerability,omitempty"`
	IsUnanswerable     bool   `json:"is_unanswerable"`
	CoverageNotes      string `json:"coverage_notes,omitempty"`
	IsPartial          bool   `json:"is_partial"`
	ErrorType          string `json:"error_type,omitempty"`
	EvidenceCount      int    `json:"evidence_count"`
	UnanswerableReason string `json:"unanswerable_reason,omitempty"`
	LoopCount          int    `json:"loop_count,omitempty"`
	LibrarianMS        int    `json:"librarian_ms,omitempty"`
	AnswerGenMS        int    `json:"answer_gen_ms,omitempty"`
}

type qualityLevel struct {
	MaxLoops            int32
	AnswerModelOverride string
	AnswerThinkingLevel string
}

// ─── ChatCompletions ────────────────────────────────────────────────

// ChatCompletions godoc
// @Summary     OpenAI Chat Completions 互換（RAG 質問応答）
// @Description OpenAI SDK 互換のチャットエンドポイント。messages の最後の user メッセージを質問として使用する。
//
//	stream=true（デフォルト）: SSE で OpenAI delta 形式にて回答をストリーミング。
//	stream=false: バッファして chat.completion JSON を返す。
//	eduanima_event 拡張フィールドに thinking/searching/evidence の中間情報を付与。
//
// @Tags        openai-compat
// @Accept      json
// @Produce     text/event-stream,application/json
// @Param       subject_id path     string              true  "Subject UUID"
// @Param       body       body     openaiChatRequest   true  "OpenAI Chat リクエスト"
// @Success     200
// @Failure     400 {object} ErrorBody
// @Failure     404 {object} ErrorBody
// @Router      /api/v1/subjects/{subject_id}/chat/completions [post]
func (h *OpenAIChatHandler) ChatCompletions(c *echo.Context) error {
	rawSubjectID, err := echo.PathParam[string](c, "subject_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid subject_id"})
	}
	subjectID, err := uuid.Parse(rawSubjectID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid subject_id"})
	}

	var req openaiChatRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid request body"})
	}

	// messages から最後の user メッセージを question として取得
	question := extractLastUserMessage(req.Messages)
	if question == "" {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "no user message found in messages"})
	}

	// stream が明示的に false の場合のみ非ストリーミング
	useStream := req.Stream == nil || *req.Stream

	userID := httpmw.GetUserID(c)
	model := req.Model
	if model == "" {
		model = "professor"
	}
	level := resolveQualityLevel(model, h.modelAnswerFast, h.modelAnswerPro, h.maxLoopsFast, h.maxLoopsStd, h.maxLoopsPro, h.thinkingFast, h.thinkingStd, h.thinkingPro)
	chatID := fmt.Sprintf("chatcmpl-%s", uuid.New().String()[:8])
	created := time.Now().Unix()

	if useStream {
		return h.handleStreaming(c, subjectID, userID, question, model, chatID, level, created)
	}
	return h.handleNonStreaming(c, subjectID, userID, question, model, chatID, level, created)
}

// extractLastUserMessage は messages 配列から最後の role="user" のコンテンツを返す。
func extractLastUserMessage(messages []openaiMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	// user メッセージがない場合は最後のメッセージを使用
	if len(messages) > 0 {
		return strings.TrimSpace(messages[len(messages)-1].Content)
	}
	return ""
}

// ─── ストリーミング処理 ─────────────────────────────────────────────

func (h *OpenAIChatHandler) handleStreaming(
	c *echo.Context,
	subjectID, userID uuid.UUID,
	question, model, chatID string,
	level qualityLevel,
	created int64,
) error {
	w := c.Response()
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no")
	c.Response().WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	// SSE 書き込みヘルパー
	writeChunk := func(chunk *openaiChunk) error {
		b, err := json.Marshal(chunk)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	// 最初のチャンク: role アナウンス
	roleContent := ""
	if err := writeChunk(&openaiChunk{
		ID: chatID, Object: "chat.completion.chunk",
		Created: created, Model: model,
		Choices: []openaiChunkChoice{{
			Index:        0,
			Delta:        openaiDelta{Role: "assistant", Content: &roleContent},
			FinishReason: nil,
		}},
	}); err != nil {
		return err
	}

	// ChatUseCase.Ask の onEvent コールバック: eduanimaR → OpenAI delta 変換
	onEvent := func(eventType domain.SSEEventType, data any) error {
		switch eventType {

		// 中間イベント: eduanima_event 拡張フィールドとして付与、content は null
		case domain.SSEEventThinking, domain.SSEEventSearching, domain.SSEEventEvidence:
			return writeChunk(&openaiChunk{
				ID: chatID, Object: "chat.completion.chunk",
				Created: created, Model: model,
				Choices: []openaiChunkChoice{{
					Index: 0, Delta: openaiDelta{}, FinishReason: nil,
				}},
				EduanimaEvent: &openaiEduanimaEvent{
					Type: string(eventType),
					Data: data,
				},
			})

		// 回答テキストチャンク: 標準 OpenAI delta.content として送信
		case domain.SSEEventAnswer:
			text := extractText(data)
			return writeChunk(&openaiChunk{
				ID: chatID, Object: "chat.completion.chunk",
				Created: created, Model: model,
				Choices: []openaiChunkChoice{{
					Index: 0, Delta: openaiDelta{Content: &text}, FinishReason: nil,
				}},
			})

		// 完了: finish_reason="stop" + [DONE] センチネル
		case domain.SSEEventDone:
			stop := "stop"
			if err := writeChunk(&openaiChunk{
				ID: chatID, Object: "chat.completion.chunk",
				Created: created, Model: model,
				Choices: []openaiChunkChoice{{
					Index: 0, Delta: openaiDelta{}, FinishReason: &stop,
				}},
			}); err != nil {
				return err
			}
			_, err := fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			return err

		// エラー: エラーチャンクを送信
		case domain.SSEEventError:
			errMsg := extractErrorMessage(data)
			errChunk := map[string]any{
				"error": map[string]any{"message": errMsg, "type": "server_error"},
			}
			b, _ := json.Marshal(errChunk)
			_, writeErr := fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
			return writeErr
		}
		return nil
	}

	_, ucErr := h.uc.AskWithOptions(c.Request().Context(), subjectID, userID, question, usecases.AskOptions{
		MaxLoops:            level.MaxLoops,
		AnswerModelOverride: level.AnswerModelOverride,
		AnswerThinkingLevel: level.AnswerThinkingLevel,
	}, onEvent)
	if ucErr != nil {
		// エラーは onEvent 内で既に送信済み
		_ = onEvent(domain.SSEEventError, map[string]any{"message": ucErr.Error()})
	}
	return nil
}

// ─── 非ストリーミング処理 ──────────────────────────────────────────

func (h *OpenAIChatHandler) handleNonStreaming(
	c *echo.Context,
	subjectID, userID uuid.UUID,
	question, model, chatID string,
	level qualityLevel,
	created int64,
) error {
	var answerBuf strings.Builder
	var sources []openaiSource
	var collectedSources []openaiSource
	var meta *openaiMeta

	// すべてのイベントをバッファ
	onEvent := func(eventType domain.SSEEventType, data any) error {
		switch eventType {
		case domain.SSEEventAnswer:
			answerBuf.WriteString(extractText(data))
		case domain.SSEEventEvidence:
			src := extractSource(data)
			if src != nil {
				sources = append(sources, *src)
			}
		case domain.SSEEventDone:
			meta = extractMeta(data)
			collectedSources = extractCollectedSources(data)
		}
		// thinking/searching/done/error は非ストリーミング時は収集しない
		return nil
	}

	_, ucErr := h.uc.AskWithOptions(c.Request().Context(), subjectID, userID, question, usecases.AskOptions{
		MaxLoops:            level.MaxLoops,
		AnswerModelOverride: level.AnswerModelOverride,
		AnswerThinkingLevel: level.AnswerThinkingLevel,
	}, onEvent)
	if ucErr != nil {
		return httpError(c, ucErr)
	}

	return c.JSON(http.StatusOK, openaiChatCompletion{
		ID:      chatID,
		Object:  "chat.completion",
		Created: created,
		Model:   model,
		Choices: []openaiComplChoice{{
			Index: 0,
			Message: openaiMsg{
				Role:    "assistant",
				Content: answerBuf.String(),
			},
			FinishReason: "stop",
		}},
		Usage:                    openaiUsage{},
		EduanimaSources:          sources,
		EduanimaCollectedSources: collectedSources,
		EduanimaMeta:             meta,
	})
}

// ─── ヘルパー関数 ──────────────────────────────────────────────────

// extractText は SSEEventAnswer の data から text フィールドを文字列として取得する。
func extractText(data any) string {
	m, ok := data.(map[string]any)
	if !ok {
		return ""
	}
	if v, ok := m["text"].(string); ok {
		return v
	}
	return ""
}

// extractErrorMessage は SSEEventError の data からエラーメッセージを取得する。
func extractErrorMessage(data any) string {
	m, ok := data.(map[string]any)
	if !ok {
		return "internal server error"
	}
	if v, ok := m["message"].(string); ok {
		return v
	}
	return "internal server error"
}

// extractSource は SSEEventEvidence の data から openaiSource を構築する。
func extractSource(data any) *openaiSource {
	m, ok := data.(map[string]any)
	if !ok {
		return nil
	}
	return sourceFromMap(m)
}

func sourceFromMap(m map[string]any) *openaiSource {
	src := &openaiSource{}
	if v, ok := m["file_name"].(string); ok {
		src.FileName = v
	}
	if v, ok := m["file_id"].(string); ok {
		src.FileID = v
	}
	if v, ok := m["page_number"].(*int); ok {
		src.PageNumber = v
	}
	if v, ok := m["page_number"].(int); ok {
		src.PageNumber = &v
	}
	if v, ok := m["page_number"].(float64); ok {
		iv := int(v)
		src.PageNumber = &iv
	}
	if v, ok := m["excerpt"].(string); ok {
		src.Excerpt = v
	}
	if v, ok := m["why_relevant"].(string); ok {
		src.WhyRelevant = v
	}
	return src
}

func extractMeta(data any) *openaiMeta {
	m, ok := data.(map[string]any)
	if !ok {
		return nil
	}
	meta := &openaiMeta{}
	if v, ok := m["answerability"].(string); ok {
		meta.Answerability = v
	}
	if v, ok := m["is_unanswerable"].(bool); ok {
		meta.IsUnanswerable = v
	}
	if v, ok := m["coverage_notes"].(string); ok {
		meta.CoverageNotes = v
	}
	if v, ok := m["is_partial"].(bool); ok {
		meta.IsPartial = v
	}
	if v, ok := m["error_type"].(string); ok {
		meta.ErrorType = v
	}
	if v, ok := m["evidence_count"].(int); ok {
		meta.EvidenceCount = v
	}
	if v, ok := m["evidence_count"].(float64); ok {
		meta.EvidenceCount = int(v)
	}
	if v, ok := m["unanswerable_reason"].(string); ok {
		meta.UnanswerableReason = v
	}
	if v, ok := m["loop_count"].(int); ok {
		meta.LoopCount = v
	}
	if v, ok := m["loop_count"].(float64); ok {
		meta.LoopCount = int(v)
	}
	if v, ok := m["librarian_ms"].(int); ok {
		meta.LibrarianMS = v
	}
	if v, ok := m["librarian_ms"].(float64); ok {
		meta.LibrarianMS = int(v)
	}
	if v, ok := m["answer_gen_ms"].(int); ok {
		meta.AnswerGenMS = v
	}
	if v, ok := m["answer_gen_ms"].(float64); ok {
		meta.AnswerGenMS = int(v)
	}
	return meta
}

func resolveQualityLevel(model string, modelAnswerFast string, modelAnswerPro string, maxLoopsFast int32, maxLoopsStd int32, maxLoopsPro int32, thinkingFast string, thinkingStd string, thinkingPro string) qualityLevel {
	switch strings.TrimSpace(model) {
	case "professor-fast", "professor-lite":
		return qualityLevel{MaxLoops: maxLoopsFast, AnswerModelOverride: modelAnswerFast, AnswerThinkingLevel: thinkingFast}
	case "professor-pro":
		return qualityLevel{MaxLoops: maxLoopsPro, AnswerModelOverride: strings.TrimSpace(modelAnswerPro), AnswerThinkingLevel: thinkingPro}
	default:
		return qualityLevel{MaxLoops: maxLoopsStd, AnswerThinkingLevel: thinkingStd}
	}
}

func parseInt32Env(key string, def int32) int32 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return int32(n)
}

func parseStringEnv(key string, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func extractCollectedSources(data any) []openaiSource {
	m, ok := data.(map[string]any)
	if !ok {
		return nil
	}
	raw, ok := m["collected_sources"].([]any)
	if !ok {
		return nil
	}
	out := make([]openaiSource, 0, len(raw))
	for _, item := range raw {
		sm, ok := item.(map[string]any)
		if !ok {
			continue
		}
		src := sourceFromMap(sm)
		if src == nil || src.FileName == "" {
			continue
		}
		out = append(out, *src)
	}
	return out
}
