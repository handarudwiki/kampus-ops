package lecture

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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

type Peserta struct {
	MahasiswaID string `json:"mahasiswa_id"`
	NIM         string `json:"nim"`
	Nama        string `json:"nama"`
}

type Rekap struct {
	Hadir int `json:"hadir"`
	Izin  int `json:"izin"`
	Sakit int `json:"sakit"`
	Alpa  int `json:"alpa"`
}

type Pertemuan struct {
	ID        string `json:"id"`
	KelasID   string `json:"kelas_id"`
	Ke        int    `json:"ke"`
	Tanggal   string `json:"tanggal"`
	Topik     string `json:"topik"`
	DosenNama string `json:"dosen_nama"`
	Rekap     Rekap  `json:"rekap"`
}

type PresensiRow struct {
	MahasiswaID string  `json:"mahasiswa_id"`
	NIM         string  `json:"nim"`
	Nama        string  `json:"nama"`
	Status      *string `json:"status"`
}

type KomponenNilai struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	BobotPersen int    `json:"bobot_persen"`
}

type NilaiRow struct {
	KrsDetailID    string              `json:"krs_detail_id"`
	NilaiID        *string             `json:"nilai_id"`
	MahasiswaID    string              `json:"mahasiswa_id"`
	NIM            string              `json:"nim"`
	Nama           string              `json:"nama"`
	Komponen       map[string]*float64 `json:"komponen"`
	Angka          *string             `json:"angka"`
	Huruf          *string             `json:"huruf"`
	BobotX100      *int                `json:"bobot_x100"`
	Final          bool                `json:"final"`
	KoreksiPending bool                `json:"koreksi_pending"`
}

type NilaiSheet struct {
	Komponen  []KomponenNilai `json:"komponen"`
	Rows      []NilaiRow      `json:"rows"`
	Finalized bool            `json:"finalized"`
	Deadline  string          `json:"deadline"`
}

type KoreksiNilai struct {
	ID            string `json:"id"`
	NilaiID       string `json:"nilai_id"`
	KelasNama     string `json:"kelas_nama"`
	MkNama        string `json:"mk_nama"`
	MahasiswaNama string `json:"mahasiswa_nama"`
	NIM           string `json:"nim"`
	HurufLama     string `json:"huruf_lama"`
	HurufBaru     string `json:"huruf_baru"`
	AngkaLama     string `json:"angka_lama"`
	AngkaBaru     string `json:"angka_baru"`
	Alasan        string `json:"alasan"`
	Status        string `json:"status"`
	DiajukanOleh  string `json:"diajukan_oleh"`
	CreatedAt     string `json:"created_at"`
}

// ---- access ----

type kelasAccess struct {
	K       academic.Kelas
	Hist    []academic.DosenAssignment
	DosenID string // caller's dosen record ("" if none)
	Role    string
	ProdiID string
}

// access resolves whether the caller may see a class: the lecturer (any assignment),
// operator / admin_kampus, or kaprodi of the course's prodi. Others get 404 (EC-16 style).
func (s *Service) access(ctx context.Context, q db.Querier, kelasID string) (kelasAccess, error) {
	var a kelasAccess
	k, err := academic.GetKelas(ctx, q, kelasID)
	if err != nil {
		return a, httpx.NotFound("Kelas tidak ditemukan")
	}
	a.K = k
	if a.Hist, err = academic.KelasDosenHistory(ctx, q, kelasID); err != nil {
		return a, err
	}
	if err := q.QueryRow(ctx, `SELECT ku.prodi_id FROM mata_kuliah mk JOIN kurikulum ku ON ku.id=mk.kurikulum_id WHERE mk.id=$1 AND mk.tenant_id=$2`,
		k.MkID, appctx.TenantID(ctx)).Scan(&a.ProdiID); err != nil {
		return a, err
	}
	p, _ := appctx.PrincipalFrom(ctx)
	a.Role = p.ActiveRole
	if a.DosenID, err = academic.DosenIDByUser(ctx, q, p.UserID); err != nil {
		return a, err
	}
	switch p.ActiveRole {
	case "operator", "admin_kampus":
		return a, nil
	case "kaprodi":
		if sc := p.ScopeProdi("kaprodi"); sc == nil || *sc == a.ProdiID {
			return a, nil
		}
	case "dosen", "dosen_wali":
		if a.DosenID != "" && (Teaches(a.Hist, a.DosenID) || (k.DosenID != nil && *k.DosenID == a.DosenID)) {
			return a, nil
		}
	}
	return a, httpx.NotFound("Kelas tidak ditemukan")
}

