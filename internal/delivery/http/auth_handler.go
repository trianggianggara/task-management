package http

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"task-management/internal/apperror"
	"task-management/internal/usecase"
)

type AuthHandler struct {
	authUC    usecase.AuthUsecase
	validator *validator.Validate
}

func NewAuthHandler(authUC usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{
		authUC:    authUC,
		validator: validator.New(),
	}
}

// @Summary     Register new user
// @Description Register a new user account and receive a JWT token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body RegisterRequest true "Register payload"
// @Success     201 {object} SuccessResponse{data=AuthResponse}
// @Failure     400 {object} apperror.ErrorResponse
// @Failure     409 {object} apperror.ErrorResponse
// @Failure     422 {object} apperror.ErrorResponse
// @Failure     429 {object} apperror.ErrorResponse
// @Router      /api/v1/auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return apperror.BadRequest("invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return apperror.ValidationError(err.Error())
	}

	user, token, err := h.authUC.Register(c.Request().Context(), req.Name, req.Email, req.Password)
	if err != nil {
		return err
	}

	userResp := ToUserResponse(user)
	return Success(c, http.StatusCreated, "User registered successfully", AuthResponse{
		Token: token,
		User:  &userResp,
	})
}

// @Summary     Login
// @Description Authenticate with email and password to receive a JWT token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body LoginRequest true "Login payload"
// @Success     200 {object} SuccessResponse{data=AuthResponse}
// @Failure     400 {object} apperror.ErrorResponse
// @Failure     401 {object} apperror.ErrorResponse
// @Failure     422 {object} apperror.ErrorResponse
// @Failure     429 {object} apperror.ErrorResponse
// @Router      /api/v1/auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return apperror.BadRequest("invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return apperror.ValidationError(err.Error())
	}

	token, err := h.authUC.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return err
	}

	return Success(c, http.StatusOK, "Login successful", AuthResponse{
		Token: token,
	})
}
