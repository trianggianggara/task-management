package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func BodyLimit(maxBytes int64) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Request().Body = http.MaxBytesReader(
				c.Response().Writer,
				c.Request().Body,
				maxBytes,
			)
			return next(c)
		}
	}
}
