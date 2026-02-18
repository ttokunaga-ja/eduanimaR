package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	httpmw "github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/adapters/http/middleware"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/usecases"
)

// SubjectHandler は科目 REST ハンドラー
type SubjectHandler struct {
	uc *usecases.SubjectUseCase
}

// NewSubjectHandler は SubjectHandler を生成する。
func NewSubjectHandler(uc *usecases.SubjectUseCase) *SubjectHandler {
	return &SubjectHandler{uc: uc}
}

// Register は Echo グループにルートを登録する。
func (h *SubjectHandler) Register(g *echo.Group) {
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/:id", h.Get)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

// ─── レスポンス型 ──────────────────────────────────────────

type subjectResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	LMSCourseID *string `json:"lms_course_id,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func toSubjectResp(s *domain.Subject) subjectResponse {
	return subjectResponse{
		ID:          s.ID.String(),
		Name:        s.Name,
		LMSCourseID: s.LMSCourseID,
		CreatedAt:   s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   s.UpdatedAt.Format(time.RFC3339),
	}
}

// ─── ハンドラー ────────────────────────────────────────────

// List godoc
// @Summary 科目一覧取得
// @Tags subjects
// @Produce json
// @Success 200 {array} subjectResponse
// @Router /api/v1/subjects [get]
func (h *SubjectHandler) List(c echo.Context) error {
	userID := httpmw.GetUserID(c)
	subjects, err := h.uc.ListByUser(c.Request().Context(), userID)
	if err != nil {
		return httpError(c, err)
	}
	out := make([]subjectResponse, 0, len(subjects))
	for _, s := range subjects {
		out = append(out, toSubjectResp(s))
	}
	return c.JSON(http.StatusOK, out)
}

// Get godoc
// @Summary 科目取得
// @Tags subjects
// @Produce json
// @Param id path string true "Subject ID"
// @Success 200 {object} subjectResponse
// @Router /api/v1/subjects/{id} [get]
func (h *SubjectHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid subject id"})
	}
	userID := httpmw.GetUserID(c)
	s, err := h.uc.GetByIDAndUser(c.Request().Context(), id, userID)
	if err != nil {
		return httpError(c, err)
	}
	return c.JSON(http.StatusOK, toSubjectResp(s))
}

// Create godoc
// @Summary 科目作成
// @Tags subjects
// @Accept json
// @Produce json
// @Param body body createSubjectRequest true "Request body"
// @Success 201 {object} subjectResponse
// @Router /api/v1/subjects [post]
func (h *SubjectHandler) Create(c echo.Context) error {
	var req struct {
		Name        string  `json:"name"`
		LMSCourseID *string `json:"lms_course_id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid request body"})
	}
	userID := httpmw.GetUserID(c)
	s, err := h.uc.Create(c.Request().Context(), userID, usecases.CreateSubjectInput{
		Name:        req.Name,
		LMSCourseID: req.LMSCourseID,
	})
	if err != nil {
		return httpError(c, err)
	}
	return c.JSON(http.StatusCreated, toSubjectResp(s))
}

// Update godoc
// @Summary 科目名更新
// @Tags subjects
// @Accept json
// @Produce json
// @Param id path string true "Subject ID"
// @Success 200 {object} subjectResponse
// @Router /api/v1/subjects/{id} [put]
func (h *SubjectHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid subject id"})
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid request body"})
	}
	userID := httpmw.GetUserID(c)
	s, err := h.uc.UpdateName(c.Request().Context(), id, userID, req.Name)
	if err != nil {
		return httpError(c, err)
	}
	return c.JSON(http.StatusOK, toSubjectResp(s))
}

// Delete godoc
// @Summary 科目削除
// @Tags subjects
// @Param id path string true "Subject ID"
// @Success 204
// @Router /api/v1/subjects/{id} [delete]
func (h *SubjectHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid subject id"})
	}
	userID := httpmw.GetUserID(c)
	if err := h.uc.Delete(c.Request().Context(), id, userID); err != nil {
		return httpError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
