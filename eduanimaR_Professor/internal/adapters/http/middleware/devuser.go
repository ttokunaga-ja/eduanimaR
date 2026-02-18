// Package middleware は Echo 用ミドルウェアを提供する。
package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
)

// ctxKeyUserID はコンテキストへのユーザーID格納キー
const ctxKeyUserID = "userID"

// DevUser は Phase 1 用の固定ユーザーミドルウェア。
// X-Dev-User ヘッダーを無視し、常に固定の dev user を設定する。
// Phase 2 では JWT 検証ミドルウェアに差し替える。
func DevUser() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ctxKeyUserID, domain.DevUserID)
			return next(c)
		}
	}
}

// GetUserID は Echo コンテキストからユーザー ID を取得する。
// DevUser ミドルウェアが設定していない場合は DevUserID をフォールバックとして返す。
func GetUserID(c echo.Context) uuid.UUID {
	if v, ok := c.Get(ctxKeyUserID).(uuid.UUID); ok {
		return v
	}
	return domain.DevUserID
}
