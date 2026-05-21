package main

import (
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	_ "task-management/docs"
	"task-management/internal/config"
	"task-management/internal/contract"
	"task-management/internal/delivery/http"
)

// @title           Task Management API
// @version         1.0
// @description     Multi-user task management API with idempotency support.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@taskmanagement.io

// @host           localhost:8080

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	c, stopper, err := contract.New(cfg)
	if err != nil {
		slog.Error("failed to initialize", "error", err)
		os.Exit(1)
	}
	if stopper != nil {
		defer stopper()
	}
	defer c.Common.DB.Close()

	runMigrations(cfg.DatabaseURL)

	http.New(cfg, c).Start()
}

func runMigrations(dsn string) {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		slog.Error("migration init failed", "error", err)
		os.Exit(1)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")
}
