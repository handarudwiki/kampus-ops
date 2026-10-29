package academic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/kampusops/backend/internal/audit"
	"github.com/kampusops/backend/internal/domain"
	"github.com/kampusops/backend/internal/notification"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/httpx"
	"github.com/kampusops/backend/internal/tenant"
)

type Service struct {
	pool    *pgxpool.Pool
	tenants *tenant.Store
	Now     func() time.Time
}

func NewService(pool *pgxpool.Pool, tenants *tenant.Store) *Service {
	return &Service{pool: pool, tenants: tenants, Now: time.Now}
}

// createAccount creates a login with a single role (password = initial password).
func (s *Service) createAccount(ctx context.Context, tx pgx.Tx, nama, email, identifier, password, role string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO users (tenant_id, nama, email, identifier, password_hash) VALUES ($1,$2,lower($3),$4,$5) RETURNING id`,
		appctx.TenantID(ctx), nama, email, identifier, string(hash)).Scan(&id)
	if db.IsUniqueViolation(err) {
		return "", httpx.Conflict("DUPLICATE", "Email atau identifier sudah dipakai akun lain")
	}
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `INSERT INTO user_role (tenant_id, user_id, role) VALUES ($1,$2,$3)`, appctx.TenantID(ctx), id, role)
	return id, err
}

// ===== Periode (FR-06) =====

type PeriodeInput struct {
	Kode    string         `json:"kode"`
	Tahun   int            `json:"tahun"`
	Tipe    string         `json:"tipe"`
	Status  string         `json:"status"`
	Tahapan domain.Tahapan `json:"tahapan"`
}

func periodeKode(tahun int, tipe string) string {
	n := map[string]int{"Ganjil": 1, "Genap": 2, "Pendek": 3}[tipe]
	return fmt.Sprintf("%d%d", tahun, n)
}

func validateTahapan(t domain.Tahapan) error {
	v := httpx.NewV()
	v.Date("tahapan.krs_mulai", t.KrsMulai).Date("tahapan.krs_selesai", t.KrsSelesai).
		Date("tahapan.kuliah_mulai", t.KuliahMulai).Date("tahapan.kuliah_selesai", t.KuliahSelesai).
		Date("tahapan.uts", t.UTS).Date("tahapan.uas", t.UAS).Date("tahapan.nilai_selesai", t.NilaiSelesai)
	v.Check(t.KrsMulai <= t.KrsSelesai, "tahapan.krs_selesai", "harus setelah krs_mulai")
	if t.RevisiMulai != nil && t.RevisiSelesai != nil {
		v.Date("tahapan.revisi_mulai", *t.RevisiMulai).Date("tahapan.revisi_selesai", *t.RevisiSelesai)
	}
	return v.Err()
}

func (s *Service) SavePeriode(ctx context.Context, id string, in PeriodeInput) (Periode, error) {
	if in.Status == "" {
		in.Status = "draft"
	}
	if err := httpx.NewV().Range("tahun", in.Tahun, 2000, 2100).OneOf("tipe", in.Tipe, "Ganjil", "Genap", "Pendek").
		OneOf("status", in.Status, "draft", "aktif", "selesai").Err(); err != nil {
		return Periode{}, err
	}
	if err := validateTahapan(in.Tahapan); err != nil {
		return Periode{}, err
	}
	if in.Kode == "" {
		in.Kode = periodeKode(in.Tahun, in.Tipe)
	}
	var before any
	if id != "" {
		b, err := GetPeriode(ctx, s.pool, id)
		if err != nil {
			return Periode{}, err
		}
		before = b
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		if id == "" {
			err = tx.QueryRow(ctx, `INSERT INTO periode (tenant_id, kode, tahun, tipe, status, tahapan) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
				appctx.TenantID(ctx), in.Kode, in.Tahun, in.Tipe, in.Status, in.Tahapan).Scan(&id)
		} else {
			_, err = tx.Exec(ctx, `UPDATE periode SET kode=$3, tahun=$4, tipe=$5, status=$6, tahapan=$7, updated_at=now() WHERE tenant_id=$1 AND id=$2`,
				appctx.TenantID(ctx), id, in.Kode, in.Tahun, in.Tipe, in.Status, in.Tahapan)
		}
		if db.IsUniqueViolation(err) && in.Status == "aktif" {
			return httpx.Rule("PERIOD_ALREADY_ACTIVE", "Sudah ada periode aktif; selesaikan periode tersebut terlebih dahulu")
		}
		if err != nil {
			return dup(err, "kode")
		}
		return audit.Write(ctx, tx, "periode", id, saveAksi(before), before, in)
	})
	if err != nil {
		return Periode{}, err
	}
	return GetPeriode(ctx, s.pool, id)
}

