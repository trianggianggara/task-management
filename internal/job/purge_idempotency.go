package job

import (
	"context"
	"time"

	"task-management/internal/repository"
)

type PurgeIdempotencyKeysJob struct {
	repo repository.IdempotencyRepository
}

func NewPurgeIdempotencyKeysJob(repo repository.IdempotencyRepository) *PurgeIdempotencyKeysJob {
	return &PurgeIdempotencyKeysJob{repo: repo}
}

func (j *PurgeIdempotencyKeysJob) Name() string {
	return "purge-idempotency-keys"
}

func (j *PurgeIdempotencyKeysJob) Interval() time.Duration {
	return 1 * time.Hour
}

func (j *PurgeIdempotencyKeysJob) Run(ctx context.Context) error {
	return j.repo.PurgeExpired(ctx)
}
