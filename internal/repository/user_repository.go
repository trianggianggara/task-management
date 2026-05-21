package repository

import (
	"context"

	"task-management/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
	UpdateTeamID(ctx context.Context, userID string, teamID *string) error
	FindTeamByCode(ctx context.Context, code string) (*domain.Team, error)
}
