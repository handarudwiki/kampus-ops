package academic

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/kampusops/backend/internal/domain"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/httpx"
)

// ---- periode ----

const periodeCols = `p.id, p.kode, p.tahun, p.tipe, p.status, p.tahapan, iso(p.exported_at)`

func scanPeriode(row pgx.Row) (Periode, error) {
	var p Periode
	var raw []byte
	err := row.Scan(&p.ID, &p.Kode, &p.Tahun, &p.Tipe, &p.Status, &raw, &p.ExportedAt)
	p.Tahapan = parseTahapan(raw)
	return p, err
}

// ActivePeriode returns the tenant's active period.
func ActivePeriode(ctx context.Context, q db.Querier) (Periode, error) {
	p, err := scanPeriode(q.QueryRow(ctx, `SELECT `+periodeCols+` FROM periode p WHERE p.tenant_id=$1 AND p.status='aktif'`, appctx.TenantID(ctx)))
	if db.IsNoRows(err) {
		return p, httpx.NotFound("Tidak ada periode aktif")
	}
	return p, err
}

func GetPeriode(ctx context.Context, q db.Querier, id string) (Periode, error) {
	return scanPeriode(q.QueryRow(ctx, `SELECT `+periodeCols+` FROM periode p WHERE p.tenant_id=$1 AND p.id=$2`, appctx.TenantID(ctx), id))
}

// PreviousRegularPeriode returns the latest Ganjil/Genap period before kode, if any.
func PreviousRegularPeriode(ctx context.Context, q db.Querier, kode string) (*Periode, error) {
	p, err := scanPeriode(q.QueryRow(ctx, `SELECT `+periodeCols+` FROM periode p WHERE p.tenant_id=$1 AND p.kode < $2
		AND p.tipe <> 'Pendek' ORDER BY p.kode DESC LIMIT 1`, appctx.TenantID(ctx), kode))
	if db.IsNoRows(err) {
		return nil, nil
	}
	return &p, err
}

// ---- kelas ----

const KelasCols = `k.id, k.periode_id, k.mk_id, mk.kode, mk.nama, k.sks, k.nama, k.dosen_id, d.nama, k.ruang_id, r.nama,
	k.hari, to_char(k.jam_mulai,'HH24:MI'), to_char(k.jam_selesai,'HH24:MI'), k.kuota, k.terisi, k.status`

const KelasFrom = ` FROM kelas k JOIN mata_kuliah mk ON mk.id = k.mk_id
	LEFT JOIN dosen d ON d.id = k.dosen_id LEFT JOIN ruang r ON r.id = k.ruang_id `

func ScanKelas(row pgx.Row) (Kelas, error) {
	var k Kelas
	err := row.Scan(&k.ID, &k.PeriodeID, &k.MkID, &k.MkKode, &k.MkNama, &k.Sks, &k.Nama, &k.DosenID, &k.DosenNama,
		&k.RuangID, &k.RuangNama, &k.Slot.Hari, &k.Slot.JamMulai, &k.Slot.JamSelesai, &k.Kuota, &k.Terisi, &k.Status)
	return k, err
}

