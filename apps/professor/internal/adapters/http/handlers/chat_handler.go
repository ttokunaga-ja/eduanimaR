package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	httpmw "github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/adapters/http/middleware"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/usecases"
)

// ChatHandler は質問応答セッションの HTTP ハンドラー。
// POST /api/v1/subjects/:subject_id/chats     → SSE ストリーミング回答（Ask）
// GET  /api/v1/subjects/:subject_id/chats     → セッション一覧（ListSessions）
// POST /api/v1/subjects/:subject_id/chats/:session_id/feedback → フィードバック記録
type ChatHandler struct {
	uc *usecases.ChatUseCase
}

// NewChatHandler は ChatHandler を生成する。
func NewChatHandler(uc *usecases.ChatUseCase) *ChatHandler {
	return &ChatHandler{uc: uc}
}

// Register は Echo グループにルートを登録する。
func (h *ChatHandler) Register(g *echo.Group) {
	g.POST("", h.Ask)
	g.GET("", h.ListSessions)
	g.POST("/:session_id/feedback", h.Feedback)
}

// ─── Ask (SSE) ────────────────────────────────────────────────────

// askRequest は POST /chats のリクエストボディ。
type askRequest struct {
	Question string `json:"question"`
}

// Ask godoc
// @Summary     質問応答（SSE ストリーミング）
// @Description Librarian を使った RAG パイプラインを実行し、SSE で回答をストリーミングする
// @Tags        chats
// @Accept      json
// @Produce     text/event-stream
// @Param       subject_id path     string     true "Subject UUID"
// @Param       body       body     askRequest true "質問"
// @Success     200
// @Failure     400 {object} ErrorBody
// @Failure     404 {object} ErrorBody
// @Router      /api/v1/subjects/{subject_id}/chats [post]
func (h *ChatHandler) Ask(c echo.Context) error {
	subjectID, err := uuid.Parse(c.Param("subject_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid subject_id"})
	}

	var req askRequest
	if err := c.Bind(&req); err != nil || req.Question == "" {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "question is required"})
	}

	userID := httpmw.GetUserID(c)

	// ─── SSE ヘッダー設定 ───────────────────────────────────────
	w := c.Response().Writer
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no") // nginx バッファリング無効化
	c.Response().WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported by the response writer")
	}

	// ─── SSE 書き込みヘルパー ────────────────────────────────────
	writeEvent := func(eventType domain.SSEEventType, data any) error {
		payload := map[string]any{
			"type": string(eventType),
			"data": data,
		}
		b, jsonErr := json.Marshal(payload)
		if jsonErr != nil {
			return jsonErr
		}
		if _, writeErr := fmt.Fprintf(w, "data: %s\n\n", b); writeErr != nil {
			return writeErr
		}
		flusher.Flush()
		return nil
	}

	// ─── ユースケース呼び出し ────────────────────────────────────
	_, ucErr := h.uc.Ask(c.Request().Context(), subjectID, userID, req.Question, writeEvent)
	if ucErr != nil {
		// SSEEventError は usecase 内で既に送信試行済みだが念のため再送
		_ = writeEvent(domain.SSEEventError, map[string]any{"message": ucErr.Error()})
	}

	return nil
}

// ─── ListSessions ─────────────────────────────────────────────────

// qaSessionResponse は QASession の JSON 表現。
type qaSessionResponse struct {
	ID         string          `json:"id"`
	Question   string          `json:"question"`
	Answer     *string         `json:"answer,omitempty"`
	Sources    []domain.Source `json:"sources,omitempty"`
	Feedback   *int            `json:"feedback,omitempty"`
	CreatedAt  string          `json:"created_at"`
	AnsweredAt *string         `json:"answered_at,omitempty"`
}

// listSessionsResponse はセッション一覧レスポンス。
type listSessionsResponse struct {
	Sessions []qaSessionResponse `json:"sessions"`
	Total    int64               `json:"total"`
	Limit    int                 `json:"limit"`
	Offset   int                 `json:"offset"`
}

func toQASessionResp(s *domain.QASession) qaSessionResponse {
	r := qaSessionResponse{
		ID:        s.ID.String(),
		Question:  s.Question,
		Answer:    s.Answer,
		Sources:   s.Sources,
		Feedback:  s.Feedback,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
	}
	if s.AnsweredAt != nil {
		t := s.AnsweredAt.Format(time.RFC3339)
		r.AnsweredAt = &t
	}
	return r
}

// ListSessions godoc
// @Summary     質問応答セッション一覧
// @Tags        chats
// @Produce     json
// @Param       subject_id path  string true  "Subject UUID"
// @Param       limit      query int    false "件数（デフォルト20）"
// @Param       offset     query int    false "オフセット（デフォルト0）"
// @Success     200 {object} listSessionsResponse
// @Router      /api/v1/subjects/{subject_id}/chats [get]
func (h *ChatHandler) ListSessions(c echo.Context) error {
	subjectID, err := uuid.Parse(c.Param("subject_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid subject_id"})
	}

	limit := 20
	offset := 0
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	userID := httpmw.GetUserID(c)

	sessions, err := h.uc.ListSessions(c.Request().Context(), subjectID, userID, limit, offset)
	if err != nil {
		return httpError(c, err)
	}

	total, err := h.uc.CountSessions(c.Request().Context(), subjectID, userID)
	if err != nil {
		return httpError(c, err)
	}

	out := make([]qaSessionResponse, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, toQASessionResp(s))
	}

	return c.JSON(http.StatusOK, listSessionsResponse{
		Sessions: out,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	})
}

// ─── Feedback ─────────────────────────────────────────────────────

// feedbackRequest は POST /chats/:session_id/feedback のリクエストボディ。
type feedbackRequest struct {
	Feedback int `json:"feedback"` // 1: good / -1: bad
}

// Feedback godoc
// @Summary     フィードバック送信
// @Tags        chats
// @Accept      json
// @Produce     json
// @Param       subject_id path  string          true "Subject UUID"
// @Param       session_id path  string          true "Session UUID"
// @Param       body       body  feedbackRequest true "フィードバック"
// @Success     200 {object} qaSessionResponse
// @Router      /api/v1/subjects/{subject_id}/chats/{session_id}/feedback [post]
func (h *ChatHandler) Feedback(c echo.Context) error {
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid session_id"})
	}

	var req feedbackRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid request body"})
	}
	if req.Feedback != 1 && req.Feedback != -1 {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "feedback must be 1 (good) or -1 (bad)"})
	}

	userID := httpmw.GetUserID(c)

	session, err := h.uc.UpdateFeedback(c.Request().Context(), sessionID, userID, req.Feedback)
	if err != nil {
		return httpError(c, err)
	}

	return c.JSON(http.StatusOK, toQASessionResp(session))
}
