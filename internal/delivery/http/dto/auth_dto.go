package dto

import (
	"time"

	"task-management/internal/domain"
)

type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	Token string        `json:"token"`
	User  *UserResponse `json:"user,omitempty"`
}

type UserResponse struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	TeamID    *string `json:"team_id,omitempty"`
	CreatedAt string  `json:"created_at"`
}

type JoinTeamRequest struct {
	Code string `json:"code" validate:"required,min=2,max=20"`
}

func ToUserResponse(u *domain.User) UserResponse {
	createdAt := ""
	if !u.CreatedAt.IsZero() {
		createdAt = u.CreatedAt.UTC().Format(time.RFC3339)
	}
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		TeamID:    u.TeamID,
		CreatedAt: createdAt,
	}
}
