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

// MaterialHandler は教材（ファイル）REST ハンドラー
type MaterialHandler struct {
	uc *usecases.MaterialUseCase
}

// NewMaterialHandler は MaterialHandler を生成する。
func NewMaterialHandler(uc *usecases.MaterialUseCase) *MaterialHandler {
	return &MaterialHandler{uc: uc}
}

// Register は Echo グループにルートを登録する。
// ルートプレフィックス: /api/v1/subjects/:subject_id/materials
func (h *MaterialHandler) Register(g *echo.Group) {
	g.GET("", h.List)
	g.POST("", h.Upload)
	g.DELETE("/:fid", h.Delete)
}

// ─── レスポンス型 ──────────────────────────────────────────

type materialResponse struct {
	ID          string  `json:"id"`
	SubjectID   string  `json:"subject_id"`
	Name        string  `json:"name"`
	MimeType    string  `json:"mime_type"`
	SizeBytes   int64   `json:"size_bytes"`
	Status      string  `json:"status"`
	ErrorMsg    *string `json:"error_message,omitempty"`
	UploadedAt  string  `json:"uploaded_at"`
	ProcessedAt *string `json:"processed_at,omitempty"`
}

func toMaterialResp(f *domain.File) materialResponse {
	r := materialResponse{
		ID:         f.ID.String(),
		SubjectID:  f.SubjectID.String(),
		Name:       f.Name,
		MimeType:   f.MimeType,
		SizeBytes:  f.SizeBytes,
		Status:     string(f.Status),
		ErrorMsg:   f.ErrorMessage,
		UploadedAt: f.UploadedAt.Format(time.RFC3339),
	}
	if f.ProcessedAt != nil {
		s := f.ProcessedAt.Format(time.RFC3339)
		r.ProcessedAt = &s
	}
	return r
}

// ─── ハンドラー ────────────────────────────────────────────

// List godoc
// @Summary 教材一覧取得
// @Tags materials
// @Produce json
// @Param subject_id path string true "Subject ID"
// @Success 200 {array} materialResponse
// @Router /api/v1/subjects/{subject_id}/materials [get]
func (h *MaterialHandler) List(c echo.Context) error {
	subjectID, err := uuid.Parse(c.Param("subject_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid subject id"})
	}
	userID := httpmw.GetUserID(c)
	files, err := h.uc.ListBySubject(c.Request().Context(), subjectID, userID)
	if err != nil {
		return httpError(c, err)
	}
	out := make([]materialResponse, 0, len(files))
	for _, f := range files {
		out = append(out, toMaterialResp(f))
	}
	return c.JSON(http.StatusOK, out)
}

// Upload godoc
// @Summary 教材アップロード
// @Tags materials
// @Accept multipart/form-data
// @Produce json
// @Param subject_id path string true "Subject ID"
// @Param file formData file true "File to upload"
// @Success 201 {object} materialResponse
// @Router /api/v1/subjects/{subject_id}/materials [post]
func (h *MaterialHandler) Upload(c echo.Context) error {
	subjectID, err := uuid.Parse(c.Param("subject_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid subject id"})
	}

	fh, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "file is required"})
	}

	src, err := fh.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorBody{Error: "failed to open file"})
	}
	defer src.Close()

	// Content-Type を判定（フォームのヘッダーを優先）
	mimeType := fh.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	userID := httpmw.GetUserID(c)
	file, err := h.uc.Upload(c.Request().Context(), usecases.UploadMaterialInput{
		SubjectID: subjectID,
		UserID:    userID,
		FileName:  fh.Filename,
		MimeType:  mimeType,
		Size:      fh.Size,
		Reader:    src,
	})
	if err != nil {
		return httpError(c, err)
	}
	return c.JSON(http.StatusCreated, toMaterialResp(file))
}

// Delete godoc
// @Summary 教材削除
// @Tags materials
// @Param subject_id path string true "Subject ID"
// @Param fid path string true "File ID"
// @Success 204
// @Router /api/v1/subjects/{subject_id}/materials/{fid} [delete]
func (h *MaterialHandler) Delete(c echo.Context) error {
	fileID, err := uuid.Parse(c.Param("fid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid file id"})
	}
	userID := httpmw.GetUserID(c)
	if err := h.uc.Delete(c.Request().Context(), fileID, userID); err != nil {
		return httpError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
