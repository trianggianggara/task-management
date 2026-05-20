package http_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"task-management/internal/apperror"
	handler "task-management/internal/delivery/http"
	appMiddleware "task-management/internal/delivery/middleware"
	"task-management/internal/domain"
	"task-management/internal/usecase"
)

type stubUserRepo struct{}

func (s *stubUserRepo) Create(ctx context.Context, user *domain.User) error {
	user.ID = "user-1"
	return nil
}

func (s *stubUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if email == "exists@example.com" {
		return &domain.User{ID: "existing", Email: email}, nil
	}
	if email == "test@example.com" {
		return &domain.User{ID: "user-1", Email: email, PasswordHash: "$2a$12$hashed"}, nil
	}
	return nil, sql.ErrNoRows
}

func (s *stubUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return &domain.User{ID: id, Email: "test@example.com", Name: "Test"}, nil
}

type stubPasswordService struct{}

func (s *stubPasswordService) Hash(plain string) (string, error) {
	return "$2a$12$hashed", nil
}

func (s *stubPasswordService) Verify(plain, hashed string) error {
	if plain == "correct" {
		return nil
	}
	return &someError{msg: "hash mismatch"}
}

type someError struct{ msg string }

func (e *someError) Error() string { return e.msg }

func setupTestEcho() *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = appMiddleware.ErrorHandler
	e.Use(appMiddleware.RequestID())

	uc := usecase.NewAuthUsecase(&stubUserRepo{}, &stubPasswordService{}, "test-secret", 24)
	h := handler.NewAuthHandler(uc)

	e.POST("/api/v1/auth/register", h.Register)
	e.POST("/api/v1/auth/login", h.Login)

	return e
}

func TestRegisterHandler_Success(t *testing.T) {
	e := setupTestEcho()

	body := `{"name":"Test User","email":"new@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp handler.SuccessResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, "User registered successfully", resp.Message)
	assert.NotEmpty(t, resp.Meta.RequestID)
	assert.NotEmpty(t, resp.Meta.Timestamp)

	authResp, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, authResp["token"])
	user, ok := authResp["user"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Test User", user["name"])
	assert.Equal(t, "new@example.com", user["email"])
}

func TestRegisterHandler_DuplicateEmail(t *testing.T) {
	e := setupTestEcho()

	body := `{"name":"Test","email":"exists@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)

	var errResp apperror.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "error", errResp.Status)
	assert.Equal(t, "CONFLICT", errResp.Code)
}

func TestRegisterHandler_InvalidBody(t *testing.T) {
	e := setupTestEcho()

	body := `{"name":"","email":"bad-email","password":"123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var errResp apperror.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "VALIDATION_ERROR", errResp.Code)
}

func TestRegisterHandler_MalformedJSON(t *testing.T) {
	e := setupTestEcho()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLoginHandler_Success(t *testing.T) {
	e := setupTestEcho()

	body := `{"email":"test@example.com","password":"correct"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp handler.SuccessResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, "Login successful", resp.Message)
	assert.NotEmpty(t, resp.Meta.RequestID)

	authResp, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, authResp["token"])
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	e := setupTestEcho()

	body := `{"email":"test@example.com","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var errResp apperror.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "UNAUTHORIZED", errResp.Code)
}

func TestLoginHandler_InvalidBody(t *testing.T) {
	e := setupTestEcho()

	body := `{"email":"","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUnauthenticatedRequest(t *testing.T) {
	e := setupTestEcho()

	protected := e.Group("/api/v1/tasks")
	protected.Use(appMiddleware.Auth("test-secret"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var errResp apperror.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "UNAUTHORIZED", errResp.Code)
}
