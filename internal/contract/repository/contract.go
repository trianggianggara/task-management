package repository

import (
	"task-management/internal/config"
	"task-management/internal/contract/common"
	"task-management/internal/repository"
	"task-management/internal/repository/postgres"
)

type Contract struct {
	UserRepo       repository.UserRepository
	TaskRepo       repository.TaskRepository
	IdempotencyRepo repository.IdempotencyRepository
	TaskLogRepo    repository.TaskLogRepository
}

func New(cfg *config.Config, common *common.Contract) (*Contract, error) {
	return &Contract{
		UserRepo:       postgres.NewUserRepo(common.DB),
		TaskRepo:       postgres.NewTaskRepo(common.DB),
		IdempotencyRepo: postgres.NewIdempotencyRepo(common.DB),
		TaskLogRepo:    postgres.NewTaskLogRepo(common.DB),
	}, nil
}
