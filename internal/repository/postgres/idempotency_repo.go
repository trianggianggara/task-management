package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"task-management/internal/repository"
)

type idempotencyRepo struct {
	db *sqlx.DB
}

func NewIdempotencyRepo(db *sqlx.DB) repository.IdempotencyRepository {
	return &idempotencyRepo{db: db}
}

func (r *idempotencyRepo) ClaimKey(ctx context.Context, tx *sqlx.Tx, key string, userID string) (bool, *repository.StoredResponse, error) {
	exec := r.executor(tx)
	query := `
		INSERT INTO idempotency_keys (key, user_id, created_at, expires_at)
		VALUES ($1, $2, NOW(), NOW() + INTERVAL '24 hours')
	`
	_, err := exec.ExecContext(ctx, query, key, userID)
	if err == nil {
		return true, nil, nil
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		var cached repository.StoredResponse
		selectQuery := `
			SELECT response_status, response_body
			FROM idempotency_keys
			WHERE key = $1
		`
		err := exec.QueryRowContext(ctx, selectQuery, key).Scan(&cached.Status, &cached.Body)
		if err != nil {
			return false, nil, fmt.Errorf("idempotencyRepo.ClaimKey read: %w", err)
		}
		return false, &cached, nil
	}

	return false, nil, fmt.Errorf("idempotencyRepo.ClaimKey: %w", err)
}

func (r *idempotencyRepo) StoreResponse(ctx context.Context, tx *sqlx.Tx, key string, status int, body string) error {
	exec := r.executor(tx)
	query := `
		UPDATE idempotency_keys
		SET response_status = $1, response_body = $2::jsonb
		WHERE key = $3
	`
	result, err := exec.ExecContext(ctx, query, status, body, key)
	if err != nil {
		return fmt.Errorf("idempotencyRepo.StoreResponse: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *idempotencyRepo) PurgeExpired(ctx context.Context) error {
	query := `DELETE FROM idempotency_keys WHERE expires_at < NOW()`
	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("idempotencyRepo.PurgeExpired: %w", err)
	}
	return nil
}

func (r *idempotencyRepo) executor(tx *sqlx.Tx) interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
} {
	if tx != nil {
		return tx
	}
	return r.db
}
