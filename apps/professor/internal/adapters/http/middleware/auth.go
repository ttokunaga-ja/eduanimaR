package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"google.golang.org/api/idtoken"
)

// jwtUIDNamespace は Firebase UID (文字列) を決定論的な UUID v5 に変換するための
// 固定ネームスペース。同じ Firebase UID は常に同じ UUID にマップされる。
// TODO(Phase 3): ドメイン層が string UID に移行した際にこの変換を削除する。
var jwtUIDNamespace = uuid.MustParse("6ba7b814-9dad-11d1-80b4-00c04fd430c8")

// RequireJWT は Google Identity Platform (Firebase Auth) の ID トークンを検証する
// 本番用認証ミドルウェア。
//
// フロー:
//  1. Authorization: Bearer <token> ヘッダーからトークンを取得
//  2. Google 公開鍵 (JWKS) で RS256 署名を検証 (audience = Firebase プロジェクト ID)
//  3. Firebase UID (sub クレーム) を UUID v5 に変換して ctxKeyUserID にセット
//
// トークンが欠如・無効な場合は HTTP 401 を返し、次のハンドラーは呼ばれない。
func RequireJWT(audience string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "missing_token",
					"message": "Authorization: Bearer <token> header is required",
				})
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")

			payload, err := idtoken.Validate(context.Background(), token, audience)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "invalid_token",
					"message": "JWT validation failed",
				})
			}

			// Firebase UID (sub) を UUID v5 に変換して既存ハンドラーと互換性を保つ。
			// Phase 3 でドメイン層が string UID に移行するまでの暫定措置。
			userUUID := uuid.NewSHA1(jwtUIDNamespace, []byte(payload.Subject))
			c.Set(ctxKeyUserID, userUUID)

			return next(c)
		}
	}
}
