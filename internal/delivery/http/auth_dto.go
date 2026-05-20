package http

import (
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

func ToUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		TeamID:    u.TeamID,
		CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
