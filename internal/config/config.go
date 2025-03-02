package config

import (
	"time"

	"github.com/Bethel-nz/tickit/internal/env"
	"github.com/Bethel-nz/tickit/internal/types"
)

// LoadConfig reads environment variables and returns a populated AppConfig.
func LoadConfig() *types.AppConfig {
	return &types.AppConfig{
		DatabaseURL:    env.String("DATABASE_URL", "postgres://admin:adminpassword@db:5432/tickit?sslmode=disable", env.Require).Get(),
		AppPort:        env.Int("APP_PORT", 5479, env.Optional).Get(),
		DebugMode:      env.Bool("DEBUG_MODE", false, env.Optional).Get(),
		RequestTimeout: env.Duration("REQUEST_TIMEOUT", 5*time.Second, env.Optional).Get(),
		Threshold:      env.Float64("THRESHOLD", 0.75, env.Optional).Get(),
		RedisURL:       env.String("REDIS_URL", "localhost:6379", env.Optional).Get(),
		MaxOpenConns:   env.Int("MAX_OPEN_CONNS", 25, env.Optional).Get(),
		MaxIdleTime:    env.Duration("MAX_IDLE_TIME", 5*time.Minute, env.Optional).Get(),
	}
}
