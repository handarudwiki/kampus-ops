package reporting

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kampusops/backend/internal/academic"
	"github.com/kampusops/backend/internal/audit"
	"github.com/kampusops/backend/internal/domain"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/httpx"
)

type Service struct {
	pool *pgxpool.Pool
	Now  func() time.Time
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool, Now: time.Now} }

// statusCTE yields each student's status in period $2 (latest riwayat_status at or before it).
const statusCTE = `st AS (
	SELECT DISTINCT ON (r.mahasiswa_id) r.mahasiswa_id, r.status FROM riwayat_status r JOIN periode p ON p.id=r.periode_id
	WHERE r.tenant_id=$1 AND p.kode <= (SELECT kode FROM periode WHERE id=$2)
	ORDER BY r.mahasiswa_id, p.kode DESC, r.created_at DESC, r.id DESC)`

// LoadSnapshot reads everything the validator needs for one period.
func LoadSnapshot(ctx context.Context, q db.Querier, periodeID string) (Snapshot, error) {
	tid := appctx.TenantID(ctx)
	s := Snapshot{PeriodeID: periodeID, NIKCount: map[string]int{}}
	rows, err := q.Query(ctx, `WITH `+statusCTE+`
		SELECT m.id, m.nim, m.nama, m.nik, coalesce(st.status, m.status), krs.id,
			(SELECT count(*) FROM krs_detail d WHERE d.krs_id=krs.id AND d.status='aktif'),
			coalesce(krs.total_sks,0), coalesce(krs.max_sks,0),
			EXISTS (SELECT 1 FROM dispensasi ds WHERE ds.mahasiswa_id=m.id AND ds.periode_id=$2 AND ds.status='disetujui')
		FROM mahasiswa m LEFT JOIN st ON st.mahasiswa_id=m.id LEFT JOIN krs ON krs.mahasiswa_id=m.id AND krs.periode_id=$2
		WHERE m.tenant_id=$1 ORDER BY m.nim`, tid, periodeID)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var st Student
		if err := rows.Scan(&st.ID, &st.NIM, &st.Nama, &st.NIK, &st.Status, &st.KrsID, &st.KrsItems, &st.KrsTotalSks, &st.KrsMaxSks, &st.HasDispensasi); err != nil {
			rows.Close()
			return s, err
		}
		if st.NIK != nil && *st.NIK != "" {
			s.NIKCount[*st.NIK]++
		}
		s.Students = append(s.Students, st)
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT n.id, n.kelas_id, n.mahasiswa_id, m.nim, m.nama, mk.nama,
		EXISTS (SELECT 1 FROM krs_detail d JOIN krs ON krs.id=d.krs_id WHERE d.id=n.krs_detail_id AND d.status='aktif'
			AND d.kelas_id=n.kelas_id AND krs.mahasiswa_id=n.mahasiswa_id),
		n.perlu_lapor_ulang
		FROM nilai n JOIN kelas k ON k.id=n.kelas_id JOIN mata_kuliah mk ON mk.id=k.mk_id JOIN mahasiswa m ON m.id=n.mahasiswa_id
		WHERE n.tenant_id=$1 AND k.periode_id=$2`, tid, periodeID)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var n NilaiRec
		if err := rows.Scan(&n.ID, &n.KelasID, &n.MahasiswaID, &n.NIM, &n.Nama, &n.MkNama, &n.MatchingKrs, &n.PerluLaporUlang); err != nil {
			rows.Close()
			return s, err
		}
		s.Nilai = append(s.Nilai, n)
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT k.id, mk.kode || ' ' || mk.nama || ' — ' || k.nama, k.dosen_id, k.sks, mk.sks
		FROM kelas k JOIN mata_kuliah mk ON mk.id=k.mk_id WHERE k.tenant_id=$1 AND k.periode_id=$2 AND k.status='dibuka' ORDER BY mk.kode`, tid, periodeID)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var k KelasRec
		if err := rows.Scan(&k.ID, &k.Label, &k.DosenID, &k.Sks, &k.MkSks); err != nil {
			return s, err
		}
		s.Kelas = append(s.Kelas, k)
	}
	return s, rows.Err()
}

func (s *Service) resolvePeriode(ctx context.Context, periodeID string) (academic.Periode, error) {
	if periodeID == "" {
		return academic.ActivePeriode(ctx, s.pool)
	}
	if _, err := httpx.OptUUID("periode_id", periodeID); err != nil {
		return academic.Periode{}, err
	}
	p, err := academic.GetPeriode(ctx, s.pool, periodeID)
	if err != nil {
		return p, httpx.NotFound("Periode tidak ditemukan")
	}
	return p, nil
}

