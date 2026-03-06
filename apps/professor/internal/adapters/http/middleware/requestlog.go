package middleware

import (
	"log/slog"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// RequestLog は request_id / trace_id / user_id を共通キーとしてアクセスログを出力する。
func RequestLog() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			res := c.Response()

			requestID := res.Header().Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = req.Header.Get(echo.HeaderXRequestID)
			}
			traceID := extractTraceID(req.Header.Get("X-Cloud-Trace-Context"), requestID)
			if traceID != "" {
				res.Header().Set("X-Trace-ID", traceID)
			}

			err := next(c)

			userID := GetUserID(c).String()
			attrs := []any{
				"request_id", requestID,
				"trace_id", traceID,
				"user_id", userID,
				"method", req.Method,
				"path", req.URL.Path,
				"status", res.Status,
				"latency_ms", time.Since(start).Milliseconds(),
			}

			if err != nil {
				slog.Error("http request failed", append(attrs, "error", err.Error())...)
			} else {
				slog.Info("http request", attrs...)
			}

			return err
		}
	}
}

func extractTraceID(cloudTraceContext, fallback string) string {
	if cloudTraceContext == "" {
		return fallback
	}

	parts := strings.SplitN(cloudTraceContext, "/", 2)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return fallback
}