func (s *Service) today(ctx context.Context) string {
	return domain.LocalDate(s.Now(), appctx.Tenant(ctx).Loc)
}

// mayGrade: the lecturer holding the class today (EC-12), or an operator.
func (s *Service) mayGrade(ctx context.Context, a kelasAccess) bool {
	if a.Role == "operator" {
		return true
	}
	hist := a.Hist
	if len(hist) == 0 && a.K.DosenID != nil {
		hist = []academic.DosenAssignment{{DosenID: *a.K.DosenID, EfektifMulai: "2000-01-01"}}
	}
	return CanGrade(hist, a.DosenID, s.today(ctx))
}

// ---- teaching ----

func (s *Service) TeachingKelas(ctx context.Context, periodeID string) ([]academic.Kelas, error) {
	if periodeID == "" {
		ap, err := academic.ActivePeriode(ctx, s.pool)
		if err != nil {
			return nil, err
		}
		periodeID = ap.ID
	}
	did, err := academic.DosenIDByUser(ctx, s.pool, appctx.UserID(ctx))
	if err != nil || did == "" {
		return []academic.Kelas{}, err
	}
	return academic.QueryKelas(ctx, s.pool, `k.periode_id::text=$2 AND (k.dosen_id=$3 OR EXISTS (SELECT 1 FROM kelas_dosen kd WHERE kd.kelas_id=k.id
		AND kd.dosen_id=$3 AND kd.efektif_mulai <= $4::date AND (kd.efektif_selesai IS NULL OR kd.efektif_selesai >= $4::date)))
		ORDER BY k.hari, k.jam_mulai`, periodeID, did, s.today(ctx))
}

func (s *Service) Peserta(ctx context.Context, kelasID string) ([]Peserta, error) {
	if _, err := s.access(ctx, s.pool, kelasID); err != nil {
		return nil, err
	}
	return peserta(ctx, s.pool, kelasID)
}

func peserta(ctx context.Context, q db.Querier, kelasID string) ([]Peserta, error) {
	rows, err := q.Query(ctx, `SELECT m.id, m.nim, m.nama FROM krs_detail d JOIN krs ON krs.id=d.krs_id JOIN mahasiswa m ON m.id=krs.mahasiswa_id
		WHERE d.tenant_id=$1 AND d.kelas_id=$2 AND d.status='aktif' ORDER BY m.nim`, appctx.TenantID(ctx), kelasID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Peserta])
	if out == nil {
		out = []Peserta{}
	}
	return out, err
}

// ---- pertemuan & presensi (FR-21) ----

const pertemuanSel = `SELECT pt.id, pt.kelas_id, pt.ke, pt.tanggal::text, pt.topik, coalesce(d.nama,'-'),
	count(*) FILTER (WHERE pr.status='H'), count(*) FILTER (WHERE pr.status='I'), count(*) FILTER (WHERE pr.status='S'), count(*) FILTER (WHERE pr.status='A')
	FROM pertemuan pt LEFT JOIN dosen d ON d.id=pt.dosen_id LEFT JOIN presensi pr ON pr.pertemuan_id=pt.id `

func scanPertemuan(row pgx.Row) (Pertemuan, error) {
	var p Pertemuan
	err := row.Scan(&p.ID, &p.KelasID, &p.Ke, &p.Tanggal, &p.Topik, &p.DosenNama, &p.Rekap.Hadir, &p.Rekap.Izin, &p.Rekap.Sakit, &p.Rekap.Alpa)
	return p, err
}

