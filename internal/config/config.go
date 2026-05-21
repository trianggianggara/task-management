package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL     string
	JWTSecret       string
	JWTExpiry       time.Duration
	AppPort         string
	Environment     string
	BcryptCost      int
	DBMaxOpenConns  int
	DBMaxIdleConns  int
	RateLimitRPS    int
	RateLimitBurst  int
	BodyLimitBytes  int64
}

func Load() (*Config, error) {
	env := getEnv("ENVIRONMENT", "development")

	expiryHours := getEnvInt("JWT_EXPIRY_HOURS", 24, 1)
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		if env == "production" {
			return nil, fmt.Errorf("JWT_SECRET is required in production environment")
		}
		jwtSecret = "change-me-in-production"
		slog.Warn("JWT_SECRET not set, using insecure default — do NOT use in production")
	}

	bcryptCost := getEnvInt("BCRYPT_COST", 12, 4)
	if bcryptCost < 4 {
		slog.Warn("BCRYPT_COST too low, using minimum 4")
		bcryptCost = 4
	}
	if bcryptCost > 31 {
		slog.Warn("BCRYPT_COST too high, using maximum 31")
		bcryptCost = 31
	}

	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/taskmanagement?sslmode=disable"),
		JWTSecret:      jwtSecret,
		JWTExpiry:      time.Duration(expiryHours) * time.Hour,
		AppPort:        getEnv("APP_PORT", "8080"),
		Environment:    env,
		BcryptCost:     bcryptCost,
		DBMaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25, 1),
		DBMaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5, 0),
		RateLimitRPS:   getEnvInt("RATE_LIMIT_RPS", 5, 1),
		RateLimitBurst: getEnvInt("RATE_LIMIT_BURST", 10, 1),
		BodyLimitBytes: int64(getEnvInt("BODY_LIMIT_KB", 1024, 1)) * 1024,
	}, nil
}

func getEnvInt(key string, fallback, min int) int {
	valStr := getEnv(key, strconv.Itoa(fallback))
	val, err := strconv.Atoi(valStr)
	if err != nil || val < min {
		slog.Warn("invalid config value, using fallback", "key", key, "value", valStr, "fallback", fallback)
		return fallback
	}
	return val
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
