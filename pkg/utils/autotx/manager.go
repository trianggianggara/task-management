package autotx

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Manager interface {
	Run(ctx context.Context, fn func(tx *sqlx.Tx) error) error
}

type txManager struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) Manager {
	return &txManager{db: db}
}

func (m *txManager) Run(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
