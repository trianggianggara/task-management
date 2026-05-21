package common

import (
	"github.com/jmoiron/sqlx"
	"task-management/internal/config"
	"task-management/internal/repository/postgres"
	"task-management/pkg/utils/autotx"
	"task-management/pkg/utils/password"
)

type Contract struct {
	DB        *sqlx.DB
	TxManager autotx.Manager
	Hasher    password.Hasher
}

func New(cfg *config.Config) (*Contract, error) {
	db, err := postgres.Connect(cfg.DatabaseURL, cfg.DBMaxOpenConns, cfg.DBMaxIdleConns)
	if err != nil {
		return nil, err
	}

	hasher, err := password.NewBcryptHasher(12)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Contract{
		DB:        db,
		TxManager: autotx.New(db),
		Hasher:    hasher,
	}, nil
}
