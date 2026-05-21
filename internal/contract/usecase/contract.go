package usecase

import (
	"task-management/internal/config"
	"task-management/internal/contract/common"
	repositoryContract "task-management/internal/contract/repository"
	"task-management/internal/usecase"
)

type Contract struct {
	AuthUsecase usecase.AuthUsecase
	TaskUsecase usecase.TaskUsecase
}

func New(cfg *config.Config, common *common.Contract, repo *repositoryContract.Contract) (*Contract, error) {
	return &Contract{
		AuthUsecase: usecase.NewAuthUsecase(repo.UserRepo, common.Hasher, cfg.JWTSecret, cfg.JWTExpiry),
		TaskUsecase: usecase.NewTaskUsecase(
			common.TxManager,
			repo.TaskRepo,
			repo.IdempotencyRepo,
			repo.TaskLogRepo,
			repo.UserRepo,
		),
	}, nil
}
