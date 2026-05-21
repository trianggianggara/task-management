package contract

import (
	"task-management/internal/config"
	"task-management/internal/contract/common"
	"task-management/internal/contract/delivery"
	"task-management/internal/contract/repository"
	"task-management/internal/contract/usecase"
)

type Contract struct {
	Cfg        *config.Config
	Usecase    *usecase.Contract
	Repository *repository.Contract
	Delivery   *delivery.Contract
	Common     *common.Contract
}

func New(cfg *config.Config) (c *Contract, stopper func(), err error) {
	c = &Contract{}
	c.Cfg = cfg

	c.Common, err = common.New(cfg)
	if err != nil {
		return c, nil, err
	}

	r, err := repository.New(cfg, c.Common)
	if err != nil {
		return c, nil, err
	}
	c.Repository = r

	uc, err := usecase.New(cfg, c.Common, r)
	if err != nil {
		return c, nil, err
	}
	c.Usecase = uc

	c.Delivery = delivery.New(uc)

	return c, nil, nil
}