func (s *Service) ListPertemuan(ctx context.Context, kelasID string) ([]Pertemuan, error) {
	if _, err := s.access(ctx, s.pool, kelasID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, pertemuanSel+`WHERE pt.tenant_id=$1 AND pt.kelas_id=$2 GROUP BY pt.id, d.nama ORDER BY pt.ke`, appctx.TenantID(ctx), kelasID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Pertemuan{}
	for rows.Next() {
		p, err := scanPertemuan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) getPertemuan(ctx context.Context, q db.Querier, id string) (Pertemuan, error) {
	return scanPertemuan(q.QueryRow(ctx, pertemuanSel+`WHERE pt.tenant_id=$1 AND pt.id=$2 GROUP BY pt.id, d.nama`, appctx.TenantID(ctx), id))
}

// CreatePertemuan records the lecturer holding the class on that date (EC-12).
func (s *Service) CreatePertemuan(ctx context.Context, kelasID, tanggal, topik string) (Pertemuan, error) {
	if err := httpx.NewV().Date("tanggal", tanggal).Err(); err != nil {
		return Pertemuan{}, err
	}
	a, err := s.access(ctx, s.pool, kelasID)
	if err != nil {
		return Pertemuan{}, err
	}
	dosenOn, ok := academic.DosenOn(a.Hist, tanggal)
	if !ok && a.K.DosenID != nil {
		dosenOn, ok = *a.K.DosenID, true
	}
	if (a.Role == "dosen" || a.Role == "dosen_wali") && (!ok || dosenOn != a.DosenID) {
		return Pertemuan{}, httpx.Forbidden("GRADE_NOT_ALLOWED", "Anda bukan dosen pengampu kelas ini pada tanggal tersebut")
	}
	var dosen *string
	if ok {
		dosen = &dosenOn
	}
	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT 1 FROM kelas WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, kelasID, appctx.TenantID(ctx)); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `INSERT INTO pertemuan (tenant_id, kelas_id, ke, tanggal, topik, dosen_id)
			VALUES ($1,$2,(SELECT coalesce(max(ke),0)+1 FROM pertemuan WHERE kelas_id=$2),$3,$4,$5) RETURNING id`,
			appctx.TenantID(ctx), kelasID, tanggal, topik, dosen).Scan(&id)
		if err != nil {
			return err
		}
		return audit.Write(ctx, tx, "pertemuan", id, "create", nil, map[string]any{"kelas_id": kelasID, "tanggal": tanggal, "topik": topik, "dosen_id": dosen})
	})
	if err != nil {
		return Pertemuan{}, err
	}
	return s.getPertemuan(ctx, s.pool, id)
}

func (s *Service) pertemuanKelas(ctx context.Context, id string) (string, error) {
	var k string
	err := s.pool.QueryRow(ctx, `SELECT kelas_id FROM pertemuan WHERE tenant_id=$1 AND id=$2`, appctx.TenantID(ctx), id).Scan(&k)
	if err != nil {
		return "", httpx.NotFound("Pertemuan tidak ditemukan")
	}
	return k, nil
}

