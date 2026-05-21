package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/labstack/echo/v4"
	"task-management/pkg/utils/response"
)

func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	var appErr *response.APIError

	switch e := err.(type) {
	case *response.APIError:
		appErr = e
	case *echo.HTTPError:
		appErr = httpErrorToAppError(e)
	default:
		appErr = response.Internal("internal server error", err)
	}

	if appErr.Status >= 500 {
		requestID := GetRequestID(c)
		slog.Error("unhandled error",
			"request_id", requestID,
			"code", appErr.Code,
			"message", appErr.Message,
			"error", appErr.Err,
		)
	}

	response.SendError(c, appErr.Code, appErr.Message, appErr.Status)
}

func httpErrorToAppError(e *echo.HTTPError) *response.APIError {
	code := httpStatusToCode(e.Code)
	message := fmt.Sprintf("%v", e.Message)

	if e.Internal != nil {
		return &response.APIError{Code: code, Message: message, Status: e.Code, Err: e.Internal}
	}
	return &response.APIError{Code: code, Message: message, Status: e.Code}
}

func httpStatusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusMethodNotAllowed:
		return "METHOD_NOT_ALLOWED"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusRequestEntityTooLarge:
		return "PAYLOAD_TOO_LARGE"
	case http.StatusUnprocessableEntity:
		return "VALIDATION_ERROR"
	case http.StatusTooManyRequests:
		return "TOO_MANY_REQUESTS"
	case http.StatusInternalServerError:
		return "INTERNAL_ERROR"
	case http.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	default:
		if status >= 500 {
			return "INTERNAL_ERROR"
		}
		return "BAD_REQUEST"
	}
}

func Recover() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					stack := string(debug.Stack())
					requestID := GetRequestID(c)
					slog.Error("panic recovered",
						"request_id", requestID,
						"panic", fmt.Sprintf("%v", r),
						"stack", stack,
					)
					c.Error(response.Internal("internal server error", fmt.Errorf("panic: %v", r)))
				}
			}()
			return next(c)
		}
	}
}
