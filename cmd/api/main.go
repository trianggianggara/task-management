package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"

	_ "task-management/docs"
	"task-management/internal/config"
	handler "task-management/internal/delivery/http"
	appMiddleware "task-management/internal/delivery/middleware"
	"task-management/internal/repository/postgres"
	"task-management/internal/usecase"
	"task-management/pkg/password"
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

	db, err := postgres.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("database connected")

	runMigrations(cfg.DatabaseURL)

	userRepo := postgres.NewUserRepo(db)

	hasher := password.NewBcryptHasher()

	authUC := usecase.NewAuthUsecase(userRepo, hasher, cfg.JWTSecret, cfg.JWTExpiry)

	authHandler := handler.NewAuthHandler(authUC)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(appMiddleware.RequestID())
	e.Use(appMiddleware.Logger())
	e.Use(appMiddleware.Recover())
	e.Use(corsMiddleware())
	e.HTTPErrorHandler = appMiddleware.ErrorHandler

	if cfg.IsDevelopment() {
		e.GET("/swagger/*", echoSwagger.WrapHandler)
		slog.Info("swagger enabled", "url", "http://localhost:"+cfg.AppPort+"/swagger/index.html")
	}

	e.GET("/api/v1/health", func(c echo.Context) error {
		if err := db.PingContext(c.Request().Context()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unhealthy"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

	auth := e.Group("/api/v1/auth")
	auth.Use(appMiddleware.RateLimiter())
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		addr := ":" + cfg.AppPort
		slog.Info("server starting", "address", addr, "environment", cfg.Environment)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("server shutdown", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("server shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}
	slog.Info("server exited")
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

func corsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Access-Control-Allow-Origin", "*")
			c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, X-Request-ID")
			c.Response().Header().Set("Access-Control-Max-Age", "86400")

			if c.Request().Method == http.MethodOptions {
				return c.NoContent(http.StatusNoContent)
			}

			return next(c)
		}
	}
}