func (s *Service) Presensi(ctx context.Context, pertemuanID string) ([]PresensiRow, error) {
	kelasID, err := s.pertemuanKelas(ctx, pertemuanID)
	if err != nil {
		return nil, err
	}
	if _, err := s.access(ctx, s.pool, kelasID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT m.id, m.nim, m.nama, pr.status FROM krs_detail d JOIN krs ON krs.id=d.krs_id JOIN mahasiswa m ON m.id=krs.mahasiswa_id
		LEFT JOIN presensi pr ON pr.pertemuan_id=$3 AND pr.mahasiswa_id=m.id
		WHERE d.tenant_id=$1 AND d.kelas_id=$2 AND d.status='aktif' ORDER BY m.nim`, appctx.TenantID(ctx), kelasID, pertemuanID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[PresensiRow])
	if out == nil {
		out = []PresensiRow{}
	}
	return out, err
}

type PresensiItem struct {
	MahasiswaID string `json:"mahasiswa_id"`
	Status      string `json:"status"`
}

func (s *Service) SavePresensi(ctx context.Context, pertemuanID string, items []PresensiItem) ([]PresensiRow, error) {
	kelasID, err := s.pertemuanKelas(ctx, pertemuanID)
	if err != nil {
		return nil, err
	}
	a, err := s.access(ctx, s.pool, kelasID)
	if err != nil {
		return nil, err
	}
	if a.Role == "kaprodi" || a.Role == "admin_kampus" {
		return nil, httpx.Forbidden("FORBIDDEN_ROLE", "Hanya dosen pengampu atau operator yang dapat mengisi presensi")
	}
	ps, err := peserta(ctx, s.pool, kelasID)
	if err != nil {
		return nil, err
	}
	enrolled := map[string]bool{}
	for _, p := range ps {
		enrolled[p.MahasiswaID] = true
	}
	v := httpx.NewV()
	for _, it := range items {
		v.OneOf("items.status", it.Status, "H", "I", "S", "A")
		v.Check(enrolled[it.MahasiswaID], "items.mahasiswa_id", "mahasiswa bukan peserta kelas: "+it.MahasiswaID)
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		for _, it := range items {
			if _, err := tx.Exec(ctx, `INSERT INTO presensi (tenant_id, pertemuan_id, mahasiswa_id, status) VALUES ($1,$2,$3,$4)
				ON CONFLICT (tenant_id, pertemuan_id, mahasiswa_id) DO UPDATE SET status=EXCLUDED.status, updated_at=now()`,
				appctx.TenantID(ctx), pertemuanID, it.MahasiswaID, it.Status); err != nil {
				return err
			}
		}
		return audit.Write(ctx, tx, "presensi", pertemuanID, "save", nil, items)
	})
	if err != nil {
		return nil, err
	}
	return s.Presensi(ctx, pertemuanID)
}

// ---- nilai (FR-22, FR-23) ----

func (s *Service) komponen(ctx context.Context, q db.Querier, kelasID string) ([]domain.Komponen, error) {
	rows, err := q.Query(ctx, `SELECT key, label, bobot_persen FROM komponen_nilai WHERE tenant_id=$1 AND kelas_id=$2 ORDER BY urutan, key`, appctx.TenantID(ctx), kelasID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[domain.Komponen])
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		t, err := s.tenants.Current(ctx)
		if err != nil {
			return nil, err
		}
		out = t.Rules.KomponenDefault
	}
	return out, nil
}

func fromX100(m map[string]*int) map[string]*float64 {
	out := map[string]*float64{}
	for k, v := range m {
		if v == nil {
			out[k] = nil
			continue
		}
		f := float64(*v) / 100
		out[k] = &f
	}
	return out
}

func (s *Service) sheet(ctx context.Context, q db.Querier, a kelasAccess) (NilaiSheet, error) {
	komp, err := s.komponen(ctx, q, a.K.ID)
	if err != nil {
		return NilaiSheet{}, err
	}
	p, err := academic.GetPeriode(ctx, q, a.K.PeriodeID)
	if err != nil {
		return NilaiSheet{}, err
	}
	out := NilaiSheet{Komponen: []KomponenNilai{}, Rows: []NilaiRow{}, Deadline: p.Tahapan.NilaiSelesai}
	for _, k := range komp {
		out.Komponen = append(out.Komponen, KomponenNilai{Key: k.Key, Label: k.Label, BobotPersen: k.BobotPersen})
	}
	rows, err := q.Query(ctx, `SELECT d.id, n.id, m.id, m.nim, m.nama, coalesce(n.komponen,'{}'::jsonb), n.angka_x100, n.huruf, n.bobot_x100,
		n.final_at IS NOT NULL, EXISTS (SELECT 1 FROM koreksi_nilai kn WHERE kn.nilai_id=n.id AND kn.status='diajukan')
		FROM krs_detail d JOIN krs ON krs.id=d.krs_id JOIN mahasiswa m ON m.id=krs.mahasiswa_id LEFT JOIN nilai n ON n.krs_detail_id=d.id
		WHERE d.tenant_id=$1 AND d.kelas_id=$2 AND d.status='aktif' ORDER BY m.nim`, appctx.TenantID(ctx), a.K.ID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	anyFinal := false
	for rows.Next() {
		var r NilaiRow
		var raw []byte
		var angka *int
		if err := rows.Scan(&r.KrsDetailID, &r.NilaiID, &r.MahasiswaID, &r.NIM, &r.Nama, &raw, &angka, &r.Huruf, &r.BobotX100, &r.Final, &r.KoreksiPending); err != nil {
			return out, err
		}
		var sc map[string]*int
		_ = json.Unmarshal(raw, &sc)
		r.Komponen = fromX100(sc)
		for _, k := range komp {
			if _, ok := r.Komponen[k.Key]; !ok {
				r.Komponen[k.Key] = nil
			}
		}
		if angka != nil {
			s := domain.FormatX100(*angka)
			r.Angka = &s
		}
		anyFinal = anyFinal || r.Final
		out.Rows = append(out.Rows, r)
	}
	out.Finalized = anyFinal
	return out, rows.Err()
}

func (s *Service) GetNilai(ctx context.Context, kelasID string) (NilaiSheet, error) {
	a, err := s.access(ctx, s.pool, kelasID)
	if err != nil {
		return NilaiSheet{}, err
	}
	return s.sheet(ctx, s.pool, a)
}

type NilaiInput struct {
	KrsDetailID string              `json:"krs_detail_id"`
	Komponen    map[string]*float64 `json:"komponen"`
}

func (s *Service) checkEditable(ctx context.Context, q db.Querier, a kelasAccess) error {
	if !s.mayGrade(ctx, a) {
		return httpx.Forbidden("GRADE_NOT_ALLOWED", "Hak input nilai kelas ini dimiliki dosen pengampu saat ini")
	}
	var finalized bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM nilai WHERE tenant_id=$1 AND kelas_id=$2 AND final_at IS NOT NULL)`, appctx.TenantID(ctx), a.K.ID).Scan(&finalized); err != nil {
		return err
	}
	p, err := academic.GetPeriode(ctx, q, a.K.PeriodeID)
	if err != nil {
		return err
	}
	if !GradeEditable(finalized, p.Tahapan.NilaiSelesai, s.Now(), appctx.Tenant(ctx).Loc) {
		if finalized {
			return httpx.Rule("GRADE_LOCKED", "Nilai sudah difinalisasi; gunakan pengajuan koreksi nilai")
		}
		return httpx.Rule("GRADE_LOCKED", "Batas input nilai ("+p.Tahapan.NilaiSelesai+") sudah lewat")
	}
	return nil
}

