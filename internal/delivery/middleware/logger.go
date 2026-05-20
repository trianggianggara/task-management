package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
)

var requestLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

func Logger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			requestID := GetRequestID(c)

			err := next(c)

			latencyMs := time.Since(start).Milliseconds()
			status := c.Response().Status
			if err != nil && status == http.StatusOK {
				if he, ok := err.(*echo.HTTPError); ok {
					status = he.Code
				} else if ae, ok := err.(interface{ HTTPStatus() int }); ok {
					status = ae.HTTPStatus()
				}
			}
			level := levelFromStatus(status)

			requestLogger.LogAttrs(c.Request().Context(), level, "request",
				slog.String("request_id", requestID),
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.Int("status", status),
				slog.Int64("latency_ms", latencyMs),
			)

			return err
		}
	}
}

func levelFromStatus(status int) slog.Level {
	if status >= 500 {
		return slog.LevelError
	}
	if status >= 400 {
		return slog.LevelWarn
	}
	return slog.LevelInfo
}

func LevelFromStatus(status int) slog.Level {
	return levelFromStatus(status)
}
