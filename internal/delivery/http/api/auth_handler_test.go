package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"task-management/internal/delivery/http/api"
	appMiddleware "task-management/internal/delivery/middleware"
	"task-management/internal/domain"
	"task-management/internal/usecase"
	"task-management/pkg/utils/response"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	return nil, nil
}

func (s *stubUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return &domain.User{ID: id, Email: "test@example.com", Name: "Test"}, nil
}

func (s *stubUserRepo) UpdateTeamID(ctx context.Context, userID string, teamID *string) error {
	return nil
}

func (s *stubUserRepo) FindTeamByCode(ctx context.Context, code string) (*domain.Team, error) {
	return &domain.Team{ID: "team-1", Code: code, Name: "Test"}, nil
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
	h := api.NewAuthHandler(uc)

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

	var resp response.SuccessResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.RequestID)
	assert.NotEmpty(t, resp.Timestamp)

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

	var errResp response.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))

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

	var errResp response.ErrorResponse
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

	var resp response.SuccessResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.RequestID)

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

	var errResp response.ErrorResponse
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
	protected.GET("", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var errResp response.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "UNAUTHORIZED", errResp.Code)
}
