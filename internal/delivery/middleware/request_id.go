package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const RequestIDKey = "request_id"

func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get("X-Request-ID")
			if id == "" {
				id = uuid.New().String()
			}
			c.Set(RequestIDKey, id)
			c.Response().Header().Set("X-Request-ID", id)
			return next(c)
		}
	}
}

func GetRequestID(c echo.Context) string {
	id, _ := c.Get(RequestIDKey).(string)
	return id
}
