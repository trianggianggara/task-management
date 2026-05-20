package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/labstack/echo/v4"
	"task-management/internal/apperror"
)

func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	var appErr *apperror.AppError

	switch e := err.(type) {
	case *apperror.AppError:
		appErr = e
	case *echo.HTTPError:
		appErr = httpErrorToAppError(e)
	default:
		appErr = apperror.Internal("internal server error", err)
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

	resp := apperror.ToErrorResponse(appErr)
	resp.Meta = apperror.NewMeta(GetRequestID(c))
	c.JSON(appErr.Status, resp)
}

func httpErrorToAppError(e *echo.HTTPError) *apperror.AppError {
	code := httpStatusToCode(e.Code)
	message := fmt.Sprintf("%v", e.Message)

	if e.Internal != nil {
		return &apperror.AppError{Code: code, Message: message, Status: e.Code, Err: e.Internal}
	}
	return &apperror.AppError{Code: code, Message: message, Status: e.Code}
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
					c.Error(apperror.Internal("internal server error", fmt.Errorf("panic: %v", r)))
				}
			}()
			return next(c)
		}
	}
}
