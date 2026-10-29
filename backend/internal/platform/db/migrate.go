package db

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type migration struct {
	version int64
	name    string
	up      string
}

// Migrate applies pending *.up.sql files from fsys in version order. It uses the
// same bookkeeping table shape as golang-migrate (schema_migrations(version, dirty))
// and a Postgres advisory lock so api and worker can boot concurrently.
func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(727274)`); err != nil {
		return err
	}
	defer func() { _, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock(727274)`) }()

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version bigint PRIMARY KEY, dirty boolean NOT NULL)`); err != nil {
		return err
	}
	var current int64
	var dirty bool
	err = conn.QueryRow(ctx, `SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&current, &dirty)
	if err != nil && !IsNoRows(err) {
		return err
	}
	if dirty {
		return fmt.Errorf("database is dirty at version %d; fix manually", current)
	}
	files, err := fs.Glob(fsys, "*.up.sql")
	if err != nil {
		return err
	}
	var ms []migration
	for _, f := range files {
		v, err := strconv.ParseInt(strings.SplitN(f, "_", 2)[0], 10, 64)
		if err != nil {
			return fmt.Errorf("bad migration name %s", f)
		}
		b, err := fs.ReadFile(fsys, f)
		if err != nil {
			return err
		}
		ms = append(ms, migration{version: v, name: f, up: string(b)})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].version < ms[j].version })
	for _, m := range ms {
		if m.version <= current {
			continue
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, m.up); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations`); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(version, dirty) VALUES ($1, false)`, m.version); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		slog.Info("migration applied", "file", m.name)
	}
	return nil
}
