package api

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"task-management/pkg/utils/response"
	"task-management/internal/delivery/http/dto"
	"task-management/internal/delivery/middleware"
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
// @Param       request body dto.RegisterRequest true "Register payload"
// @Success     201 {object} object
// @Failure     400 {object} object
// @Failure     409 {object} object
// @Failure     422 {object} object
// @Failure     429 {object} object
// @Router      /api/v1/auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest("invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return response.ValidationError(err.Error())
	}

	user, token, err := h.authUC.Register(c.Request().Context(), req.Name, req.Email, req.Password)
	if err != nil {
		return err
	}

	userResp := dto.ToUserResponse(user)
	return response.Success(c, http.StatusCreated, "User registered successfully", dto.AuthResponse{
		Token: token,
		User:  &userResp,
	})
}

// @Summary     Login
// @Description Authenticate with email and password to receive a JWT token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body dto.LoginRequest true "Login payload"
// @Success     200 {object} object
// @Failure     400 {object} object
// @Failure     401 {object} object
// @Failure     422 {object} object
// @Failure     429 {object} object
// @Router      /api/v1/auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest("invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return response.ValidationError(err.Error())
	}

	token, err := h.authUC.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "Login successful", dto.AuthResponse{
		Token: token,
	})
}

// @Summary     Join team
// @Description Join a team by code. Available: ENG (Engineering), DSG (Design), PDT (Product)
// @Tags        auth
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       request body dto.JoinTeamRequest true "codes: ENG, DSG, PDT"
// @Success     200 {object} object
// @Failure     400 {object} object
// @Failure     401 {object} object
// @Failure     404 {object} object
// @Router      /api/v1/auth/team [put]
func (h *AuthHandler) JoinTeam(c echo.Context) error {
	var req dto.JoinTeamRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest("invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return response.ValidationError(err.Error())
	}

	userID := middleware.GetUserID(c)
	if err := h.authUC.JoinTeam(c.Request().Context(), userID, req.Code); err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "Team joined successfully", nil)
}

// @Summary     Leave team
// @Description Leave current team
// @Tags        auth
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Success     200 {object} object
// @Failure     401 {object} object
// @Router      /api/v1/auth/team [delete]
func (h *AuthHandler) LeaveTeam(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if err := h.authUC.LeaveTeam(c.Request().Context(), userID); err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "Team left successfully", nil)
}
