package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
	"task-management/pkg/utils/response"
)

func RateLimiter(ctx context.Context, rps, burst int) echo.MiddlewareFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				for ip, cl := range clients {
					if time.Since(cl.lastSeen) > 3*time.Minute {
						delete(clients, ip)
					}
				}
				mu.Unlock()
			}
		}
	}()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			mu.Lock()
			cl, exists := clients[ip]
			if !exists {
				cl = &client{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
				clients[ip] = cl
			}
			cl.lastSeen = time.Now()
			mu.Unlock()

			if !cl.limiter.Allow() {
				return response.TooManyRequests("rate limit exceeded, try again later")
			}

			return next(c)
		}
	}
}