func (s *Service) SaveNilai(ctx context.Context, kelasID string, in []NilaiInput) (NilaiSheet, error) {
	a, err := s.access(ctx, s.pool, kelasID)
	if err != nil {
		return NilaiSheet{}, err
	}
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return NilaiSheet{}, err
	}
	komp, err := s.komponen(ctx, s.pool, kelasID)
	if err != nil {
		return NilaiSheet{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		if _, err := tx.Exec(ctx, `SELECT 1 FROM kelas WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, kelasID, tid); err != nil {
			return err
		}
		if err := s.checkEditable(ctx, tx, a); err != nil {
			return err
		}
		for _, row := range in {
			var mhs string
			if err := tx.QueryRow(ctx, `SELECT krs.mahasiswa_id FROM krs_detail d JOIN krs ON krs.id=d.krs_id WHERE d.tenant_id=$1 AND d.id::text=$2 AND d.kelas_id=$3 AND d.status='aktif'`,
				tid, row.KrsDetailID, kelasID).Scan(&mhs); err != nil {
				return httpx.ValidationField("rows.krs_detail_id", "bukan peserta kelas ini: "+row.KrsDetailID)
			}
			// merge with existing scores
			existing := map[string]*int{}
			var raw []byte
			if err := tx.QueryRow(ctx, `SELECT komponen FROM nilai WHERE tenant_id=$1 AND krs_detail_id=$2`, tid, row.KrsDetailID).Scan(&raw); err == nil {
				_ = json.Unmarshal(raw, &existing)
			}
			scores, err := ScoresToX100(komp, row.Komponen)
			if err != nil {
				return httpx.ValidationField("rows.komponen", err.Error())
			}
			for k, v := range scores {
				existing[k] = v
			}
			g := ComputeGrade(t.Rules, komp, existing)
			b, _ := json.Marshal(existing)
			if _, err := tx.Exec(ctx, `INSERT INTO nilai (tenant_id, krs_detail_id, mahasiswa_id, kelas_id, komponen, angka_x100, huruf, bobot_x100)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (tenant_id, krs_detail_id) WHERE krs_detail_id IS NOT NULL
				DO UPDATE SET komponen=EXCLUDED.komponen, angka_x100=EXCLUDED.angka_x100, huruf=EXCLUDED.huruf, bobot_x100=EXCLUDED.bobot_x100, updated_at=now()`,
				tid, row.KrsDetailID, mhs, kelasID, b, g.AngkaX100, g.Huruf, g.BobotX100); err != nil {
				return err
			}
		}
		return audit.Write(ctx, tx, "nilai_kelas", kelasID, "save", nil, in)
	})
	if err != nil {
		return NilaiSheet{}, err
	}
	return s.sheet(ctx, s.pool, a)
}

// Finalize locks all grades of a class (FR-23) and recomputes IPK caches in the same tx.
func (s *Service) Finalize(ctx context.Context, kelasID string) (NilaiSheet, error) {
	a, err := s.access(ctx, s.pool, kelasID)
	if err != nil {
		return NilaiSheet{}, err
	}
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return NilaiSheet{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		if _, err := tx.Exec(ctx, `SELECT 1 FROM kelas WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, kelasID, tid); err != nil {
			return err
		}
		if !s.mayGrade(ctx, a) {
			return httpx.Forbidden("GRADE_NOT_ALLOWED", "Hak finalisasi nilai dimiliki dosen pengampu saat ini")
		}
		sh, err := s.sheet(ctx, tx, a)
		if err != nil {
			return err
		}
		if sh.Finalized {
			return httpx.Rule("GRADE_LOCKED", "Nilai kelas ini sudah difinalisasi")
		}
		if len(sh.Rows) == 0 {
			return httpx.ValidationField("rows", "kelas tidak memiliki peserta")
		}
		missing := []string{}
		for _, r := range sh.Rows {
			if r.Huruf == nil {
				missing = append(missing, r.NIM)
			}
		}
		if len(missing) > 0 {
			return httpx.ValidationField("rows", "nilai belum lengkap untuk NIM: "+strings.Join(missing, ", "))
		}
		if _, err := tx.Exec(ctx, `UPDATE nilai SET final_at=now(), updated_at=now() WHERE tenant_id=$1 AND kelas_id=$2 AND final_at IS NULL`, tid, kelasID); err != nil {
			return err
		}
		for _, r := range sh.Rows {
			if err := academic.RecomputeStudent(ctx, tx, t.Rules, r.MahasiswaID); err != nil {
				return err
			}
			var uid *string
			_ = tx.QueryRow(ctx, `SELECT user_id FROM mahasiswa WHERE id=$1 AND tenant_id=$2`, r.MahasiswaID, tid).Scan(&uid)
			if uid != nil {
				if err := notification.Send(ctx, tx, *uid, "Nilai terbit", fmt.Sprintf("Nilai %s sudah terbit.", a.K.MkNama), notification.Link("/akademik/khs")); err != nil {
					return err
				}
			}
		}
		return audit.Write(ctx, tx, "nilai_kelas", kelasID, "finalize", nil, map[string]any{"jumlah": len(sh.Rows)})
	})
	if err != nil {
		return NilaiSheet{}, err
	}
	return s.sheet(ctx, s.pool, a)
}