// ActivatePeriode implements FR-06: only one active period per tenant.
func (s *Service) ActivatePeriode(ctx context.Context, id string) (Periode, error) {
	p, err := GetPeriode(ctx, s.pool, id)
	if err != nil {
		return p, err
	}
	if p.Status == "aktif" {
		return p, nil
	}
	if cur, err := ActivePeriode(ctx, s.pool); err == nil {
		return p, httpx.Rule("PERIOD_ALREADY_ACTIVE", "Periode "+cur.Kode+" masih aktif").With(map[string]string{"active_periode_id": cur.ID})
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE periode SET status='aktif', updated_at=now() WHERE tenant_id=$1 AND id=$2`, appctx.TenantID(ctx), id)
		if db.IsUniqueViolation(err) {
			return httpx.Rule("PERIOD_ALREADY_ACTIVE", "Sudah ada periode aktif")
		}
		if err != nil {
			return err
		}
		return audit.Write(ctx, tx, "periode", id, "activate", map[string]string{"status": p.Status}, map[string]string{"status": "aktif"})
	})
	if err != nil {
		return p, err
	}
	return GetPeriode(ctx, s.pool, id)
}

// ===== Mahasiswa =====

type MahasiswaInput struct {
	NIM         string  `json:"nim"`
	NIK         *string `json:"nik"`
	Nama        string  `json:"nama"`
	Email       string  `json:"email"`
	ProdiID     string  `json:"prodi_id"`
	Angkatan    int     `json:"angkatan"`
	KurikulumID string  `json:"kurikulum_id"`
	DosenWaliID *string `json:"dosen_wali_id"`
}

func (s *Service) SaveMahasiswa(ctx context.Context, id string, in MahasiswaInput) (Mahasiswa, error) {
	if err := httpx.NewV().Required("nim", in.NIM).Required("nama", in.Nama).Email("email", in.Email).
		Range("angkatan", in.Angkatan, 1990, 2100).Err(); err != nil {
		return Mahasiswa{}, err
	}
	if err := MustExist(ctx, s.pool, "prodi", "prodi_id", in.ProdiID); err != nil {
		return Mahasiswa{}, err
	}
	if err := MustExist(ctx, s.pool, "kurikulum", "kurikulum_id", in.KurikulumID); err != nil {
		return Mahasiswa{}, err
	}
	if in.DosenWaliID != nil && *in.DosenWaliID != "" {
		if err := MustExist(ctx, s.pool, "dosen", "dosen_wali_id", *in.DosenWaliID); err != nil {
			return Mahasiswa{}, err
		}
	} else {
		in.DosenWaliID = nil
	}
	if in.NIK != nil && strings.TrimSpace(*in.NIK) == "" {
		in.NIK = nil
	}
	var before any
	if id != "" {
		b, err := GetMahasiswa(ctx, s.pool, id)
		if err != nil {
			return Mahasiswa{}, err
		}
		before = b
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		if id == "" {
			ap, err := ActivePeriode(ctx, tx)
			if err != nil {
				return err
			}
			userID, err := s.createAccount(ctx, tx, in.Nama, in.Email, in.NIM, in.NIM, "mahasiswa")
			if err != nil {
				return err
			}
			err = tx.QueryRow(ctx, `INSERT INTO mahasiswa (tenant_id, user_id, nim, nik, nama, email, prodi_id, angkatan, kurikulum_id, dosen_wali_id)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`, tid, userID, in.NIM, in.NIK, in.Nama, in.Email, in.ProdiID, in.Angkatan, in.KurikulumID, in.DosenWaliID).Scan(&id)
			if err != nil {
				return dup(err, "nim")
			}
			if _, err := tx.Exec(ctx, `INSERT INTO riwayat_status (tenant_id, mahasiswa_id, periode_id, status, alasan, created_by) VALUES ($1,$2,$3,'aktif','Mahasiswa baru',$4)`,
				tid, id, ap.ID, appctx.UserID(ctx)); err != nil {
				return err
			}
		} else {
			_, err := tx.Exec(ctx, `UPDATE mahasiswa SET nim=$3, nik=$4, nama=$5, email=$6, prodi_id=$7, angkatan=$8, kurikulum_id=$9, dosen_wali_id=$10, updated_at=now()
				WHERE tenant_id=$1 AND id=$2`, tid, id, in.NIM, in.NIK, in.Nama, in.Email, in.ProdiID, in.Angkatan, in.KurikulumID, in.DosenWaliID)
			if err != nil {
				return dup(err, "nim")
			}
		}
		return audit.Write(ctx, tx, "mahasiswa", id, saveAksi(before), before, in)
	})
	if err != nil {
		return Mahasiswa{}, err
	}
	return GetMahasiswa(ctx, s.pool, id)
}

// ChangeStatus implements status transitions with the EC-08 cuti rules: transition must
// be valid, cuti is limited (tenant config, default 2), and a cuti replaces the unpaid
// UKT bill with a cuti fee. Every change is a new riwayat_status row for the active period.
func (s *Service) ChangeStatus(ctx context.Context, mhsID, to, alasan string) ([]RiwayatStatus, error) {
	if !domain.IsValidStatus(to) {
		return nil, httpx.ValidationField("status", "status tidak dikenal")
	}
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return nil, err
	}
	m, err := GetMahasiswa(ctx, s.pool, mhsID)
	if err != nil {
		return nil, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		// serialise status changes per student
		if _, err := tx.Exec(ctx, `SELECT 1 FROM mahasiswa WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, mhsID, tid); err != nil {
			return err
		}
		ap, err := ActivePeriode(ctx, tx)
		if err != nil {
			return err
		}
		from, err := CurrentStatus(ctx, tx, mhsID)
		if err != nil {
			return err
		}
		if !domain.CanTransition(from, to) {
			return httpx.ValidationField("status", fmt.Sprintf("transisi status %s → %s tidak diizinkan", from, to))
		}
		if to == domain.StatusCuti {
			used, err := CutiCount(ctx, tx, mhsID, ap.ID)
			if err != nil {
				return err
			}
			if !domain.CutiAllowed(used, t.Rules.CutiMax) {
				return httpx.Rule("CUTI_LIMIT_REACHED", fmt.Sprintf("Batas cuti %d semester sudah tercapai", t.Rules.CutiMax)).
					With(map[string]int{"cuti_terpakai": used, "cuti_maks": t.Rules.CutiMax})
			}
		}
		var a *string
		if strings.TrimSpace(alasan) != "" {
			a = &alasan
		}
		if _, err := tx.Exec(ctx, `INSERT INTO riwayat_status (tenant_id, mahasiswa_id, periode_id, status, alasan, created_by) VALUES ($1,$2,$3,$4,$5,$6)`,
			tid, mhsID, ap.ID, to, a, appctx.UserID(ctx)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE mahasiswa SET status=$3, updated_at=now() WHERE id=$1 AND tenant_id=$2`, mhsID, tid, to); err != nil {
			return err
		}
		if to == domain.StatusCuti {
			if err := s.replaceBillWithCuti(ctx, tx, m, ap, t.Rules.BiayaCuti); err != nil {
				return err
			}
		}
		if err := audit.Write(ctx, tx, "mahasiswa", mhsID, "status_change", map[string]string{"status": from}, map[string]any{"status": to, "alasan": alasan, "periode": ap.Kode}); err != nil {
			return err
		}
		if m.UserID != nil {
			return notification.Send(ctx, tx, *m.UserID, "Status akademik berubah", fmt.Sprintf("Status Anda berubah dari %s menjadi %s.", from, to), notification.Link("/akademik/profil"))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return Riwayat(ctx, s.pool, mhsID)
}

// replaceBillWithCuti: EC-08 — an unpaid UKT bill of the active period is replaced by a
// cuti fee bill (a partially paid bill is kept for manual refund handling).
func (s *Service) replaceBillWithCuti(ctx context.Context, tx pgx.Tx, m Mahasiswa, ap Periode, fee int64) error {
	tid := appctx.TenantID(ctx)
	var before []byte
	var billID string
	err := tx.QueryRow(ctx, `DELETE FROM tagihan t WHERE tenant_id=$1 AND mahasiswa_id=$2 AND periode_id=$3 AND terbayar=0
		AND NOT EXISTS (SELECT 1 FROM pembayaran p WHERE p.tagihan_id=t.id) AND jenis='ukt' RETURNING id, to_jsonb(t)`, tid, m.ID, ap.ID).Scan(&billID, &before)
	if err != nil && !db.IsNoRows(err) {
		return err
	}
	if billID != "" {
		if err := audit.Write(ctx, tx, "tagihan", billID, "replaced_by_cuti", before, nil); err != nil {
			return err
		}
	}
	jt := ap.Tahapan.KrsSelesai
	if jt == "" {
		jt = domain.LocalDate(s.Now().AddDate(0, 0, 14), time.UTC)
	}
	var newID string
	err = tx.QueryRow(ctx, `INSERT INTO tagihan (tenant_id, mahasiswa_id, periode_id, jenis, jumlah, jatuh_tempo) VALUES ($1,$2,$3,'cuti',$4,$5)
		ON CONFLICT DO NOTHING RETURNING id`, tid, m.ID, ap.ID, fee, jt).Scan(&newID)
	if db.IsNoRows(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return audit.Write(ctx, tx, "tagihan", newID, "create", nil, map[string]any{"jenis": "cuti", "jumlah": fee})
}

// MasaStudiWarning implements EC-09: students with ≤ N semesters left (or exceeded).
type MasaStudiRow struct {
	Mahasiswa
	SisaSemester int    `json:"sisa_semester"`
	Flag         string `json:"flag"`
}

func (s *Service) MasaStudiWarnings(ctx context.Context, prodiScope *string) ([]MasaStudiRow, error) {
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return nil, err
	}
	ap, err := ActivePeriode(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	scope := ""
	if prodiScope != nil {
		scope = *prodiScope
	}
	// coarse SQL pre-filter (semester_ke ≥ maks - warn - max cuti), exact rule in Go
	ms, err := QueryMahasiswa(ctx, s.pool, `m.status IN ('aktif','cuti','non_aktif') AND ($2='' OR m.prodi_id::text=$2)
		AND (($3 - m.angkatan) * 2 + 2) >= p.masa_studi_maks - $4`, `ORDER BY m.angkatan, m.nim`, scope, ap.Tahun, t.Rules.MasaStudiWarningSisa)
	if err != nil {
		return nil, err
	}
	out := []MasaStudiRow{}
	for _, m := range ms {
		cuti, err := CutiCount(ctx, s.pool, m.ID, "")
		if err != nil {
			return nil, err
		}
		_, sisa := domain.MasaStudi(m.SemesterKe, cuti, m.MasaStudiMaks)
		if flag := domain.MasaStudiFlag(sisa, t.Rules.MasaStudiWarningSisa); flag != "" {
			out = append(out, MasaStudiRow{Mahasiswa: m, SisaSemester: sisa, Flag: flag})
		}
	}
	return out, nil
}
