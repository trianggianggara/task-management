package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"task-management/internal/domain"
	"task-management/internal/repository"
)

type taskLogRepo struct {
	db *sqlx.DB
}

func NewTaskLogRepo(db *sqlx.DB) repository.TaskLogRepository {
	return &taskLogRepo{db: db}
}

func (r *taskLogRepo) Create(ctx context.Context, tx *sqlx.Tx, log *domain.TaskLog) error {
	query := `
		INSERT INTO task_logs (task_id, action, old_value, new_value, changed_by)
		VALUES ($1, $2, $3::jsonb, $4::jsonb, $5)
		RETURNING id, created_at
	`
	err := tx.QueryRowContext(ctx, query,
		log.TaskID, log.Action, log.OldValue, log.NewValue, log.ChangedBy,
	).Scan(&log.ID, &log.CreatedAt)
	if err != nil {
		return fmt.Errorf("taskLogRepo.Create: %w", err)
	}
	return nil
}
