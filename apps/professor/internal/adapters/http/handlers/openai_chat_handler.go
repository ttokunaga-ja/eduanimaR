package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	uc *usecases.ChatUseCase
}

// NewOpenAIChatHandler は OpenAIChatHandler を生成する。
func NewOpenAIChatHandler(uc *usecases.ChatUseCase) *OpenAIChatHandler {
	return &OpenAIChatHandler{uc: uc}
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
	EduanimaSources []openaiSource `json:"eduanima_sources,omitempty"`
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
	FileName    string `json:"file_name"`
	Excerpt     string `json:"excerpt"`
	WhyRelevant string `json:"why_relevant,omitempty"`
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
	chatID := fmt.Sprintf("chatcmpl-%s", uuid.New().String()[:8])
	created := time.Now().Unix()

	if useStream {
		return h.handleStreaming(c, subjectID, userID, question, model, chatID, created)
	}
	return h.handleNonStreaming(c, subjectID, userID, question, model, chatID, created)
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

	_, ucErr := h.uc.Ask(c.Request().Context(), subjectID, userID, question, onEvent)
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
	created int64,
) error {
	var answerBuf strings.Builder
	var sources []openaiSource

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
		}
		// thinking/searching/done/error は非ストリーミング時は収集しない
		return nil
	}

	_, ucErr := h.uc.Ask(c.Request().Context(), subjectID, userID, question, onEvent)
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
		Usage:           openaiUsage{},
		EduanimaSources: sources,
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
	src := &openaiSource{}
	if v, ok := m["file_name"].(string); ok {
		src.FileName = v
	}
	if v, ok := m["excerpt"].(string); ok {
		src.Excerpt = v
	}
	if v, ok := m["why_relevant"].(string); ok {
		src.WhyRelevant = v
	}
	return src
}