// ---- koreksi nilai (EC-11) ----

const koreksiSel = `SELECT kn.id, kn.nilai_id, k.nama, mk.nama, m.nama, m.nim, kn.huruf_lama, kn.huruf_baru, kn.angka_lama, kn.angka_baru,
	kn.alasan, kn.status, u.nama, iso(kn.created_at)
	FROM koreksi_nilai kn JOIN nilai n ON n.id=kn.nilai_id JOIN kelas k ON k.id=n.kelas_id JOIN mata_kuliah mk ON mk.id=k.mk_id
	JOIN kurikulum ku ON ku.id=mk.kurikulum_id JOIN mahasiswa m ON m.id=n.mahasiswa_id JOIN users u ON u.id=kn.diajukan_oleh `

func scanKoreksi(row pgx.Row) (KoreksiNilai, error) {
	var k KoreksiNilai
	var al, ab int
	err := row.Scan(&k.ID, &k.NilaiID, &k.KelasNama, &k.MkNama, &k.MahasiswaNama, &k.NIM, &k.HurufLama, &k.HurufBaru, &al, &ab, &k.Alasan, &k.Status, &k.DiajukanOleh, &k.CreatedAt)
	k.AngkaLama, k.AngkaBaru = domain.FormatX100(al), domain.FormatX100(ab)
	return k, err
}

func (s *Service) getKoreksi(ctx context.Context, q db.Querier, id string) (KoreksiNilai, error) {
	return scanKoreksi(q.QueryRow(ctx, koreksiSel+`WHERE kn.tenant_id=$1 AND kn.id=$2`, appctx.TenantID(ctx), id))
}