// QueryKelas runs a kelas select with extra WHERE (must reference $1 = tenant).
func QueryKelas(ctx context.Context, q db.Querier, where string, args ...any) ([]Kelas, error) {
	rows, err := q.Query(ctx, `SELECT `+KelasCols+KelasFrom+` WHERE k.tenant_id = $1 AND `+where,
		append([]any{appctx.TenantID(ctx)}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Kelas{}
	for rows.Next() {
		k, err := ScanKelas(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func GetKelas(ctx context.Context, q db.Querier, id string) (Kelas, error) {
	return ScanKelas(q.QueryRow(ctx, `SELECT `+KelasCols+KelasFrom+` WHERE k.tenant_id=$1 AND k.id=$2`, appctx.TenantID(ctx), id))
}

// KelasDosenHistory loads the EC-12 assignment history.
func KelasDosenHistory(ctx context.Context, q db.Querier, kelasID string) ([]DosenAssignment, error) {
	rows, err := q.Query(ctx, `SELECT dosen_id, efektif_mulai::text, efektif_selesai::text FROM kelas_dosen
		WHERE tenant_id=$1 AND kelas_id=$2 ORDER BY efektif_mulai`, appctx.TenantID(ctx), kelasID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DosenAssignment
	for rows.Next() {
		var a DosenAssignment
		if err := rows.Scan(&a.DosenID, &a.EfektifMulai, &a.EfektifSelesai); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// DosenIDByUser returns the dosen record linked to a user ("" when none).
func DosenIDByUser(ctx context.Context, q db.Querier, userID string) (string, error) {
	var id string
	err := q.QueryRow(ctx, `SELECT id FROM dosen WHERE tenant_id=$1 AND user_id=$2`, appctx.TenantID(ctx), userID).Scan(&id)
	if db.IsNoRows(err) {
		return "", nil
	}
	return id, err
}

// ---- mahasiswa ----

const mhsCols = `m.id, m.nim, m.nik, m.nama, m.email, m.prodi_id, p.nama, m.angkatan, m.kurikulum_id, m.dosen_wali_id, d.nama,
	m.status, m.ipk_num, m.ipk_den, m.sks_lulus, m.user_id, p.masa_studi_maks`
const mhsFrom = ` FROM mahasiswa m JOIN prodi p ON p.id = m.prodi_id LEFT JOIN dosen d ON d.id = m.dosen_wali_id `

func scanMhs(row pgx.Row) (Mahasiswa, error) {
	var m Mahasiswa
	var num, den int64
	err := row.Scan(&m.ID, &m.NIM, &m.NIK, &m.Nama, &m.Email, &m.ProdiID, &m.ProdiNama, &m.Angkatan, &m.KurikulumID,
		&m.DosenWaliID, &m.DosenWaliNama, &m.Status, &num, &den, &m.SksLulus, &m.UserID, &m.MasaStudiMaks)
	m.IPK = domain.FormatIP(num, den)
	return m, err
}

// QueryMahasiswa runs a mahasiswa select with extra WHERE ($1 = tenant) and tail (ORDER/LIMIT).
func QueryMahasiswa(ctx context.Context, q db.Querier, where, tail string, args ...any) ([]Mahasiswa, error) {
	rows, err := q.Query(ctx, `SELECT `+mhsCols+mhsFrom+` WHERE m.tenant_id=$1 AND `+where+` `+tail,
		append([]any{appctx.TenantID(ctx)}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Mahasiswa{}
	for rows.Next() {
		m, err := scanMhs(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	fillSemester(ctx, q, out)
	return out, nil
}

func fillSemester(ctx context.Context, q db.Querier, ms []Mahasiswa) {
	ap, err := ActivePeriode(ctx, q)
	if err != nil {
		return
	}
	for i := range ms {
		ms[i].SemesterKe = domain.SemesterKe(ms[i].Angkatan, ap.Tahun, ap.Tipe)
	}
}

func GetMahasiswa(ctx context.Context, q db.Querier, id string) (Mahasiswa, error) {
	ms, err := QueryMahasiswa(ctx, q, `m.id = $2`, ``, id)
	if err != nil {
		return Mahasiswa{}, err
	}
	if len(ms) == 0 {
		return Mahasiswa{}, httpx.NotFound("Mahasiswa tidak ditemukan")
	}
	return ms[0], nil
}

// StudentByUser resolves the mahasiswa of the logged-in user.
func StudentByUser(ctx context.Context, q db.Querier, userID string) (Mahasiswa, error) {
	ms, err := QueryMahasiswa(ctx, q, `m.user_id = $2`, ``, userID)
	if err != nil {
		return Mahasiswa{}, err
	}
	if len(ms) == 0 {
		return Mahasiswa{}, httpx.NotFound("Data mahasiswa untuk akun ini tidak ditemukan")
	}
	return ms[0], nil
}

// CurrentStatus reads the student's status from riwayat_status (latest row) — the
// source of truth for KRS, billing and PDDikti rules (BRD §5.6).
func CurrentStatus(ctx context.Context, q db.Querier, mhsID string) (string, error) {
	var s string
	err := q.QueryRow(ctx, `SELECT status FROM riwayat_status WHERE tenant_id=$1 AND mahasiswa_id=$2
		ORDER BY created_at DESC, id DESC LIMIT 1`, appctx.TenantID(ctx), mhsID).Scan(&s)
	if db.IsNoRows(err) {
		return domain.StatusAktif, nil
	}
	return s, err
}

// CutiCount counts distinct periods spent on cuti, excluding exceptPeriode.
func CutiCount(ctx context.Context, q db.Querier, mhsID, exceptPeriode string) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT count(DISTINCT periode_id) FROM riwayat_status WHERE tenant_id=$1 AND mahasiswa_id=$2
		AND status='cuti' AND periode_id::text <> $3`, appctx.TenantID(ctx), mhsID, exceptPeriode).Scan(&n)
	return n, err
}

func Riwayat(ctx context.Context, q db.Querier, mhsID string) ([]RiwayatStatus, error) {
	rows, err := q.Query(ctx, `SELECT p.kode, r.status, r.alasan, iso(r.created_at) FROM riwayat_status r
		JOIN periode p ON p.id = r.periode_id WHERE r.tenant_id=$1 AND r.mahasiswa_id=$2 ORDER BY r.created_at DESC, r.id DESC`,
		appctx.TenantID(ctx), mhsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RiwayatStatus{}
	for rows.Next() {
		var r RiwayatStatus
		if err := rows.Scan(&r.PeriodeKode, &r.Status, &r.Alasan, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ---- grades ----

// GradeRecord is a final grade with its KRS context.
type GradeRecord struct {
	domain.GradeEntry
	PeriodeID string
	KelasID   string
}

// GradeHistory returns all FINAL grades of a student (ordered by period).
func GradeHistory(ctx context.Context, q db.Querier, mhsID string) ([]GradeRecord, error) {
	rows, err := q.Query(ctx, `SELECT mk.id, mk.kode, mk.nama, k.sks, n.bobot_x100, n.huruf, p.kode, mk.semester, p.id, k.id
		FROM nilai n JOIN kelas k ON k.id = n.kelas_id JOIN mata_kuliah mk ON mk.id = k.mk_id JOIN periode p ON p.id = k.periode_id
		WHERE n.tenant_id=$1 AND n.mahasiswa_id=$2 AND n.final_at IS NOT NULL AND n.bobot_x100 IS NOT NULL
		ORDER BY p.kode, mk.kode`, appctx.TenantID(ctx), mhsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GradeRecord
	for rows.Next() {
		var g GradeRecord
		if err := rows.Scan(&g.MkID, &g.MkKode, &g.MkNama, &g.Sks, &g.BobotX100, &g.Huruf, &g.PeriodeKode, &g.Semester, &g.PeriodeID, &g.KelasID); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// Entries converts grade records to domain entries, optionally only for one period
// or up to (and including) a period kode.
func Entries(gs []GradeRecord, filter func(GradeRecord) bool) []domain.GradeEntry {
	out := []domain.GradeEntry{}
	for _, g := range gs {
		if filter == nil || filter(g) {
			out = append(out, g.GradeEntry)
		}
	}
	return out
}

// RecomputeStudent refreshes the cached IPK numbers and passed SKS (run in the same
// tx as grade finalisation / correction approval).
func RecomputeStudent(ctx context.Context, q db.Querier, rules domain.Rules, mhsID string) error {
	gs, err := GradeHistory(ctx, q, mhsID)
	if err != nil {
		return err
	}
	num, den, lulus := rules.Cumulative(Entries(gs, nil), nil)
	_, err = q.Exec(ctx, `UPDATE mahasiswa SET ipk_num=$3, ipk_den=$4, sks_lulus=$5, updated_at=now() WHERE id=$1 AND tenant_id=$2`,
		mhsID, appctx.TenantID(ctx), num, den, lulus)
	return err
}

// ---- helpers ----

var allowedTables = map[string]bool{"fakultas": true, "prodi": true, "dosen": true, "ruang": true, "periode": true,
	"mahasiswa": true, "kurikulum": true, "mata_kuliah": true, "kelas": true, "users": true, "gelombang": true}

// MustExist returns a VALIDATION_ERROR on field when id is not a row of table in this
// tenant (prevents referencing another tenant's rows through plain FKs).
func MustExist(ctx context.Context, q db.Querier, table, field, id string) error {
	if !allowedTables[table] {
		panic("MustExist: table not allowed " + table)
	}
	if strings.TrimSpace(id) == "" {
		return httpx.ValidationField(field, "wajib diisi")
	}
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM `+table+` WHERE tenant_id=$1 AND id::text=$2)`, appctx.TenantID(ctx), id).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return httpx.ValidationField(field, "tidak ditemukan")
	}
	return nil
}
