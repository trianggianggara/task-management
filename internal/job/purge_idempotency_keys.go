package job

import (
	"context"
	"time"

	"task-management/internal/repository"
)

type PurgeIdempotencyKeys struct {
	repo repository.IdempotencyRepository
}

func NewPurgeIdempotencyKeys(repo repository.IdempotencyRepository) *PurgeIdempotencyKeys {
	return &PurgeIdempotencyKeys{repo: repo}
}

func (j *PurgeIdempotencyKeys) Name() string { return "purge-idempotency-keys" }

func (j *PurgeIdempotencyKeys) Interval() time.Duration { return 1 * time.Hour }

func (j *PurgeIdempotencyKeys) Run(ctx context.Context) error {
	return j.repo.PurgeExpired(ctx)
}
