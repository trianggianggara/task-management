package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"task-management/internal/domain"
	"task-management/internal/repository"
)

type taskRepo struct {
	db *sqlx.DB
}

func NewTaskRepo(db *sqlx.DB) repository.TaskRepository {
	return &taskRepo{db: db}
}

func (r *taskRepo) Create(ctx context.Context, tx *sqlx.Tx, task *domain.Task) error {
	query := `
		INSERT INTO tasks (user_id, title, description, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	exec := r.executor(tx)
	return exec.QueryRowContext(ctx, query,
		task.UserID, task.Title, task.Description, task.Status,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
}

func (r *taskRepo) FindByID(ctx context.Context, id string) (*domain.Task, error) {
	task := &domain.Task{}
	query := `
		SELECT id, user_id, assignee_id, title, description, status, created_at, updated_at, deleted_at
		FROM tasks
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &task.UserID, &task.AssigneeID, &task.Title, &task.Description,
		&task.Status, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("taskRepo.FindByID: %w", err)
	}
	return task, nil
}

func (r *taskRepo) FindByUserID(ctx context.Context, userID string, filter repository.TaskFilter) ([]domain.Task, int64, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "user_id = $1")
	args = append(args, userID)

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.Status != "" {
		args = append(args, string(filter.Status))
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}

	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions, fmt.Sprintf("title ILIKE $%d", len(args)))
	}

	whereClause := strings.Join(conditions, " AND ")

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks WHERE %s", whereClause)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("taskRepo.FindByUserID count: %w", err)
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit

	dataQuery := fmt.Sprintf(`
		SELECT id, user_id, assignee_id, title, description, status, created_at, updated_at, deleted_at
		FROM tasks
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, len(args)+1, len(args)+2)

	args = append(args, filter.Limit, offset)

	rows, err := r.db.QueryxContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("taskRepo.FindByUserID query: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		var task domain.Task
		if err := rows.Scan(
			&task.ID, &task.UserID, &task.AssigneeID, &task.Title, &task.Description,
			&task.Status, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("taskRepo.FindByUserID scan: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, total, nil
}

func (r *taskRepo) Update(ctx context.Context, tx *sqlx.Tx, task *domain.Task) error {
	query := `
		UPDATE tasks
		SET title = $1, description = $2, status = $3, updated_at = NOW()
		WHERE id = $4 AND deleted_at IS NULL
		RETURNING updated_at
	`
	exec := r.executor(tx)
	return exec.QueryRowContext(ctx, query,
		task.Title, task.Description, task.Status, task.ID,
	).Scan(&task.UpdatedAt)
}

func (r *taskRepo) SoftDelete(ctx context.Context, tx *sqlx.Tx, id string) error {
	query := `
		UPDATE tasks SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	exec := r.executor(tx)
	result, err := exec.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("taskRepo.SoftDelete: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *taskRepo) FindByIDForUpdate(ctx context.Context, tx *sqlx.Tx, id string) (*domain.Task, error) {
	task := &domain.Task{}
	query := `
		SELECT id, user_id, assignee_id, title, description, status, created_at, updated_at, deleted_at
		FROM tasks
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`
	exec := r.executor(tx)
	err := exec.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &task.UserID, &task.AssigneeID, &task.Title, &task.Description,
		&task.Status, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("taskRepo.FindByIDForUpdate: %w", err)
	}
	return task, nil
}

func (r *taskRepo) UpdateAssignee(ctx context.Context, tx *sqlx.Tx, taskID string, assigneeID *string) error {
	query := `
		UPDATE tasks SET assignee_id = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`
	exec := r.executor(tx)
	result, err := exec.ExecContext(ctx, query, assigneeID, taskID)
	if err != nil {
		return fmt.Errorf("taskRepo.UpdateAssignee: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *taskRepo) executor(tx *sqlx.Tx) interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if tx != nil {
		return tx
	}
	return r.db
}
