package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// RequireAuth は Phase 2 で JWT 検証に差し替えるまでの本番用認証プレースホルダー。
// APP_ENV=production のときのみ登録され、すべてのリクエストに 401 を返す。
// Phase 2 で実際の IdP 連携（Google Identity など）に置き換えること。
func RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error":   "authentication_required",
				"message": "JWT authentication is not yet implemented for production. Deploy with APP_ENV=development for testing, or implement JWT middleware (Phase 2).",
			})
		}
	}
}
