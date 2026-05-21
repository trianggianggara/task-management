package http

import (
	"context"

	appMiddleware "task-management/internal/delivery/middleware"
)

func NewRoute(a *Api) {
	pub := a.e.Group("/api/v1/auth")
	pub.Use(appMiddleware.RateLimiter(context.Background(), a.cfg.RateLimitRPS, a.cfg.RateLimitBurst))
	pub.POST("/register", a.c.Delivery.AuthHandler.Register)
	pub.POST("/login", a.c.Delivery.AuthHandler.Login)

	prot := a.e.Group("/api/v1/auth")
	prot.Use(appMiddleware.Auth(a.cfg.JWTSecret))
	prot.PUT("/team", a.c.Delivery.AuthHandler.JoinTeam)
	prot.DELETE("/team", a.c.Delivery.AuthHandler.LeaveTeam)

	tasks := a.e.Group("/api/v1/tasks")
	tasks.Use(appMiddleware.Auth(a.cfg.JWTSecret))
	tasks.POST("", a.c.Delivery.TaskHandler.Create)
	tasks.GET("", a.c.Delivery.TaskHandler.List)
	tasks.GET("/:id", a.c.Delivery.TaskHandler.Get)
	tasks.PUT("/:id", a.c.Delivery.TaskHandler.Update)
	tasks.DELETE("/:id", a.c.Delivery.TaskHandler.Delete)
	tasks.POST("/:id/assign", a.c.Delivery.TaskHandler.Assign)
}