func (s *Service) RequestKoreksi(ctx context.Context, nilaiID string, komponen map[string]*float64, alasan string) (KoreksiNilai, error) {
	if err := httpx.NewV().Required("alasan", alasan).Check(len(komponen) > 0, "komponen", "wajib diisi").Err(); err != nil {
		return KoreksiNilai{}, err
	}
	tid := appctx.TenantID(ctx)
	var kelasID string
	var final bool
	var raw []byte
	var angka, bobot *int
	var huruf *string
	err := s.pool.QueryRow(ctx, `SELECT kelas_id, final_at IS NOT NULL, komponen, angka_x100, huruf, bobot_x100 FROM nilai WHERE tenant_id=$1 AND id=$2`, tid, nilaiID).
		Scan(&kelasID, &final, &raw, &angka, &huruf, &bobot)
	if err != nil {
		return KoreksiNilai{}, httpx.NotFound("Nilai tidak ditemukan")
	}
	a, err := s.access(ctx, s.pool, kelasID)
	if err != nil {
		return KoreksiNilai{}, err
	}
	if !final || angka == nil {
		return KoreksiNilai{}, httpx.Rule("GRADE_NOT_ALLOWED", "Nilai belum final; ubah langsung melalui input nilai")
	}
	if !s.mayGrade(ctx, a) {
		return KoreksiNilai{}, httpx.Forbidden("GRADE_NOT_ALLOWED", "Hanya dosen pengampu saat ini yang dapat mengajukan koreksi")
	}
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return KoreksiNilai{}, err
	}
	komp, err := s.komponen(ctx, s.pool, kelasID)
	if err != nil {
		return KoreksiNilai{}, err
	}
	old := map[string]*int{}
	_ = json.Unmarshal(raw, &old)
	scores, err := ScoresToX100(komp, komponen)
	if err != nil {
		return KoreksiNilai{}, httpx.ValidationField("komponen", err.Error())
	}
	merged := map[string]*int{}
	for k, v := range old {
		merged[k] = v
	}
	for k, v := range scores {
		merged[k] = v
	}
	g := ComputeGrade(t.Rules, komp, merged)
	if !g.Complete {
		return KoreksiNilai{}, httpx.ValidationField("komponen", "semua komponen harus terisi")
	}
	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		ob, _ := json.Marshal(old)
		nb, _ := json.Marshal(merged)
		err := tx.QueryRow(ctx, `INSERT INTO koreksi_nilai (tenant_id, nilai_id, komponen_lama, komponen_baru, angka_lama, angka_baru, huruf_lama, huruf_baru,
			bobot_lama, bobot_baru, alasan, diajukan_oleh) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`,
			tid, nilaiID, ob, nb, *angka, *g.AngkaX100, *huruf, *g.Huruf, *bobot, *g.BobotX100, alasan, appctx.UserID(ctx)).Scan(&id)
		if db.IsUniqueViolation(err) {
			return httpx.Conflict("DUPLICATE", "Masih ada pengajuan koreksi yang menunggu untuk nilai ini")
		}
		if err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, "koreksi_nilai", id, "create", map[string]any{"huruf": *huruf, "angka_x100": *angka},
			map[string]any{"huruf": *g.Huruf, "angka_x100": *g.AngkaX100, "alasan": alasan}); err != nil {
			return err
		}
		// notify the kaprodi(s) of the prodi
		rows, err := tx.Query(ctx, `SELECT user_id FROM user_role WHERE tenant_id=$1 AND role='kaprodi' AND (scope_prodi_id IS NULL OR scope_prodi_id=$2)`, tid, a.ProdiID)
		if err != nil {
			return err
		}
		uids, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		for _, u := range uids {
			if err := notification.Send(ctx, tx, u, "Pengajuan koreksi nilai", fmt.Sprintf("Koreksi nilai %s: %s → %s.", a.K.MkNama, *huruf, *g.Huruf), notification.Link("/koreksi-nilai")); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return KoreksiNilai{}, err
	}
	return s.getKoreksi(ctx, s.pool, id)
}

