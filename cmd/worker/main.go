package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"task-management/internal/config"
	"task-management/internal/contract/common"
	"task-management/internal/job"
	"task-management/internal/repository/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	commonContract, err := common.New(cfg)
	if err != nil {
		slog.Error("failed to init common", "error", err)
		os.Exit(1)
	}
	defer commonContract.DB.Close()

	idempRepo := postgres.NewIdempotencyRepo(commonContract.DB)

	runner := job.NewRunner(
		job.NewPurgeIdempotencyKeys(idempRepo),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("worker started", "jobs", 1)
	runner.Start(ctx)
	slog.Info("worker stopped")
}
