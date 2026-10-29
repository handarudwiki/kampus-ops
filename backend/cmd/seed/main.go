// Command seed truncates all tenant data and loads the synthetic demo dataset.
// Usage: DATABASE_URL=... go run ./cmd/seed
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kampusops/backend/internal/platform/config"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/logger"
	"github.com/kampusops/backend/internal/seed"
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
	start := time.Now()
	sum, err := seed.Run(ctx, pool, time.Now())
	if err != nil {
		log.Error("seed failed", "err", err)
		os.Exit(1)
	}
	fmt.Printf("seed OK in %s: %s\n", time.Since(start).Round(time.Millisecond), sum)
	fmt.Println("demo accounts (password " + seed.Password + "): tenant UNIKA — super@kampusops.id, admin@unika.ac.id, baak@unika.ac.id,")
	fmt.Println("  0011223344 (kaprodi+dosen+dosen_wali), 0055667788 (dosen), keuangan@unika.ac.id, 2024110001, 2024110002, calon@mail.com")
	fmt.Println("  tenant POLTEK (Asia/Makassar) — admin@poltek.ac.id, baak@poltek.ac.id, keuangan@poltek.ac.id, 2024110001")
}
