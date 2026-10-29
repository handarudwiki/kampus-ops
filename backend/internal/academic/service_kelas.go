package academic

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/kampusops/backend/internal/audit"
	"github.com/kampusops/backend/internal/domain"
	"github.com/kampusops/backend/internal/notification"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/httpx"
)

type KelasInput struct {
	PeriodeID string  `json:"periode_id"`
	MkID      string  `json:"mk_id"`
	Nama      string  `json:"nama"`
	Sks       int     `json:"sks"`
	DosenID   *string `json:"dosen_id"`
	RuangID   *string `json:"ruang_id"`
	Slot      Slot    `json:"slot"`
	Kuota     int     `json:"kuota"`
}

func emptyToNil(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

// checkKaprodiScope: a kaprodi may only manage classes / courses of their own prodi.
func checkKaprodiScope(ctx context.Context, q db.Querier, mkID string) error {
	p, _ := appctx.PrincipalFrom(ctx)
	if p.ActiveRole != "kaprodi" {
		return nil
	}
	scope := p.ScopeProdi("kaprodi")
	if scope == nil {
		return nil
	}
	var prodi string
	if err := q.QueryRow(ctx, `SELECT ku.prodi_id FROM mata_kuliah mk JOIN kurikulum ku ON ku.id=mk.kurikulum_id WHERE mk.tenant_id=$1 AND mk.id=$2`,
		appctx.TenantID(ctx), mkID).Scan(&prodi); err != nil {
		return err
	}
	if prodi != *scope {
		return httpx.Forbidden("FORBIDDEN_ROLE", "Mata kuliah bukan milik prodi Anda")
	}
	return nil
}

// SaveKelas creates/updates a class with FR-15 clash detection (409 with conflicts).
func (s *Service) SaveKelas(ctx context.Context, id string, in KelasInput) (Kelas, error) {
	in.DosenID, in.RuangID = emptyToNil(in.DosenID), emptyToNil(in.RuangID)
	v := httpx.NewV().Required("nama", in.Nama).Range("slot.hari", in.Slot.Hari, 1, 6).Range("kuota", in.Kuota, 1, 1000)
	if _, _, err := in.Slot.Minutes(); err != nil {
		v.Check(false, "slot", err.Error())
	}
	if err := v.Err(); err != nil {
		return Kelas{}, err
	}
	var before *Kelas
	if id != "" {
		b, err := GetKelas(ctx, s.pool, id)
		if err != nil {
			return Kelas{}, err
		}
		before = &b
		if in.PeriodeID == "" {
			in.PeriodeID = b.PeriodeID
		}
		if in.MkID == "" {
			in.MkID = b.MkID
		}
		if in.Kuota < b.Terisi {
			return Kelas{}, httpx.ValidationField("kuota", fmt.Sprintf("tidak boleh kurang dari peserta terdaftar (%d)", b.Terisi))
		}
	}
	if err := MustExist(ctx, s.pool, "periode", "periode_id", in.PeriodeID); err != nil {
		return Kelas{}, err
	}
	if err := MustExist(ctx, s.pool, "mata_kuliah", "mk_id", in.MkID); err != nil {
		return Kelas{}, err
	}
	if in.DosenID != nil {
		if err := MustExist(ctx, s.pool, "dosen", "dosen_id", *in.DosenID); err != nil {
			return Kelas{}, err
		}
	}
	if in.RuangID != nil {
		var ok bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ruang WHERE tenant_id=$1 AND id::text=$2)`, appctx.TenantID(ctx), *in.RuangID).Scan(&ok); err != nil {
			return Kelas{}, err
		}
		if !ok {
			return Kelas{}, httpx.ValidationField("ruang_id", "tidak ditemukan")
		}
	}
	if err := checkKaprodiScope(ctx, s.pool, in.MkID); err != nil {
		return Kelas{}, err
	}
	if in.Sks == 0 {
		if err := s.pool.QueryRow(ctx, `SELECT sks FROM mata_kuliah WHERE tenant_id=$1 AND id=$2`, appctx.TenantID(ctx), in.MkID).Scan(&in.Sks); err != nil {
			return Kelas{}, err
		}
	}
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return Kelas{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		// lock the period's schedule to make clash detection race-free
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "kelas-schedule:"+in.PeriodeID); err != nil {
			return err
		}
		others, err := QueryKelas(ctx, tx, `k.periode_id=$2 AND k.status='dibuka' AND k.hari=$3`, in.PeriodeID, in.Slot.Hari)
		if err != nil {
			return err
		}
		sched := make([]ScheduledClass, 0, len(others))
		byID := map[string]Kelas{}
		for _, o := range others {
			sched = append(sched, o.Scheduled())
			byID[o.ID] = o
		}
		room, lect := FindClashes(ScheduledClass{ID: id, RuangID: in.RuangID, DosenID: in.DosenID, Slot: in.Slot}, sched)
		if len(room) > 0 || len(lect) > 0 {
			code, msg, ids := "SCHEDULE_ROOM_CLASH", "Ruang sudah dipakai kelas lain pada slot ini", room
			if len(room) == 0 {
				code, msg, ids = "SCHEDULE_LECTURER_CLASH", "Dosen sudah mengajar kelas lain pada slot ini", lect
			}
			conflicts := []Kelas{}
			for _, cid := range append(append([]string{}, room...), lect...) {
				conflicts = append(conflicts, byID[cid])
			}
			_ = ids
			return httpx.Conflict(code, msg).With(map[string]any{"conflicts": dedupKelas(conflicts)})
		}
		if id == "" {
			err = tx.QueryRow(ctx, `INSERT INTO kelas (tenant_id, periode_id, mk_id, nama, sks, dosen_id, ruang_id, hari, jam_mulai, jam_selesai, kuota)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`, tid, in.PeriodeID, in.MkID, in.Nama, in.Sks, in.DosenID, in.RuangID,
				in.Slot.Hari, in.Slot.JamMulai, in.Slot.JamSelesai, in.Kuota).Scan(&id)
			if err != nil {
				return dup(err, "nama")
			}
			for i, k := range t.Rules.KomponenDefault {
				if _, err := tx.Exec(ctx, `INSERT INTO komponen_nilai (tenant_id, kelas_id, key, label, bobot_persen, urutan) VALUES ($1,$2,$3,$4,$5,$6)`,
					tid, id, k.Key, k.Label, k.BobotPersen, i); err != nil {
					return err
				}
			}
			if in.DosenID != nil {
				if _, err := tx.Exec(ctx, `INSERT INTO kelas_dosen (tenant_id, kelas_id, dosen_id, efektif_mulai) VALUES ($1,$2,$3,'2000-01-01')`, tid, id, *in.DosenID); err != nil {
					return err
				}
			}
		} else {
			_, err = tx.Exec(ctx, `UPDATE kelas SET periode_id=$3, mk_id=$4, nama=$5, sks=$6, dosen_id=$7, ruang_id=$8, hari=$9, jam_mulai=$10, jam_selesai=$11, kuota=$12, updated_at=now()
				WHERE tenant_id=$1 AND id=$2`, tid, id, in.PeriodeID, in.MkID, in.Nama, in.Sks, in.DosenID, in.RuangID, in.Slot.Hari, in.Slot.JamMulai, in.Slot.JamSelesai, in.Kuota)
			if err != nil {
				if db.IsCheckViolation(err) {
					return httpx.ValidationField("kuota", "tidak boleh kurang dari peserta terdaftar")
				}
				return dup(err, "nama")
			}
			// direct edit of the lecturer (not a mid-semester change) resets history
			if before != nil && !samePtr(before.DosenID, in.DosenID) {
				if _, err := tx.Exec(ctx, `DELETE FROM kelas_dosen WHERE tenant_id=$1 AND kelas_id=$2`, tid, id); err != nil {
					return err
				}
				if in.DosenID != nil {
					if _, err := tx.Exec(ctx, `INSERT INTO kelas_dosen (tenant_id, kelas_id, dosen_id, efektif_mulai) VALUES ($1,$2,$3,'2000-01-01')`, tid, id, *in.DosenID); err != nil {
						return err
					}
				}
			}
		}
		var b any
		if before != nil {
			b = before
		}
		return audit.Write(ctx, tx, "kelas", id, saveAksi(b), b, in)
	})
	if err != nil {
		return Kelas{}, err
	}
	return GetKelas(ctx, s.pool, id)
}

func samePtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func dedupKelas(ks []Kelas) []Kelas {
	seen := map[string]bool{}
	out := []Kelas{}
	for _, k := range ks {
		if !seen[k.ID] {
			seen[k.ID] = true
			out = append(out, k)
		}
	}
	return out
}

// CancelKelas implements EC-13: the class is cancelled, active KRS items are cancelled,
// seats/SKS released, students notified, and each affected student receives an approved
// krs_susulan dispensation so they can pick a replacement class even if the window closed.
func (s *Service) CancelKelas(ctx context.Context, id, alasan string) (Kelas, error) {
	if err := httpx.NewV().Required("alasan", alasan).Err(); err != nil {
		return Kelas{}, err
	}
	k, err := GetKelas(ctx, s.pool, id)
	if err != nil {
		return k, err
	}
	if k.Status == "dibatalkan" {
		return k, httpx.Rule("VALIDATION_ERROR", "Kelas sudah dibatalkan")
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		if _, err := tx.Exec(ctx, `UPDATE kelas SET status='dibatalkan', terisi=0, updated_at=now() WHERE tenant_id=$1 AND id=$2`, tid, id); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `UPDATE krs_detail d SET status='dibatalkan', updated_at=now() FROM krs, mahasiswa m
			WHERE d.tenant_id=$1 AND d.kelas_id=$2 AND d.status='aktif' AND krs.id=d.krs_id AND m.id=krs.mahasiswa_id
			RETURNING d.id, krs.id, m.id, m.user_id, krs.periode_id`, tid, id)
		if err != nil {
			return err
		}
		type aff struct{ detail, krs, mhs, periode string; user *string }
		var affected []aff
		for rows.Next() {
			var a aff
			if err := rows.Scan(&a.detail, &a.krs, &a.mhs, &a.user, &a.periode); err != nil {
				rows.Close()
				return err
			}
			affected = append(affected, a)
		}
		rows.Close()
		for _, a := range affected {
			if _, err := tx.Exec(ctx, `UPDATE krs SET total_sks = (SELECT coalesce(sum(sks),0) FROM krs_detail WHERE krs_id=$2 AND status='aktif'), updated_at=now()
				WHERE tenant_id=$1 AND id=$2`, tid, a.krs); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO dispensasi (tenant_id, mahasiswa_id, periode_id, jenis, alasan, status, disetujui_oleh, created_by, decided_at)
				VALUES ($1,$2,$3,'krs_susulan',$4,'disetujui',$5,$5,now())`, tid, a.mhs, a.periode, "Kelas "+k.MkNama+" "+k.Nama+" dibatalkan: "+alasan, appctx.UserID(ctx)); err != nil {
				return err
			}
			if err := audit.Write(ctx, tx, "krs_detail", a.detail, "cancel_by_kelas", map[string]string{"status": "aktif"}, map[string]string{"status": "dibatalkan"}); err != nil {
				return err
			}
			if a.user != nil {
				if err := notification.Send(ctx, tx, *a.user, "Kelas dibatalkan",
					fmt.Sprintf("Kelas %s %s dibatalkan (%s). SKS dikembalikan; silakan pilih kelas pengganti.", k.MkNama, k.Nama, alasan),
					notification.Link("/krs")); err != nil {
					return err
				}
			}
		}
		return audit.Write(ctx, tx, "kelas", id, "cancel", k, map[string]any{"status": "dibatalkan", "alasan": alasan, "krs_terdampak": len(affected)})
	})
	if err != nil {
		return Kelas{}, err
	}
	return GetKelas(ctx, s.pool, id)
}

