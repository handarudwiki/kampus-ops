package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/kampusops/backend/internal/academic"
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

type Gelombang struct {
	ID            string `json:"id"`
	Nama          string `json:"nama"`
	Mulai         string `json:"mulai"`
	Selesai       string `json:"selesai"`
	Kuota         int    `json:"kuota"`
	Terisi        int    `json:"terisi"`
	BiayaFormulir int64  `json:"biaya_formulir"`
}

type Pendaftar struct {
	ID               string   `json:"id"`
	NoDaftar         string   `json:"no_daftar"`
	Nama             string   `json:"nama"`
	Email            string   `json:"email"`
	NIK              string   `json:"nik"`
	GelombangID      string   `json:"gelombang_id"`
	GelombangNama    string   `json:"gelombang_nama"`
	ProdiPilihanID   string   `json:"prodi_pilihan_id"`
	ProdiPilihanNama string   `json:"prodi_pilihan_nama"`
	Status           string   `json:"status"`
	Berkas           []Berkas `json:"berkas"`
	NIM              *string  `json:"nim"`
	CreatedAt        string   `json:"created_at"`
}

const gelSel = `SELECT id, nama, mulai::text, selesai::text, kuota, terisi, biaya_formulir FROM gelombang `

func (s *Service) ListGelombang(ctx context.Context, onlyOpen bool) ([]Gelombang, error) {
	where := `WHERE tenant_id=$1`
	args := []any{appctx.TenantID(ctx)}
	if onlyOpen {
		today := domain.LocalDate(s.Now(), appctx.Tenant(ctx).Loc)
		where += ` AND mulai <= $2::date AND selesai >= $2::date AND terisi < kuota`
		args = append(args, today)
	}
	rows, err := s.pool.Query(ctx, gelSel+where+` ORDER BY mulai DESC`, args...)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Gelombang])
	if out == nil {
		out = []Gelombang{}
	}
	return out, err
}

