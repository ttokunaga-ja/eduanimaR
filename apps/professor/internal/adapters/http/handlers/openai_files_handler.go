package handlers

import (
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	httpmw "github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/adapters/http/middleware"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/usecases"
)

// OpenAIFilesHandler は OpenAI Files API 互換エンドポイントを提供する。
//
// クライアント設定例:
//
//	client = OpenAI(
//	    base_url="https://host/api/v1/subjects/{subject_id}",
//	    api_key="<Firebase ID Token>"
//	)
//	# ファイルアップロード
//	with open("document.pdf", "rb") as f:
//	    client.files.create(file=f, purpose="assistants")
//	# ステータス確認
//	client.files.retrieve("file-550e8400e29b41d4a716446655440000")
type OpenAIFilesHandler struct {
	uc *usecases.MaterialUseCase
}

// NewOpenAIFilesHandler は OpenAIFilesHandler を生成する。
func NewOpenAIFilesHandler(uc *usecases.MaterialUseCase) *OpenAIFilesHandler {
	return &OpenAIFilesHandler{uc: uc}
}

// Register は Echo グループにルートを登録する。
func (h *OpenAIFilesHandler) Register(g *echo.Group) {
	g.POST("", h.Upload)
	g.GET("/:file_id", h.GetFile)
}

// ─── OpenAI File オブジェクト型 ─────────────────────────────────────

// openaiFileObject は OpenAI Files API の File オブジェクト。
// 参考: https://platform.openai.com/docs/api-reference/files/object
type openaiFileObject struct {
	// ID は "file-" プレフィックス + UUID (ハイフンなし) 形式。
	// 例: "file-550e8400e29b41d4a716446655440000"
	ID       string `json:"id"`
	Object   string `json:"object"`   // 常に "file"
	Bytes    int64  `json:"bytes"`    // ファイルサイズ (bytes)
	Created  int64  `json:"created"`  // Unix タイムスタンプ
	Filename string `json:"filename"` // 元のファイル名
	Purpose  string `json:"purpose"`  // アップロード時の purpose フィールド（pass-through）
	// Status は OpenAI Files API 互換の処理状態。
	//   "uploaded"  → pending / processing（まだ処理中）
	//   "processed" → ready（検索可能）
	//   "error"     → failed（処理失敗）
	Status string `json:"status"`
	// StatusDetails は status="error" 時のエラー詳細。
	StatusDetails *string `json:"status_details,omitempty"`
}

// ─── ステータス / ID 変換ヘルパー ─────────────────────────────────────

// toOpenAIStatus は domain.FileStatus を OpenAI Files API 互換のステータス文字列に変換する。
func toOpenAIStatus(s domain.FileStatus) string {
	switch s {
	case domain.FileStatusCompleted:
		return "processed"
	case domain.FileStatusFailed:
		return "error"
	default:
		// pending, processing → まだ処理中
		return "uploaded"
	}
}

// toOpenAIFileID は UUID を OpenAI 互換の "file-{hex32}" 形式に変換する。
func toOpenAIFileID(id uuid.UUID) string {
	// UUID のハイフンを除去して file- プレフィックスを付与
	return "file-" + strings.ReplaceAll(id.String(), "-", "")
}

// parseOpenAIFileID は OpenAI 互換 ID または通常の UUID 文字列を uuid.UUID に変換する。
// 受け付けるフォーマット:
//   - "file-550e8400e29b41d4a716446655440000"  (OpenAI 互換形式)
//   - "550e8400-e29b-41d4-a716-446655440000"   (標準 UUID)
//   - "550e8400e29b41d4a716446655440000"        (ハイフンなし hex)
func parseOpenAIFileID(fileID string) (uuid.UUID, error) {
	// "file-" プレフィックスを除去
	raw := strings.TrimPrefix(fileID, "file-")

	// 標準 UUID 形式（36文字: 8-4-4-4-12）
	if len(raw) == 36 {
		return uuid.Parse(raw)
	}

	// ハイフンなし hex 形式（32文字）
	if len(raw) == 32 {
		formatted := fmt.Sprintf("%s-%s-%s-%s-%s",
			raw[0:8], raw[8:12], raw[12:16], raw[16:20], raw[20:32],
		)
		return uuid.Parse(formatted)
	}

	return uuid.UUID{}, fmt.Errorf("invalid file ID format: %q", fileID)
}

