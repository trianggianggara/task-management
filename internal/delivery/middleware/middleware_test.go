package middleware_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-management/internal/apperror"
	appMiddleware "task-management/internal/delivery/middleware"
)

func TestRequestID_Middleware_GeneratesUUID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := appMiddleware.RequestID()(func(c echo.Context) error {
		id := appMiddleware.GetRequestID(c)
		assert.NotEmpty(t, id)
		assert.Len(t, id, 36)
		return c.String(http.StatusOK, id)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
}

func TestRequestID_Middleware_ReusesHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "my-custom-id")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := appMiddleware.RequestID()(func(c echo.Context) error {
		id := appMiddleware.GetRequestID(c)
		assert.Equal(t, "my-custom-id", id)
		return c.String(http.StatusOK, id)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, "my-custom-id", rec.Header().Get("X-Request-ID"))
}

func TestLogger_Middleware_InfoFor2xx(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "log-test-id")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latencyMs := time.Since(start).Milliseconds()
			level := appMiddleware.LevelFromStatus(c.Response().Status)
			logger.LogAttrs(c.Request().Context(), level, "request",
				slog.String("request_id", appMiddleware.GetRequestID(c)),
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.Int("status", c.Response().Status),
				slog.Int64("latency_ms", latencyMs),
			)
			return err
		}
	}

	wrapped := appMiddleware.RequestID()(handler(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))

	err := wrapped(c)
	require.NoError(t, err)

	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "INFO", logEntry["level"])
	assert.Equal(t, "request", logEntry["msg"])
	assert.Equal(t, "log-test-id", logEntry["request_id"])
	assert.Equal(t, http.MethodGet, logEntry["method"])
	assert.Equal(t, "/test", logEntry["path"])
	assert.Equal(t, float64(http.StatusOK), logEntry["status"])
}

func TestLogger_Middleware_WarnFor4xx(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latencyMs := time.Since(start).Milliseconds()
			level := appMiddleware.LevelFromStatus(c.Response().Status)
			logger.LogAttrs(c.Request().Context(), level, "request",
				slog.String("request_id", "warn-test"),
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.Int("status", c.Response().Status),
				slog.Int64("latency_ms", latencyMs),
			)
			return err
		}
	}

	wrapped := handler(func(c echo.Context) error {
		return c.JSON(http.StatusNotFound, apperror.ToErrorResponse(apperror.NotFound("not found")))
	})

	err := wrapped(c)
	require.NoError(t, err)

	var logEntry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &logEntry))

	assert.Equal(t, "WARN", logEntry["level"])
	assert.Equal(t, float64(http.StatusNotFound), logEntry["status"])
}

func TestLogger_Middleware_ErrorFor5xx(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latencyMs := time.Since(start).Milliseconds()
			level := appMiddleware.LevelFromStatus(c.Response().Status)
			logger.LogAttrs(c.Request().Context(), level, "request",
				slog.String("request_id", "err-test"),
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.Int("status", c.Response().Status),
				slog.Int64("latency_ms", latencyMs),
			)
			return err
		}
	}

	wrapped := handler(func(c echo.Context) error {
		return c.JSON(http.StatusInternalServerError, apperror.ToErrorResponse(apperror.Internal("boom", nil)))
	})

	err := wrapped(c)
	require.NoError(t, err)

	var logEntry map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &logEntry))

	assert.Equal(t, "ERROR", logEntry["level"])
	assert.Equal(t, float64(http.StatusInternalServerError), logEntry["status"])
}

func TestErrorHandler_AppError(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = appMiddleware.ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	appMiddleware.ErrorHandler(apperror.NotFound("task not found"), c)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp apperror.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, "NOT_FOUND", resp.Code)
	assert.Equal(t, "task not found", resp.Message)
	assert.NotEmpty(t, resp.Timestamp)
}

func TestErrorHandler_EchoHTTPError_404(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = appMiddleware.ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	appMiddleware.ErrorHandler(echo.ErrNotFound, c)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp apperror.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, "NOT_FOUND", resp.Code)
	assert.NotEmpty(t, resp.Message)
}

