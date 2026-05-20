package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"

	"task-management/internal/config"
	appMiddleware "task-management/internal/delivery/middleware"
)

// @title           Task Management API
// @version         1.0
// @description     Multi-user task management API with idempotency support.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@taskmanagement.io

// @host           localhost:8080
// @BasePath       /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	cfg := config.Load()

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
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

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
