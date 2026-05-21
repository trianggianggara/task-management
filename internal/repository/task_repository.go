package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"task-management/internal/domain"
)

type TaskFilter struct {
	Status domain.TaskStatus
	Search string
	Page   int
	Limit  int
}

type TaskRepository interface {
	Create(ctx context.Context, tx *sqlx.Tx, task *domain.Task) error
	FindByID(ctx context.Context, id string) (*domain.Task, error)
	FindByUserID(ctx context.Context, userID string, filter TaskFilter) ([]domain.Task, int64, error)
	Update(ctx context.Context, tx *sqlx.Tx, task *domain.Task) error
	SoftDelete(ctx context.Context, tx *sqlx.Tx, id string) error
	FindByIDForUpdate(ctx context.Context, tx *sqlx.Tx, id string) (*domain.Task, error)
	UpdateAssignee(ctx context.Context, tx *sqlx.Tx, taskID string, assigneeID *string) error
}
