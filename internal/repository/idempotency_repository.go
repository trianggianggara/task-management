package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type StoredResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}

type IdempotencyRepository interface {
	ClaimKey(ctx context.Context, tx *sqlx.Tx, key string, userID string) (claimed bool, cached *StoredResponse, err error)
	StoreResponse(ctx context.Context, tx *sqlx.Tx, key string, status int, body string) error
	PurgeExpired(ctx context.Context) error
}