// GantiDosen implements EC-12: a new lecturer from an effective date. Attendance already
// recorded keeps the old lecturer (pertemuan.dosen_id); grading rights follow DosenOn().
func (s *Service) GantiDosen(ctx context.Context, id, dosenID, efektif string) (Kelas, error) {
	if err := httpx.NewV().Required("dosen_id", dosenID).Date("efektif", efektif).Err(); err != nil {
		return Kelas{}, err
	}
	if err := MustExist(ctx, s.pool, "dosen", "dosen_id", dosenID); err != nil {
		return Kelas{}, err
	}
	k, err := GetKelas(ctx, s.pool, id)
	if err != nil {
		return k, err
	}
	loc := appctx.Tenant(ctx).Loc
	today := domain.LocalDate(s.Now(), loc)
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		// close the open assignment the day before the effective date
		if _, err := tx.Exec(ctx, `UPDATE kelas_dosen SET efektif_selesai = ($3::date - 1), updated_at=now()
			WHERE tenant_id=$1 AND kelas_id=$2 AND (efektif_selesai IS NULL OR efektif_selesai >= $3::date) AND efektif_mulai < $3::date`, tid, id, efektif); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM kelas_dosen WHERE tenant_id=$1 AND kelas_id=$2 AND efektif_mulai >= $3::date`, tid, id, efektif); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO kelas_dosen (tenant_id, kelas_id, dosen_id, efektif_mulai) VALUES ($1,$2,$3,$4)`, tid, id, dosenID, efektif); err != nil {
			return err
		}
		if efektif <= today {
			if _, err := tx.Exec(ctx, `UPDATE kelas SET dosen_id=$3, updated_at=now() WHERE tenant_id=$1 AND id=$2`, tid, id, dosenID); err != nil {
				return err
			}
		}
		return audit.Write(ctx, tx, "kelas", id, "ganti_dosen", map[string]any{"dosen_id": k.DosenID}, map[string]any{"dosen_id": dosenID, "efektif": efektif})
	})
	if err != nil {
		return Kelas{}, err
	}
	return GetKelas(ctx, s.pool, id)
}

// ApplyDueLecturerChanges syncs kelas.dosen_id with assignments whose effective date has
// arrived (run by the worker).
func ApplyDueLecturerChanges(ctx context.Context, q db.Querier, today string) (int64, error) {
	tag, err := q.Exec(ctx, `UPDATE kelas k SET dosen_id = kd.dosen_id, updated_at=now() FROM kelas_dosen kd
		WHERE kd.kelas_id = k.id AND k.tenant_id=$1 AND kd.efektif_mulai <= $2::date AND (kd.efektif_selesai IS NULL OR kd.efektif_selesai >= $2::date)
		AND k.dosen_id IS DISTINCT FROM kd.dosen_id`, appctx.TenantID(ctx), today)
	return tag.RowsAffected(), err
}
