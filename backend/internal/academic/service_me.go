package academic

import (
	"context"

	"github.com/kampusops/backend/internal/domain"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/httpx"
)

// Jadwal returns the student's classes in the active period.
func (s *Service) Jadwal(ctx context.Context) ([]map[string]Kelas, error) {
	m, err := StudentByUser(ctx, s.pool, appctx.UserID(ctx))
	if err != nil {
		return nil, err
	}
	ap, err := ActivePeriode(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	ks, err := QueryKelas(ctx, s.pool, `k.id IN (SELECT d.kelas_id FROM krs_detail d JOIN krs ON krs.id=d.krs_id
		WHERE krs.tenant_id=$1 AND krs.mahasiswa_id=$2 AND krs.periode_id=$3 AND d.status='aktif') ORDER BY k.hari, k.jam_mulai`, m.ID, ap.ID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]Kelas, 0, len(ks))
	for _, k := range ks {
		out = append(out, map[string]Kelas{"kelas": k})
	}
	return out, nil
}

// Periodes lists the periods in which the student has a KRS.
func (s *Service) Periodes(ctx context.Context) ([]Periode, error) {
	m, err := StudentByUser(ctx, s.pool, appctx.UserID(ctx))
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT `+periodeCols+` FROM periode p WHERE p.tenant_id=$1 AND
		EXISTS (SELECT 1 FROM krs WHERE krs.periode_id=p.id AND krs.mahasiswa_id=$2) ORDER BY p.kode DESC`, appctx.TenantID(ctx), m.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Periode{}
	for rows.Next() {
		p, err := scanPeriode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// KHS (FR-23, EC-06): the period's own grades (repeats shown as that semester's grade);
// IPK is cumulative over best grades up to and including the period.
func (s *Service) KHS(ctx context.Context, periodeID string) (Khs, error) {
	m, err := StudentByUser(ctx, s.pool, appctx.UserID(ctx))
	if err != nil {
		return Khs{}, err
	}
	return s.KHSFor(ctx, m, periodeID)
}

func (s *Service) KHSFor(ctx context.Context, m Mahasiswa, periodeID string) (Khs, error) {
	var p Periode
	var err error
	if periodeID == "" {
		// default: latest period with final grades, else the active one
		var pid string
		err = s.pool.QueryRow(ctx, `SELECT k.periode_id FROM nilai n JOIN kelas k ON k.id=n.kelas_id JOIN periode p ON p.id=k.periode_id
			WHERE n.tenant_id=$1 AND n.mahasiswa_id=$2 AND n.final_at IS NOT NULL ORDER BY p.kode DESC LIMIT 1`, appctx.TenantID(ctx), m.ID).Scan(&pid)
		if err == nil {
			p, err = GetPeriode(ctx, s.pool, pid)
		} else {
			p, err = ActivePeriode(ctx, s.pool)
		}
	} else {
		p, err = GetPeriode(ctx, s.pool, periodeID)
	}
	if err != nil {
		return Khs{}, err
	}
	hist, err := GradeHistory(ctx, s.pool, m.ID)
	if err != nil {
		return Khs{}, err
	}
	rows, err := s.pool.Query(ctx, `SELECT mk.id, mk.kode, mk.nama, d.sks, CASE WHEN n.final_at IS NOT NULL THEN n.huruf END,
		CASE WHEN n.final_at IS NOT NULL THEN n.bobot_x100 END, d.is_ulang
		FROM krs JOIN krs_detail d ON d.krs_id=krs.id AND d.status='aktif' JOIN kelas k ON k.id=d.kelas_id JOIN mata_kuliah mk ON mk.id=k.mk_id
		LEFT JOIN nilai n ON n.krs_detail_id=d.id
		WHERE krs.tenant_id=$1 AND krs.mahasiswa_id=$2 AND krs.periode_id=$3 ORDER BY mk.kode`, appctx.TenantID(ctx), m.ID, p.ID)
	if err != nil {
		return Khs{}, err
	}
	defer rows.Close()
	out := Khs{Periode: p, Rows: []KhsRow{}}
	var sem []domain.GradeEntry
	for rows.Next() {
		var r KhsRow
		var mkID string
		if err := rows.Scan(&mkID, &r.MkKode, &r.MkNama, &r.Sks, &r.Huruf, &r.BobotX100, &r.IsUlang); err != nil {
			return Khs{}, err
		}
		out.Rows = append(out.Rows, r)
		out.TotalSks += r.Sks
		if r.BobotX100 != nil {
			sem = append(sem, domain.GradeEntry{MkID: mkID, Sks: r.Sks, BobotX100: *r.BobotX100})
		}
	}
	if err := rows.Err(); err != nil {
		return Khs{}, err
	}
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return Khs{}, err
	}
	out.IPS = domain.IPS(sem)
	num, den, _ := t.Rules.Cumulative(Entries(hist, func(g GradeRecord) bool { return g.PeriodeKode <= p.Kode }), nil)
	out.IPK = domain.FormatIP(num, den)
	return out, nil
}

// Transkrip (FR-24, EC-06): best grade per course.
func (s *Service) Transkrip(ctx context.Context) (Transkrip, error) {
	m, err := StudentByUser(ctx, s.pool, appctx.UserID(ctx))
	if err != nil {
		return Transkrip{}, err
	}
	return s.TranskripFor(ctx, m)
}

func (s *Service) TranskripFor(ctx context.Context, m Mahasiswa) (Transkrip, error) {
	hist, err := GradeHistory(ctx, s.pool, m.ID)
	if err != nil {
		return Transkrip{}, err
	}
	entries := Entries(hist, nil)
	taken := map[string]int{}
	for _, e := range entries {
		taken[e.MkID]++
	}
	best := domain.BestGrades(entries, nil)
	out := Transkrip{Mahasiswa: m, Rows: []TranskripRow{}}
	for _, b := range best {
		h, bb := b.Huruf, b.BobotX100
		out.Rows = append(out.Rows, TranskripRow{
			KhsRow:      KhsRow{MkKode: b.MkKode, MkNama: b.MkNama, Sks: b.Sks, Huruf: &h, BobotX100: &bb, IsUlang: taken[b.MkID] > 1},
			Semester:    b.Semester,
			PeriodeKode: b.PeriodeKode,
		})
		out.TotalSks += b.Sks
	}
	num, den := domain.Sum(best)
	out.IPK = domain.FormatIP(num, den)
	return out, nil
}

func (s *Service) Profil(ctx context.Context) (map[string]any, error) {
	m, err := StudentByUser(ctx, s.pool, appctx.UserID(ctx))
	if err != nil {
		return nil, err
	}
	r, err := Riwayat(ctx, s.pool, m.ID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"mahasiswa": m, "riwayat": r}, nil
}

// ensure import use
var _ = httpx.NotFound
