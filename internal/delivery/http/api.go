package http

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
	"task-management/internal/contract"
	appMiddleware "task-management/internal/delivery/middleware"
)

type Api struct {
	e   *echo.Echo
	c   *contract.Contract
	cfg *config.Config
}

func New(cfg *config.Config, c *contract.Contract) *Api {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(appMiddleware.RequestID())
	e.Use(appMiddleware.Logger())
	e.Use(appMiddleware.Recover())
	e.Use(corsMiddleware())
	e.HTTPErrorHandler = appMiddleware.ErrorHandler

	return &Api{e: e, c: c, cfg: cfg}
}

func (a *Api) Start() error {
	a.registerRoutes()

	a.cfg = a.cfg

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		addr := ":" + a.cfg.AppPort
		slog.Info("server starting", "address", addr, "environment", a.cfg.Environment)
		if err := a.e.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("server shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := a.e.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}
	slog.Info("server exited")
	return nil
}

func (a *Api) registerRoutes() {
	if a.cfg.IsDevelopment() {
		a.e.GET("/swagger/*", echoSwagger.WrapHandler)
		slog.Info("swagger enabled", "url", "http://localhost:"+a.cfg.AppPort+"/swagger/index.html")
	}

	a.e.GET("/api/v1/health", func(c echo.Context) error {
		if err := a.c.Common.DB.PingContext(c.Request().Context()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unhealthy"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

	NewRoute(a)
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