func TestErrorHandler_EchoHTTPError_AllStatusCodes(t *testing.T) {
	tests := []struct {
		httpError *echo.HTTPError
		wantCode  string
	}{
		{echo.ErrBadRequest, "BAD_REQUEST"},
		{echo.ErrUnauthorized, "UNAUTHORIZED"},
		{echo.ErrForbidden, "FORBIDDEN"},
		{echo.ErrNotFound, "NOT_FOUND"},
		{echo.ErrMethodNotAllowed, "METHOD_NOT_ALLOWED"},
		{echo.ErrConflict, "CONFLICT"},
		{echo.NewHTTPError(http.StatusUnprocessableEntity, "validation"), "VALIDATION_ERROR"},
		{echo.NewHTTPError(http.StatusTooManyRequests, "rate limit"), "TOO_MANY_REQUESTS"},
		{echo.ErrInternalServerError, "INTERNAL_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.wantCode, func(t *testing.T) {
			e := echo.New()
			e.HTTPErrorHandler = appMiddleware.ErrorHandler
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			appMiddleware.ErrorHandler(tt.httpError, c)

			var resp apperror.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			assert.Equal(t, tt.wantCode, resp.Code)
		})
	}
}

func TestErrorHandler_GenericError_Returns500(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = appMiddleware.ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	appMiddleware.ErrorHandler(io.ErrUnexpectedEOF, c)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp apperror.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, "INTERNAL_ERROR", resp.Code)
	assert.Equal(t, "internal server error", resp.Message)
	assert.NotEmpty(t, resp.Timestamp)
}

func TestRecover_Middleware_RecoversFromPanic(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = appMiddleware.ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := appMiddleware.Recover()(func(c echo.Context) error {
		panic("something went terribly wrong")
	})

	err := handler(c)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp apperror.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, "INTERNAL_ERROR", resp.Code)
	assert.Equal(t, "internal server error", resp.Message)
	assert.NotEmpty(t, resp.Timestamp)
}

func TestRecover_Middleware_NoPanicPassesThrough(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := appMiddleware.Recover()(func(c echo.Context) error {
		return c.String(http.StatusOK, "all good")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "all good", rec.Body.String())
}

func TestAuth_Middleware_ReturnsAppError_NoHeader(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = appMiddleware.ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := appMiddleware.Auth("secret")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.Error(t, err)

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 401, appErr.Status)
	assert.Equal(t, "UNAUTHORIZED", appErr.Code)
}

func TestAuth_Middleware_ReturnsAppError_InvalidToken(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = appMiddleware.ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := appMiddleware.Auth("secret")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.Error(t, err)

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 401, appErr.Status)
}

func TestRateLimiter_ReturnsAppError(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = appMiddleware.ErrorHandler
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Real-IP", "192.168.1.10")

	handler := appMiddleware.RateLimiter()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	for i := 0; i < 10; i++ {
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		_ = handler(c)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := handler(c)
	require.Error(t, err)

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 429, appErr.Status)
	assert.Equal(t, "TOO_MANY_REQUESTS", appErr.Code)
}

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Real-IP", "192.168.1.11")

	handler := appMiddleware.RateLimiter()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		err := handler(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimiter_SeparatePerIP(t *testing.T) {
	e := echo.New()

	handler := appMiddleware.RateLimiter()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req1 := httptest.NewRequest(http.MethodPost, "/", nil)
	req1.Header.Set("X-Real-IP", "192.168.1.14")

	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.Header.Set("X-Real-IP", "192.168.1.15")

	for i := 0; i < 10; i++ {
		rec := httptest.NewRecorder()
		c := e.NewContext(req1, rec)
		_ = handler(c)
	}

	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)
	err1 := handler(c1)
	require.Error(t, err1)

	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	err2 := handler(c2)
	require.NoError(t, err2)
	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	e := echo.New()

	handler := appMiddleware.RateLimiter()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set("X-Real-IP", "192.168.1.16")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			err := handler(c)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	assert.Greater(t, successCount, 0)
	assert.LessOrEqual(t, successCount, 15)
}

func TestLevelFromStatus(t *testing.T) {
	assert.Equal(t, slog.LevelInfo, appMiddleware.LevelFromStatus(200))
	assert.Equal(t, slog.LevelInfo, appMiddleware.LevelFromStatus(201))
	assert.Equal(t, slog.LevelInfo, appMiddleware.LevelFromStatus(302))
	assert.Equal(t, slog.LevelWarn, appMiddleware.LevelFromStatus(400))
	assert.Equal(t, slog.LevelWarn, appMiddleware.LevelFromStatus(404))
	assert.Equal(t, slog.LevelWarn, appMiddleware.LevelFromStatus(429))
	assert.Equal(t, slog.LevelError, appMiddleware.LevelFromStatus(500))
	assert.Equal(t, slog.LevelError, appMiddleware.LevelFromStatus(502))
}
