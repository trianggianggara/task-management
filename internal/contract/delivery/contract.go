package delivery

import (
	"task-management/internal/delivery/http/api"
	usecaseContract "task-management/internal/contract/usecase"
)

type Contract struct {
	AuthHandler *api.AuthHandler
	TaskHandler *api.TaskHandler
}

func New(uc *usecaseContract.Contract) *Contract {
	return &Contract{
		AuthHandler: api.NewAuthHandler(uc.AuthUsecase),
		TaskHandler: api.NewTaskHandler(uc.TaskUsecase),
	}
}
