// Command worker runs background jobs on Asynq (Redis): periodic lecturer-change
// application, bill due reminders and IPK recompute.
package main

import (
	"context"
	"os"
	"time"
	_ "time/tzdata"

	"github.com/hibiken/asynq"

	"github.com/kampusops/backend/internal/jobs"
	"github.com/kampusops/backend/internal/platform/config"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/logger"
	"github.com/kampusops/backend/internal/tenant"
	"github.com/kampusops/backend/migrations"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, migrations.FS); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}
	r := &jobs.Runner{Pool: pool, Tenants: tenant.NewStore(pool), Now: time.Now}
	redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		log.Error("redis url", "err", err)
		os.Exit(1)
	}
	sched := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{Location: time.UTC})
	for spec, typ := range map[string]string{"@every 1h": jobs.TypeApplyDosen, "0 1 * * *": jobs.TypeDueReminder, "0 19 * * *": jobs.TypeRecompute} {
		if _, err := sched.Register(spec, asynq.NewTask(typ, nil), asynq.Unique(30*time.Minute)); err != nil {
			log.Error("schedule", "err", err)
			os.Exit(1)
		}
	}
	go func() {
		if err := sched.Run(); err != nil {
			log.Error("scheduler stopped", "err", err)
		}
	}()
	mux := asynq.NewServeMux()
	wrap := func(fn func(context.Context) error) asynq.HandlerFunc {
		return func(ctx context.Context, _ *asynq.Task) error { return fn(ctx) }
	}
	mux.Handle(jobs.TypeApplyDosen, wrap(r.ApplyDosenChanges))
	mux.Handle(jobs.TypeDueReminder, wrap(r.DueReminders))
	mux.Handle(jobs.TypeRecompute, wrap(r.RecomputeAll))
	// run once at boot so a fresh environment is consistent
	_ = r.ApplyDosenChanges(ctx)
	srv := asynq.NewServer(redisOpt, asynq.Config{Concurrency: 4, Logger: nil})
	log.Info("worker started")
	if err := srv.Run(mux); err != nil {
		log.Error("worker stopped", "err", err)
		os.Exit(1)
	}
}
