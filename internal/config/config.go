package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	JWTExpiry   time.Duration
	AppPort     string
	Environment string
}

func Load() (*Config, error) {
	env := getEnv("ENVIRONMENT", "development")

	expiryHoursStr := getEnv("JWT_EXPIRY_HOURS", "24")
	expiryHours, err := strconv.Atoi(expiryHoursStr)
	if err != nil || expiryHours < 1 {
		slog.Warn("invalid JWT_EXPIRY_HOURS, using default 24", "value", expiryHoursStr)
		expiryHours = 24
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		if env == "production" {
			return nil, fmt.Errorf("JWT_SECRET is required in production environment")
		}
		jwtSecret = "change-me-in-production"
		slog.Warn("JWT_SECRET not set, using insecure default — do NOT use in production")
	}

	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/taskmanagement?sslmode=disable"),
		JWTSecret:   jwtSecret,
		JWTExpiry:   time.Duration(expiryHours) * time.Hour,
		AppPort:     getEnv("APP_PORT", "8080"),
		Environment: env,
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}