func (s *Service) ListKoreksi(ctx context.Context, status string) ([]KoreksiNilai, error) {
	p, _ := appctx.PrincipalFrom(ctx)
	prodi, user := "", ""
	switch p.ActiveRole {
	case "kaprodi":
		if sc := p.ScopeProdi("kaprodi"); sc != nil {
			prodi = *sc
		}
	case "operator", "admin_kampus":
	default:
		user = p.UserID
	}
	rows, err := s.pool.Query(ctx, koreksiSel+`WHERE kn.tenant_id=$1 AND ($2='' OR kn.status=$2) AND ($3='' OR ku.prodi_id::text=$3) AND ($4='' OR kn.diajukan_oleh::text=$4)
		ORDER BY kn.created_at DESC LIMIT 200`, appctx.TenantID(ctx), status, prodi, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []KoreksiNilai{}
	for rows.Next() {
		k, err := scanKoreksi(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// DecideKoreksi: kaprodi approves (apply + recompute IPS/IPK + "perlu lapor ulang" if
// the period was already exported) or rejects (EC-11).
func (s *Service) DecideKoreksi(ctx context.Context, id string, approve bool, catatan string) (KoreksiNilai, error) {
	if !approve && strings.TrimSpace(catatan) == "" {
		return KoreksiNilai{}, httpx.ValidationField("catatan", "wajib diisi saat menolak")
	}
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return KoreksiNilai{}, err
	}
	p, _ := appctx.PrincipalFrom(ctx)
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		var nilaiID, status, prodi, mhs, periodeID, huruf string
		var komp []byte
		var angka, bobot int
		var exported bool
		var diajukan string
		err := tx.QueryRow(ctx, `SELECT kn.nilai_id, kn.status, ku.prodi_id, n.mahasiswa_id, k.periode_id, kn.komponen_baru, kn.angka_baru, kn.huruf_baru, kn.bobot_baru,
			p.exported_at IS NOT NULL, kn.diajukan_oleh
			FROM koreksi_nilai kn JOIN nilai n ON n.id=kn.nilai_id JOIN kelas k ON k.id=n.kelas_id JOIN mata_kuliah mk ON mk.id=k.mk_id
			JOIN kurikulum ku ON ku.id=mk.kurikulum_id JOIN periode p ON p.id=k.periode_id
			WHERE kn.tenant_id=$1 AND kn.id=$2 FOR UPDATE OF kn`, tid, id).
			Scan(&nilaiID, &status, &prodi, &mhs, &periodeID, &komp, &angka, &huruf, &bobot, &exported, &diajukan)
		if err != nil {
			return httpx.NotFound("Koreksi tidak ditemukan")
		}
		if sc := p.ScopeProdi("kaprodi"); p.ActiveRole == "kaprodi" && sc != nil && *sc != prodi {
			return httpx.NotFound("Koreksi tidak ditemukan")
		}
		if status != "diajukan" {
			return httpx.Rule("VALIDATION_ERROR", "Koreksi sudah diproses")
		}
		newStatus := "ditolak"
		if approve {
			newStatus = "disetujui"
		}
		var note *string
		if catatan != "" {
			note = &catatan
		}
		if _, err := tx.Exec(ctx, `UPDATE koreksi_nilai SET status=$3, approved_by=$4, catatan=$5, decided_at=now(), updated_at=now() WHERE tenant_id=$1 AND id=$2`,
			tid, id, newStatus, p.UserID, note); err != nil {
			return err
		}
		if approve {
			var before []byte
			if err := tx.QueryRow(ctx, `SELECT to_jsonb(n) FROM nilai n WHERE tenant_id=$1 AND id=$2`, tid, nilaiID).Scan(&before); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE nilai SET komponen=$3, angka_x100=$4, huruf=$5, bobot_x100=$6, perlu_lapor_ulang = perlu_lapor_ulang OR $7, updated_at=now()
				WHERE tenant_id=$1 AND id=$2`, tid, nilaiID, komp, angka, huruf, bobot, NeedsReReport(exported)); err != nil {
				return err
			}
			if err := academic.RecomputeStudent(ctx, tx, t.Rules, mhs); err != nil {
				return err
			}
			if err := audit.Write(ctx, tx, "nilai", nilaiID, "koreksi_applied", before, map[string]any{"huruf": huruf, "angka_x100": angka, "bobot_x100": bobot, "perlu_lapor_ulang": exported, "koreksi_id": id}); err != nil {
				return err
			}
			var uid *string
			_ = tx.QueryRow(ctx, `SELECT user_id FROM mahasiswa WHERE id=$1 AND tenant_id=$2`, mhs, tid).Scan(&uid)
			if uid != nil {
				if err := notification.Send(ctx, tx, *uid, "Nilai dikoreksi", "Salah satu nilai Anda dikoreksi menjadi "+huruf+".", notification.Link("/akademik/transkrip")); err != nil {
					return err
				}
			}
		}
		if err := notification.Send(ctx, tx, diajukan, "Koreksi nilai "+newStatus, "Pengajuan koreksi nilai Anda "+newStatus+".", notification.Link("/koreksi-nilai")); err != nil {
			return err
		}
		return audit.Write(ctx, tx, "koreksi_nilai", id, map[bool]string{true: "approve", false: "reject"}[approve], map[string]string{"status": status}, map[string]any{"status": newStatus, "catatan": catatan})
	})
	if err != nil {
		return KoreksiNilai{}, err
	}
	return s.getKoreksi(ctx, s.pool, id)
}
