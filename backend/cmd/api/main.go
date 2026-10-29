// Command api runs the KampusOps HTTP API (migrations are applied on startup).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kampusops/backend/internal/platform/config"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/logger"
	"github.com/kampusops/backend/internal/platform/redis"
	"github.com/kampusops/backend/internal/server"
	"github.com/kampusops/backend/migrations"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database connection failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if cfg.MigrateOnStart {
		if err := db.Migrate(ctx, pool, migrations.FS); err != nil {
			log.Error("migration failed", "err", err)
			os.Exit(1)
		}
	}

	var kv redis.KV
	rc, err := redis.Connect(ctx, cfg.RedisURL)
	if err != nil {
		log.Warn("redis unavailable, using in-memory idempotency store (single instance only)", "err", err)
		kv = redis.NewMemoryKV()
	} else {
		kv = redis.NewKV(rc)
		defer rc.Close()
	}

	app, _ := server.New(server.Deps{Cfg: cfg, Pool: pool, KV: kv, Log: log})
	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		_ = app.ShutdownWithTimeout(10 * time.Second)
	}()
	log.Info("api listening", "port", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Error("server stopped", "err", err)
	}
}
