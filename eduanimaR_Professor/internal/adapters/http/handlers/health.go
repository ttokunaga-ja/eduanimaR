// Package handlers は Echo HTTP ハンドラーを提供する。
package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HealthResponse はヘルスチェックのレスポンス型。
type HealthResponse struct {
	Status string `json:"status"`
}

// Healthz は GET /healthz ハンドラー。
// ロードバランサーやコンテナオーケストレーターのヘルスチェックに使用する。
func Healthz(c echo.Context) error {
	return c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}