// toOpenAIFileObject は domain.File を openaiFileObject に変換する。
func toOpenAIFileObject(f *domain.File, purpose string) openaiFileObject {
	obj := openaiFileObject{
		ID:       toOpenAIFileID(f.ID),
		Object:   "file",
		Bytes:    f.SizeBytes,
		Created:  f.UploadedAt.Unix(),
		Filename: f.Name,
		Purpose:  purpose,
		Status:   toOpenAIStatus(f.Status),
	}
	if f.Status == domain.FileStatusFailed && f.ErrorMessage != nil {
		obj.StatusDetails = f.ErrorMessage
	}
	return obj
}

// ─── Upload (POST .../files) ─────────────────────────────────────────

// Upload godoc
// @Summary     OpenAI Files 互換アップロード
// @Description OpenAI Files API 互換のファイルアップロード。
//
//	purpose フィールドはパススルー（professor では無視される）。
//	アップロード後は GET /files/:file_id で処理状態をポーリングする。
//	status が "processed" になれば chat/completions で使用可能。
//
// @Tags        openai-compat
// @Accept      multipart/form-data
// @Produce     json
// @Param       subject_id path     string true  "Subject UUID"
// @Param       file       formData file   true  "アップロードするファイル"
// @Param       purpose    formData string false "purpose (OpenAI 互換フィールド、無視される)"
// @Success     200 {object} openaiFileObject
// @Failure     400 {object} ErrorBody
// @Failure     404 {object} ErrorBody
// @Router      /api/v1/subjects/{subject_id}/files [post]
func (h *OpenAIFilesHandler) Upload(c *echo.Context) error {
	rawSubjectID, err := echo.PathParam[string](c, "subject_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid subject_id"})
	}
	subjectID, err := uuid.Parse(rawSubjectID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid subject_id"})
	}

	fh, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "file is required"})
	}

	src, err := fh.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorBody{Error: "failed to open file"})
	}
	defer func() {
		if err := src.Close(); err != nil {
			slog.Error("failed to close uploaded file stream", "error", err)
		}
	}()

	// purpose は OpenAI 互換フィールドとして受け取るが professor では無視する
	purpose := c.FormValue("purpose")
	if purpose == "" {
		purpose = "assistants"
	}

	mimeType := fh.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		if inferred := mime.TypeByExtension(strings.ToLower(filepath.Ext(fh.Filename))); inferred != "" {
			mimeType = inferred
		}
	}
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

	return c.JSON(http.StatusOK, toOpenAIFileObject(file, purpose))
}

// ─── GetFile (GET .../files/:file_id) ───────────────────────────────

// GetFile godoc
// @Summary     OpenAI File オブジェクト取得（ステータス確認）
// @Description OpenAI Files API 互換のファイルステータス確認エンドポイント。
//
//	status が "processed" になれば chat/completions で使用可能。
//	file_id は "file-{hex32}" 形式または通常の UUID 形式を受け付ける。
//
// @Tags        openai-compat
// @Produce     json
// @Param       subject_id path     string true "Subject UUID"
// @Param       file_id    path     string true "File ID（OpenAI 形式または UUID）"
// @Success     200 {object} openaiFileObject
// @Failure     400 {object} ErrorBody
// @Failure     404 {object} ErrorBody
// @Router      /api/v1/subjects/{subject_id}/files/{file_id} [get]
func (h *OpenAIFilesHandler) GetFile(c *echo.Context) error {
	rawSubjectID, err := echo.PathParam[string](c, "subject_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid subject_id"})
	}
	subjectID, err := uuid.Parse(rawSubjectID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid subject_id"})
	}

	rawFileID, err := echo.PathParam[string](c, "file_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid file_id format"})
	}
	fileID, err := parseOpenAIFileID(rawFileID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: "invalid file_id format"})
	}

	userID := httpmw.GetUserID(c)
	file, err := h.uc.GetFile(c.Request().Context(), subjectID, fileID, userID)
	if err != nil {
		return httpError(c, err)
	}

	// purpose は保存されていないため固定値を返す
	return c.JSON(http.StatusOK, toOpenAIFileObject(file, "assistants"))
}
