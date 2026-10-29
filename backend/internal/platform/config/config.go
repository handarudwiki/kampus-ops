// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"strings"
)

type Config struct {
	Env           string
	Port          string
	DatabaseURL   string
	RedisURL      string
	JWTSecret     string
	WebhookSecret string
	CORSOrigins   string
	LogLevel      string
	// MigrateOnStart runs embedded migrations when the API boots (default true).
	MigrateOnStart bool
}

func Load() Config {
	return Config{
		Env:            get("APP_ENV", "development"),
		Port:           get("PORT", "8080"),
		DatabaseURL:    get("DATABASE_URL", "postgres://kampus:kampus@localhost:5432/kampusops?sslmode=disable"),
		RedisURL:       get("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:      get("JWT_SECRET", "dev-secret-change-me"),
		WebhookSecret:  get("WEBHOOK_SECRET", "dev-webhook-secret"),
		CORSOrigins:    get("CORS_ORIGINS", "http://localhost:5173"),
		LogLevel:       get("LOG_LEVEL", "info"),
		MigrateOnStart: strings.ToLower(get("MIGRATE_ON_START", "true")) != "false",
	}
}

func get(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}
