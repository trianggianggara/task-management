package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"task-management/internal/domain"
)

type TaskLogRepository interface {
	Create(ctx context.Context, tx *sqlx.Tx, log *domain.TaskLog) error
}