// Validate runs the validator and stores the run (validasi_run + validasi_pddikti rows).
func (s *Service) Validate(ctx context.Context, periodeID string) (Result, error) {
	p, err := s.resolvePeriode(ctx, periodeID)
	if err != nil {
		return Result{}, err
	}
	snap, err := LoadSnapshot(ctx, s.pool, p.ID)
	if err != nil {
		return Result{}, err
	}
	res := Validate(snap, s.Now())
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		b, _ := json.Marshal(res)
		var runID string
		var ranBy *string
		if uid := appctx.UserID(ctx); uid != "" {
			ranBy = &uid
		}
		if err := tx.QueryRow(ctx, `INSERT INTO validasi_run (tenant_id, periode_id, score, errors, warnings, result, ran_by) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
			tid, p.ID, res.Score, res.Errors, res.Warnings, b, ranBy).Scan(&runID); err != nil {
			return err
		}
		if len(res.Items) == 0 {
			return nil
		}
		rules, ents, ids, lvls, msgs := make([]string, len(res.Items)), make([]string, len(res.Items)), make([]string, len(res.Items)), make([]string, len(res.Items)), make([]string, len(res.Items))
		for i, it := range res.Items {
			rules[i], ents[i], ids[i], lvls[i], msgs[i] = it.Rule, it.Entity, it.EntityID, it.Level, it.Message
		}
		_, err := tx.Exec(ctx, `INSERT INTO validasi_pddikti (tenant_id, run_id, periode_id, aturan, entitas, entitas_id, tingkat, pesan)
			SELECT $1, $2, $3, a, e, i::uuid, l, m FROM unnest($4::text[], $5::text[], $6::text[], $7::text[], $8::text[]) AS u(a, e, i, l, m)`,
			tid, runID, p.ID, rules, ents, ids, lvls, msgs)
		return err
	})
	return res, err
}

// Latest returns the most recent stored run or nil.
func (s *Service) Latest(ctx context.Context, periodeID string) (*Result, error) {
	p, err := s.resolvePeriode(ctx, periodeID)
	if err != nil {
		return nil, err
	}
	var b []byte
	err = s.pool.QueryRow(ctx, `SELECT result FROM validasi_run WHERE tenant_id=$1 AND periode_id=$2 ORDER BY ran_at DESC LIMIT 1`, appctx.TenantID(ctx), p.ID).Scan(&b)
	if db.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var r Result
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

var exportEntities = map[string]bool{"mahasiswa": true, "kelas": true, "krs": true, "nilai": true, "aktivitas": true}

// Export produces Feeder-like CSV; blocked with PDDIKTI_BLOCKED while errors exist (FR-30).
func (s *Service) Export(ctx context.Context, entity, periodeID string) ([]byte, string, error) {
	if !exportEntities[entity] {
		return nil, "", httpx.NotFound("Entitas ekspor tidak dikenal")
	}
	p, err := s.resolvePeriode(ctx, periodeID)
	if err != nil {
		return nil, "", err
	}
	snap, err := LoadSnapshot(ctx, s.pool, p.ID)
	if err != nil {
		return nil, "", err
	}
	res := Validate(snap, s.Now())
	if res.Errors > 0 {
		return nil, "", httpx.Rule("PDDIKTI_BLOCKED", "Ekspor diblokir: masih ada "+strconv.Itoa(res.Errors)+" error validasi PDDikti").
			With(map[string]any{"errors": res.Errors, "warnings": res.Warnings, "by_rule": res.ByRule})
	}
	tid := appctx.TenantID(ctx)
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	var q string
	switch entity {
	case "mahasiswa":
		_ = w.Write([]string{"nim", "nama_mahasiswa", "nik", "kode_prodi", "angkatan", "status_mahasiswa"})
		q = `WITH ` + statusCTE + ` SELECT m.nim, m.nama, coalesce(m.nik,''), pr.kode, m.angkatan::text, coalesce(st.status, m.status)
			FROM mahasiswa m JOIN prodi pr ON pr.id=m.prodi_id LEFT JOIN st ON st.mahasiswa_id=m.id WHERE m.tenant_id=$1 AND coalesce(st.status, m.status) IN ('aktif','cuti','non_aktif') ORDER BY m.nim`
	case "kelas":
		_ = w.Write([]string{"kode_mk", "nama_mk", "nama_kelas", "sks", "nidn_dosen", "kuota", "peserta"})
		q = `SELECT mk.kode, mk.nama, k.nama, k.sks::text, coalesce(d.nidn,''), k.kuota::text, k.terisi::text FROM kelas k JOIN mata_kuliah mk ON mk.id=k.mk_id
			LEFT JOIN dosen d ON d.id=k.dosen_id WHERE k.tenant_id=$1 AND k.periode_id=$2 AND k.status='dibuka' ORDER BY mk.kode, k.nama`
	case "krs":
		_ = w.Write([]string{"nim", "kode_mk", "nama_kelas", "sks", "status_krs"})
		q = `SELECT m.nim, mk.kode, k.nama, d.sks::text, krs.status FROM krs JOIN krs_detail d ON d.krs_id=krs.id AND d.status='aktif'
			JOIN kelas k ON k.id=d.kelas_id JOIN mata_kuliah mk ON mk.id=k.mk_id JOIN mahasiswa m ON m.id=krs.mahasiswa_id
			WHERE krs.tenant_id=$1 AND krs.periode_id=$2 ORDER BY m.nim, mk.kode`
	case "nilai":
		_ = w.Write([]string{"nim", "kode_mk", "nama_kelas", "nilai_angka", "nilai_huruf", "nilai_indeks"})
		q = `SELECT m.nim, mk.kode, k.nama, coalesce(to_char(n.angka_x100/100.0,'FM990.00'),''), coalesce(n.huruf,''), coalesce(to_char(n.bobot_x100/100.0,'FM0.00'),'')
			FROM nilai n JOIN kelas k ON k.id=n.kelas_id JOIN mata_kuliah mk ON mk.id=k.mk_id JOIN mahasiswa m ON m.id=n.mahasiswa_id
			WHERE n.tenant_id=$1 AND k.periode_id=$2 AND n.final_at IS NOT NULL ORDER BY m.nim, mk.kode`
	case "aktivitas":
		_ = w.Write([]string{"nim", "status_mahasiswa", "ips", "sks_semester", "ipk", "sks_total"})
		q = `WITH ` + statusCTE + `, sem AS (
				SELECT n.mahasiswa_id, sum(k.sks * n.bobot_x100) AS num, sum(k.sks) AS den FROM nilai n JOIN kelas k ON k.id=n.kelas_id
				WHERE n.tenant_id=$1 AND k.periode_id=$2 AND n.final_at IS NOT NULL GROUP BY n.mahasiswa_id)
			SELECT m.nim, coalesce(st.status, m.status), coalesce(sem.num,0)::text, coalesce(sem.den,0)::text, m.ipk_num::text, m.ipk_den::text
			FROM mahasiswa m LEFT JOIN st ON st.mahasiswa_id=m.id LEFT JOIN sem ON sem.mahasiswa_id=m.id
			WHERE m.tenant_id=$1 AND coalesce(st.status, m.status) IN ('aktif','cuti','non_aktif') ORDER BY m.nim`
	}
	args := []any{tid, p.ID}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			rows.Close()
			return nil, "", err
		}
		rec := make([]string, len(vals))
		for i, v := range vals {
			rec[i], _ = v.(string)
		}
		if entity == "aktivitas" { // render IPS/IPK with decimal rounding (EC-10)
			num, _ := strconv.ParseInt(rec[2], 10, 64)
			den, _ := strconv.ParseInt(rec[3], 10, 64)
			inum, _ := strconv.ParseInt(rec[4], 10, 64)
			iden, _ := strconv.ParseInt(rec[5], 10, 64)
			rec[2], rec[4] = domain.FormatIP(num, den), domain.FormatIP(inum, iden)
		}
		_ = w.Write(rec)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	w.Flush()
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE periode SET exported_at=now(), updated_at=now() WHERE tenant_id=$1 AND id=$2`, tid, p.ID); err != nil {
			return err
		}
		if entity == "nilai" { // re-reported corrections are now cleared (EC-11 / PV-08)
			if _, err := tx.Exec(ctx, `UPDATE nilai n SET perlu_lapor_ulang=false, updated_at=now() FROM kelas k
				WHERE k.id=n.kelas_id AND n.tenant_id=$1 AND k.periode_id=$2 AND n.perlu_lapor_ulang`, tid, p.ID); err != nil {
				return err
			}
		}
		return audit.Write(ctx, tx, "periode", p.ID, "pddikti_export", nil, map[string]any{"entity": entity, "warnings": res.Warnings})
	})
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "pddikti_" + entity + "_" + p.Kode + ".csv", nil
}
