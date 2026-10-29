// Package jobs holds background jobs run by cmd/worker (via Asynq): applying
// lecturer changes whose effective date arrived (EC-12), bill due-date reminders
// (FR-33) and a nightly IPK cache recompute (EC-10).
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kampusops/backend/internal/academic"
	"github.com/kampusops/backend/internal/domain"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/tenant"
)

const (
	TypeApplyDosen  = "kelas:apply-dosen-change"
	TypeDueReminder = "tagihan:due-reminder"
	TypeRecompute   = "mahasiswa:recompute-ipk"
)

type Runner struct {
	Pool    *pgxpool.Pool
	Tenants *tenant.Store
	Now     func() time.Time
}

func (r *Runner) forEachTenant(ctx context.Context, fn func(ctx context.Context, t tenant.Full) error) error {
	ts, err := r.Tenants.List(ctx)
	if err != nil {
		return err
	}
	for _, t := range ts {
		if err := fn(appctx.WithTenant(ctx, t.Info()), t); err != nil {
			return fmt.Errorf("tenant %s: %w", t.Code, err)
		}
	}
	return nil
}

func (r *Runner) ApplyDosenChanges(ctx context.Context) error {
	return r.forEachTenant(ctx, func(ctx context.Context, t tenant.Full) error {
		n, err := academic.ApplyDueLecturerChanges(ctx, r.Pool, domain.LocalDate(r.Now(), t.Loc))
		if n > 0 {
			slog.Info("lecturer changes applied", "tenant", t.Code, "kelas", n)
		}
		return err
	})
}

// DueReminders notifies students of unpaid bills due within 3 days (once per bill per day).
func (r *Runner) DueReminders(ctx context.Context) error {
	return r.forEachTenant(ctx, func(ctx context.Context, t tenant.Full) error {
		today := domain.LocalDate(r.Now(), t.Loc)
		tag, err := r.Pool.Exec(ctx, `INSERT INTO notifikasi (tenant_id, user_id, judul, isi, link)
			SELECT t.tenant_id, m.user_id, 'Tagihan jatuh tempo', 'Tagihan ' || t.jenis || ' jatuh tempo ' || to_char(t.jatuh_tempo,'DD-MM-YYYY') || '.', '/tagihan'
			FROM tagihan t JOIN mahasiswa m ON m.id=t.mahasiswa_id
			WHERE t.tenant_id=$1 AND t.status<>'lunas' AND m.user_id IS NOT NULL AND t.jatuh_tempo BETWEEN $2::date AND $2::date + 3
			AND NOT EXISTS (SELECT 1 FROM notifikasi n WHERE n.user_id=m.user_id AND n.judul='Tagihan jatuh tempo' AND n.created_at::date = now()::date)`,
			t.ID, today)
		if err == nil && tag.RowsAffected() > 0 {
			slog.Info("due reminders sent", "tenant", t.Code, "count", tag.RowsAffected())
		}
		return err
	})
}

func (r *Runner) RecomputeAll(ctx context.Context) error {
	return r.forEachTenant(ctx, func(ctx context.Context, t tenant.Full) error {
		rows, err := r.Pool.Query(ctx, `SELECT id FROM mahasiswa WHERE tenant_id=$1`, t.ID)
		if err != nil {
			return err
		}
		var ids []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()
		for _, id := range ids {
			if err := academic.RecomputeStudent(ctx, r.Pool, t.Rules, id); err != nil {
				return err
			}
		}
		slog.Info("ipk recomputed", "tenant", t.Code, "mahasiswa", len(ids))
		return nil
	})
}
