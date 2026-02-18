package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
)

// ErrorBody はエラーレスポンスの共通形式
type ErrorBody struct {
	Error string `json:"error"`
}

// httpError はドメインエラーを HTTP ステータスコードに変換して返す。
func httpError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return c.JSON(http.StatusNotFound, ErrorBody{Error: err.Error()})
	case errors.Is(err, domain.ErrForbidden):
		return c.JSON(http.StatusForbidden, ErrorBody{Error: err.Error()})
	case errors.Is(err, domain.ErrInvalidInput):
		return c.JSON(http.StatusBadRequest, ErrorBody{Error: err.Error()})
	case errors.Is(err, domain.ErrConflict):
		return c.JSON(http.StatusConflict, ErrorBody{Error: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, ErrorBody{Error: "internal server error"})
	}
}