func (s *Service) CreateGelombang(ctx context.Context, in Gelombang) (Gelombang, error) {
	if err := httpx.NewV().Required("nama", in.Nama).Date("mulai", in.Mulai).Date("selesai", in.Selesai).Check(in.Mulai <= in.Selesai, "selesai", "harus setelah mulai").
		Range("kuota", in.Kuota, 1, 100000).Check(in.BiayaFormulir >= 0, "biaya_formulir", "tidak boleh negatif").Err(); err != nil {
		return Gelombang{}, err
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `INSERT INTO gelombang (tenant_id, nama, mulai, selesai, kuota, biaya_formulir) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
			appctx.TenantID(ctx), in.Nama, in.Mulai, in.Selesai, in.Kuota, in.BiayaFormulir).Scan(&in.ID)
		if db.IsUniqueViolation(err) {
			return httpx.Conflict("DUPLICATE", "Nama gelombang sudah dipakai")
		}
		if err != nil {
			return err
		}
		return audit.Write(ctx, tx, "gelombang", in.ID, "create", nil, in)
	})
	return in, err
}

const pendSel = `SELECT p.id, p.no_daftar, p.nama, p.email, p.nik, p.gelombang_id, g.nama, p.prodi_pilihan_id, pr.nama, p.status, p.berkas, p.nim, iso(p.created_at)
	FROM pendaftar p JOIN gelombang g ON g.id=p.gelombang_id JOIN prodi pr ON pr.id=p.prodi_pilihan_id `

func scanPend(row pgx.Row) (Pendaftar, error) {
	var p Pendaftar
	var raw []byte
	err := row.Scan(&p.ID, &p.NoDaftar, &p.Nama, &p.Email, &p.NIK, &p.GelombangID, &p.GelombangNama, &p.ProdiPilihanID, &p.ProdiPilihanNama, &p.Status, &raw, &p.NIM, &p.CreatedAt)
	_ = json.Unmarshal(raw, &p.Berkas)
	if p.Berkas == nil {
		p.Berkas = []Berkas{}
	}
	return p, err
}

func (s *Service) getPendaftar(ctx context.Context, q db.Querier, id string) (Pendaftar, error) {
	p, err := scanPend(q.QueryRow(ctx, pendSel+`WHERE p.tenant_id=$1 AND p.id=$2`, appctx.TenantID(ctx), id))
	if err != nil {
		return p, httpx.NotFound("Pendaftar tidak ditemukan")
	}
	return p, nil
}

func (s *Service) ListPendaftar(ctx context.Context, status, gelombangID string, pg httpx.Page) ([]Pendaftar, int, error) {
	where := `WHERE p.tenant_id=$1 AND ($2='' OR p.status=$2) AND ($3='' OR p.gelombang_id::text=$3) AND (p.nama ILIKE $4 OR p.no_daftar ILIKE $4 OR p.email ILIKE $4 OR coalesce(p.nim,'') ILIKE $4)`
	args := []any{appctx.TenantID(ctx), status, gelombangID, pg.Like()}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM pendaftar p `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, pendSel+where+` ORDER BY p.created_at DESC LIMIT $5 OFFSET $6`, append(args, pg.PerPage, pg.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Pendaftar{}
	for rows.Next() {
		p, err := scanPend(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

// PublicOptions: open waves + prodi for the public registration page.
func (s *Service) PublicOptions(ctx context.Context, tenantCode string) (map[string]any, error) {
	t, err := s.tenants.ByCode(ctx, tenantCode)
	if err != nil {
		return nil, httpx.NotFound("Kampus tidak ditemukan")
	}
	ctx = appctx.WithTenant(ctx, t.Info())
	gs, err := s.ListGelombang(ctx, true)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT p.id, p.fakultas_id, f.nama, p.kode, p.nama, p.jenjang, p.masa_studi_maks FROM prodi p JOIN fakultas f ON f.id=p.fakultas_id
		WHERE p.tenant_id=$1 ORDER BY p.nama`, t.ID)
	if err != nil {
		return nil, err
	}
	ps, err := pgx.CollectRows(rows, pgx.RowToStructByPos[academic.Prodi])
	if ps == nil {
		ps = []academic.Prodi{}
	}
	return map[string]any{"gelombang": gs, "prodi": ps}, err
}

type RegisterInput struct {
	TenantCode     string `json:"tenant_code"`
	Nama           string `json:"nama"`
	Email          string `json:"email"`
	NIK            string `json:"nik"`
	GelombangID    string `json:"gelombang_id"`
	ProdiPilihanID string `json:"prodi_pilihan_id"`
	Password       string `json:"password"`
}

// Register (no auth): creates the applicant account, the pendaftar row (status daftar)
// and the form-fee bill with a VA number.
func (s *Service) Register(ctx context.Context, in RegisterInput) (Pendaftar, error) {
	v := httpx.NewV().Required("tenant_code", in.TenantCode).Required("nama", in.Nama).Email("email", in.Email).
		Check(ValidNIK(in.NIK), "nik", "NIK harus 16 digit").UUID("gelombang_id", in.GelombangID).UUID("prodi_pilihan_id", in.ProdiPilihanID).
		Check(len(in.Password) >= 8, "password", "minimal 8 karakter")
	if err := v.Err(); err != nil {
		return Pendaftar{}, err
	}
	t, err := s.tenants.ByCode(ctx, in.TenantCode)
	if err != nil {
		return Pendaftar{}, httpx.NotFound("Kampus tidak ditemukan")
	}
	ctx = appctx.WithTenant(ctx, t.Info())
	if err := academic.MustExist(ctx, s.pool, "prodi", "prodi_pilihan_id", in.ProdiPilihanID); err != nil {
		return Pendaftar{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return Pendaftar{}, err
	}
	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := t.ID
		var g Gelombang
		if err := tx.QueryRow(ctx, gelSel+`WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tid, in.GelombangID).
			Scan(&g.ID, &g.Nama, &g.Mulai, &g.Selesai, &g.Kuota, &g.Terisi, &g.BiayaFormulir); err != nil {
			return httpx.ValidationField("gelombang_id", "tidak ditemukan")
		}
		if !GelombangOpen(g.Mulai, g.Selesai, s.Now(), t.Loc) {
			return httpx.ValidationField("gelombang_id", "gelombang pendaftaran tidak sedang dibuka")
		}
		tag, err := tx.Exec(ctx, `UPDATE gelombang SET terisi=terisi+1, updated_at=now() WHERE id=$1 AND tenant_id=$2 AND terisi < kuota`, g.ID, tid)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return httpx.ValidationField("gelombang_id", "kuota gelombang sudah penuh")
		}
		var userID string
		err = tx.QueryRow(ctx, `INSERT INTO users (tenant_id, nama, email, password_hash) VALUES ($1,$2,lower($3),$4) RETURNING id`, tid, in.Nama, in.Email, string(hash)).Scan(&userID)
		if db.IsUniqueViolation(err) {
			return httpx.ValidationField("email", "email sudah terdaftar")
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO user_role (tenant_id, user_id, role) VALUES ($1,$2,'calon_mahasiswa')`, tid, userID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "pmb-no:"+tid); err != nil {
			return err
		}
		year := s.Now().In(t.Loc).Year()
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM pendaftar WHERE tenant_id=$1 AND no_daftar LIKE $2`, tid, fmt.Sprintf("PMB%d%%", year)).Scan(&n); err != nil {
			return err
		}
		noDaftar := fmt.Sprintf("PMB%d%05d", year, n+1)
		berkas, _ := json.Marshal(DefaultBerkas())
		if err := tx.QueryRow(ctx, `INSERT INTO pendaftar (tenant_id, user_id, no_daftar, nama, email, nik, gelombang_id, prodi_pilihan_id, berkas)
			VALUES ($1,$2,$3,$4,lower($5),$6,$7,$8,$9) RETURNING id`, tid, userID, noDaftar, in.Nama, in.Email, in.NIK, g.ID, in.ProdiPilihanID, berkas).Scan(&id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET identifier=$3 WHERE id=$1 AND tenant_id=$2`, userID, tid, noDaftar); err != nil {
			return err
		}
		va := t.Rules.VAPrefix + "9" + strings.TrimPrefix(noDaftar, "PMB")
		due := domain.LocalDate(s.Now().AddDate(0, 0, 7), t.Loc)
		var tagihanID string
		if err := tx.QueryRow(ctx, `INSERT INTO tagihan (tenant_id, pendaftar_id, jenis, jumlah, jatuh_tempo, va_number) VALUES ($1,$2,'formulir_pmb',$3,$4,$5) RETURNING id`,
			tid, id, g.BiayaFormulir, due, va).Scan(&tagihanID); err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, "pendaftar", id, "register", nil, map[string]any{"no_daftar": noDaftar, "gelombang": g.Nama, "prodi_pilihan_id": in.ProdiPilihanID}); err != nil {
			return err
		}
		return audit.Write(ctx, tx, "tagihan", tagihanID, "create", nil, map[string]any{"jenis": "formulir_pmb", "jumlah": g.BiayaFormulir, "va_number": va})
	})
	if err != nil {
		return Pendaftar{}, err
	}
	return s.getPendaftar(ctx, s.pool, id)
}

func (s *Service) Me(ctx context.Context) (Pendaftar, error) {
	var id string
	if err := s.pool.QueryRow(ctx, `SELECT id FROM pendaftar WHERE tenant_id=$1 AND user_id=$2 ORDER BY created_at DESC LIMIT 1`,
		appctx.TenantID(ctx), appctx.UserID(ctx)).Scan(&id); err != nil {
		return Pendaftar{}, httpx.NotFound("Data pendaftaran tidak ditemukan")
	}
	return s.getPendaftar(ctx, s.pool, id)
}

// Transition moves an applicant through the FR-09 state machine.
func (s *Service) Transition(ctx context.Context, id, to string) (Pendaftar, error) {
	if to == DaftarUlang {
		return s.DaftarUlang(ctx, id)
	}
	p, _ := appctx.PrincipalFrom(ctx)
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		var from string
		var owner *string
		var raw []byte
		if err := tx.QueryRow(ctx, `SELECT status, user_id, berkas FROM pendaftar WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tid, id).Scan(&from, &owner, &raw); err != nil {
			return httpx.NotFound("Pendaftar tidak ditemukan")
		}
		if p.ActiveRole == "calon_mahasiswa" {
			if owner == nil || *owner != p.UserID {
				return httpx.NotFound("Pendaftar tidak ditemukan")
			}
			if !ApplicantMay(from, to) {
				return httpx.Rule("PMB_INVALID_TRANSITION", "Transisi "+from+" → "+to+" tidak dapat dilakukan pendaftar")
			}
		}
		if !CanTransition(from, to) {
			return httpx.Rule("PMB_INVALID_TRANSITION", fmt.Sprintf("Transisi status %s → %s tidak diizinkan", from, to)).
				With(map[string]any{"from": from, "allowed": transitions[from]})
		}
		berkas := raw
		if to == Verifikasi { // documents submitted
			var b []Berkas
			_ = json.Unmarshal(raw, &b)
			for i := range b {
				b[i].OK = true
			}
			berkas, _ = json.Marshal(b)
		}
		if _, err := tx.Exec(ctx, `UPDATE pendaftar SET status=$3, berkas=$4, updated_at=now() WHERE tenant_id=$1 AND id=$2`, tid, id, to, berkas); err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, "pendaftar", id, "transition", map[string]string{"status": from}, map[string]string{"status": to}); err != nil {
			return err
		}
		if owner != nil && *owner != p.UserID {
			return notification.Send(ctx, tx, *owner, "Status pendaftaran", "Status pendaftaran Anda sekarang: "+to+".", notification.Link("/pmb"))
		}
		return nil
	})
	if err != nil {
		return Pendaftar{}, err
	}
	return s.getPendaftar(ctx, s.pool, id)
}

