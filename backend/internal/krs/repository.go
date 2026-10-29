package krs

import (
	"context"

	"github.com/kampusops/backend/internal/academic"
	"github.com/kampusops/backend/internal/domain"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/db"
)

type krsRow struct {
	ID, MahasiswaID, PeriodeID, Status string
	TotalSks, MaxSks                   int
	IpsLalu, Catatan                   *string
	SubmittedAt, DecidedAt             *string
}

const krsCols = `id, mahasiswa_id, periode_id, status, total_sks, max_sks, ips_lalu, catatan, iso(submitted_at), iso(decided_at)`

func scanKrs(row interface{ Scan(...any) error }) (krsRow, error) {
	var k krsRow
	err := row.Scan(&k.ID, &k.MahasiswaID, &k.PeriodeID, &k.Status, &k.TotalSks, &k.MaxSks, &k.IpsLalu, &k.Catatan, &k.SubmittedAt, &k.DecidedAt)
	return k, err
}

func findKrs(ctx context.Context, q db.Querier, mhsID, periodeID string, forUpdate bool) (krsRow, error) {
	sql := `SELECT ` + krsCols + ` FROM krs WHERE tenant_id=$1 AND mahasiswa_id=$2 AND periode_id=$3`
	if forUpdate {
		sql += ` FOR UPDATE`
	}
	return scanKrs(q.QueryRow(ctx, sql, appctx.TenantID(ctx), mhsID, periodeID))
}

func getKrs(ctx context.Context, q db.Querier, id string, forUpdate bool) (krsRow, error) {
	sql := `SELECT ` + krsCols + ` FROM krs WHERE tenant_id=$1 AND id=$2`
	if forUpdate {
		sql += ` FOR UPDATE`
	}
	return scanKrs(q.QueryRow(ctx, sql, appctx.TenantID(ctx), id))
}

type itemRow struct {
	Kelas   academic.Kelas
	IsUlang bool
}

func krsItems(ctx context.Context, q db.Querier, krsID string) ([]itemRow, error) {
	rows, err := q.Query(ctx, `SELECT `+academic.KelasCols+`, kd.is_ulang`+academic.KelasFrom+`
		JOIN krs_detail kd ON kd.kelas_id = k.id AND kd.status='aktif'
		WHERE kd.tenant_id=$1 AND kd.krs_id=$2 ORDER BY k.hari, k.jam_mulai`, appctx.TenantID(ctx), krsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []itemRow{}
	for rows.Next() {
		var it itemRow
		k := &it.Kelas
		if err := rows.Scan(&k.ID, &k.PeriodeID, &k.MkID, &k.MkKode, &k.MkNama, &k.Sks, &k.Nama, &k.DosenID, &k.DosenNama,
			&k.RuangID, &k.RuangNama, &k.Slot.Hari, &k.Slot.JamMulai, &k.Slot.JamSelesai, &k.Kuota, &k.Terisi, &k.Status, &it.IsUlang); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// bills loads mandatory bills (ukt, praktikum) of the period with installment due sums.
func bills(ctx context.Context, q db.Querier, mhsID, periodeID, today string) ([]Bill, error) {
	rows, err := q.Query(ctx, `SELECT t.jenis, t.status, t.terbayar,
		coalesce((SELECT sum(c.jumlah) FROM cicilan c WHERE c.tagihan_id=t.id AND c.jatuh_tempo <= $4::date), 0),
		EXISTS (SELECT 1 FROM cicilan c WHERE c.tagihan_id=t.id)
		FROM tagihan t WHERE t.tenant_id=$1 AND t.mahasiswa_id=$2 AND t.periode_id=$3 AND t.jenis IN ('ukt','praktikum')`,
		appctx.TenantID(ctx), mhsID, periodeID, today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Bill
	for rows.Next() {
		var b Bill
		if err := rows.Scan(&b.Jenis, &b.Status, &b.Terbayar, &b.CicilanDue, &b.HasCicilan); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func approvedDispensasi(ctx context.Context, q db.Querier, mhsID, periodeID string) ([]string, error) {
	rows, err := q.Query(ctx, `SELECT DISTINCT jenis FROM dispensasi WHERE tenant_id=$1 AND mahasiswa_id=$2 AND periode_id=$3 AND status='disetujui'`,
		appctx.TenantID(ctx), mhsID, periodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func equivPairs(ctx context.Context, q db.Querier) (map[string][]string, error) {
	rows, err := q.Query(ctx, `SELECT mk_lama_id, mk_baru_id FROM ekuivalensi WHERE tenant_id=$1`, appctx.TenantID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pairs [][2]string
	for rows.Next() {
		var a, b string
		if err := rows.Scan(&a, &b); err != nil {
			return nil, err
		}
		pairs = append(pairs, [2]string{a, b})
	}
	return BuildEquiv(pairs), rows.Err()
}

// prereqsFor returns prerequisites keyed by course id for the given course ids.
func prereqsFor(ctx context.Context, q db.Querier, r domain.Rules, mkIDs []string) (map[string][]Prereq, error) {
	out := map[string][]Prereq{}
	if len(mkIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `SELECT pr.mk_id, s.id, s.kode, s.nama, pr.nilai_min FROM prasyarat pr JOIN mata_kuliah s ON s.id=pr.mk_syarat_id
		WHERE pr.tenant_id=$1 AND pr.mk_id = ANY($2::uuid[])`, appctx.TenantID(ctx), mkIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var mk string
		var p Prereq
		if err := rows.Scan(&mk, &p.MkID, &p.Kode, &p.Nama, &p.NilaiMin); err != nil {
			return nil, err
		}
		if p.NilaiMin != nil {
			if b, ok := r.BobotOf(*p.NilaiMin); ok {
				p.MinBobot = &b
			}
		}
		out[mk] = append(out[mk], p)
	}
	return out, rows.Err()
}
