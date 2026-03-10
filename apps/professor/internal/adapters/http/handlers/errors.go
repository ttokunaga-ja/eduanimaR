package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
)

// ErrorBody はエラーレスポンスの共通形式
type ErrorBody struct {
	Error string `json:"error"`
}

// httpError はドメインエラーを HTTP ステータスコードに変換して返す。
func httpError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		slog.Info("http error", "status", http.StatusNotFound, "error", err)
		return c.JSON(http.StatusNotFound, ErrorBody{Error: err.Error()})
	case errors.Is(err, domain.ErrForbidden):
		slog.Info("http error", "status", http.StatusForbidden, "error", err)
		return c.JSON(http.StatusForbidden, ErrorBody{Error: err.Error()})
	case errors.Is(err, domain.ErrInvalidInput):
		slog.Info("http error", "status", http.StatusBadRequest, "error", err)
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: err.Error()})
	case errors.Is(err, domain.ErrConflict):
		slog.Info("http error", "status", http.StatusConflict, "error", err)
		return c.JSON(http.StatusConflict, ErrorBody{Error: err.Error()})
	default:
		slog.Error("http internal error", "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorBody{Error: "internal server error"})
	}
}