// DaftarUlang issues the NIM (EC-04): the (tenant, prodi, angkatan) sequence row is
// locked with SELECT ... FOR UPDATE, and (tenant_id, nim) is unique as a backstop.
func (s *Service) DaftarUlang(ctx context.Context, id string) (Pendaftar, error) {
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return Pendaftar{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		var status, nama, email, nik, prodiID string
		var userID *string
		if err := tx.QueryRow(ctx, `SELECT status, nama, email, nik, prodi_pilihan_id, user_id FROM pendaftar WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tid, id).
			Scan(&status, &nama, &email, &nik, &prodiID, &userID); err != nil {
			return httpx.NotFound("Pendaftar tidak ditemukan")
		}
		if !CanTransition(status, DaftarUlang) {
			return httpx.Rule("PMB_INVALID_TRANSITION", "Daftar ulang hanya untuk pendaftar berstatus lulus (saat ini: "+status+")")
		}
		ap, err := academic.ActivePeriode(ctx, tx)
		if err != nil {
			return err
		}
		angkatan := ap.Tahun
		var prodiKode string
		if err := tx.QueryRow(ctx, `SELECT kode FROM prodi WHERE tenant_id=$1 AND id=$2`, tid, prodiID).Scan(&prodiKode); err != nil {
			return err
		}
		// EC-04: per-(prodi, angkatan) sequence with a row lock.
		if _, err := tx.Exec(ctx, `INSERT INTO nim_sequence (tenant_id, prodi_id, angkatan) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, tid, prodiID, angkatan); err != nil {
			return err
		}
		var last int
		if err := tx.QueryRow(ctx, `SELECT last_no FROM nim_sequence WHERE tenant_id=$1 AND prodi_id=$2 AND angkatan=$3 FOR UPDATE`, tid, prodiID, angkatan).Scan(&last); err != nil {
			return err
		}
		var nim string
		for {
			last++
			nim = FormatNIM(t.Rules.NIMFormat, angkatan, prodiKode, last)
			var taken bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM mahasiswa WHERE tenant_id=$1 AND nim=$2)`, tid, nim).Scan(&taken); err != nil {
				return err
			}
			if !taken {
				break
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE nim_sequence SET last_no=$4, updated_at=now() WHERE tenant_id=$1 AND prodi_id=$2 AND angkatan=$3`, tid, prodiID, angkatan, last); err != nil {
			return err
		}
		var kurID string
		if err := tx.QueryRow(ctx, `SELECT id FROM kurikulum WHERE tenant_id=$1 AND prodi_id=$2 AND aktif ORDER BY (angkatan_mulai <= $3) DESC, angkatan_mulai DESC LIMIT 1`,
			tid, prodiID, angkatan).Scan(&kurID); err != nil {
			return httpx.Rule("VALIDATION_ERROR", "Prodi belum memiliki kurikulum aktif")
		}
		var wali *string
		_ = tx.QueryRow(ctx, `SELECT d.id FROM dosen d WHERE d.tenant_id=$1 AND d.prodi_id=$2 ORDER BY (SELECT count(*) FROM mahasiswa m WHERE m.dosen_wali_id=d.id), d.nidn LIMIT 1`,
			tid, prodiID).Scan(&wali)
		if userID == nil {
			hash, _ := bcrypt.GenerateFromPassword([]byte(nim), bcrypt.DefaultCost)
			var uid string
			if err := tx.QueryRow(ctx, `INSERT INTO users (tenant_id, nama, email, identifier, password_hash) VALUES ($1,$2,lower($3),$4,$5) RETURNING id`,
				tid, nama, email, nim, string(hash)).Scan(&uid); err != nil {
				return err
			}
			userID = &uid
		} else if _, err := tx.Exec(ctx, `UPDATE users SET identifier=$3, updated_at=now() WHERE id=$1 AND tenant_id=$2`, *userID, tid, nim); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO user_role (tenant_id, user_id, role) VALUES ($1,$2,'mahasiswa') ON CONFLICT DO NOTHING`, tid, *userID); err != nil {
			return err
		}
		var mhsID string
		if err := tx.QueryRow(ctx, `INSERT INTO mahasiswa (tenant_id, user_id, nim, nik, nama, email, prodi_id, angkatan, kurikulum_id, dosen_wali_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`, tid, *userID, nim, nik, nama, email, prodiID, angkatan, kurID, wali).Scan(&mhsID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO riwayat_status (tenant_id, mahasiswa_id, periode_id, status, alasan, created_by) VALUES ($1,$2,$3,'aktif','Daftar ulang PMB',$4)`,
			tid, mhsID, ap.ID, appctx.UserID(ctx)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE pendaftar SET status='daftar_ulang', nim=$3, mahasiswa_id=$4, updated_at=now() WHERE tenant_id=$1 AND id=$2`, tid, id, nim, mhsID); err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, "pendaftar", id, "daftar_ulang", map[string]string{"status": status}, map[string]string{"status": DaftarUlang, "nim": nim}); err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, "mahasiswa", mhsID, "create", nil, map[string]any{"nim": nim, "nama": nama, "prodi_id": prodiID, "angkatan": angkatan}); err != nil {
			return err
		}
		return notification.Send(ctx, tx, *userID, "NIM terbit", "Selamat! NIM Anda: "+nim+". Gunakan NIM untuk login.", notification.Link("/"))
	})
	if err != nil {
		return Pendaftar{}, err
	}
	return s.getPendaftar(ctx, s.pool, id)
}
