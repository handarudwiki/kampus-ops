// Package dashboard builds the generic, role-specific Dashboard payload (GET /dashboard).
package dashboard

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kampusops/backend/internal/academic"
	"github.com/kampusops/backend/internal/domain"
	"github.com/kampusops/backend/internal/krs"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/httpx"
	"github.com/kampusops/backend/internal/reporting"
	"github.com/kampusops/backend/internal/tenant"
)

type StatCard struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Value string  `json:"value"`
	Hint  *string `json:"hint"`
	Tone  string  `json:"tone"`
	Trend *string `json:"trend"`
}

type Alert struct {
	Level   string  `json:"level"`
	Message string  `json:"message"`
	Link    *string `json:"link"`
}

type Series struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
}

type Chart struct {
	Title  string   `json:"title"`
	Labels []string `json:"labels"`
	Series []Series `json:"series"`
}

type Column struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type Table struct {
	Title   string              `json:"title"`
	Columns []Column            `json:"columns"`
	Rows    []map[string]string `json:"rows"`
	Link    *string             `json:"link"`
}

type Dashboard struct {
	Stats  []StatCard `json:"stats"`
	Alerts []Alert    `json:"alerts"`
	Chart  *Chart     `json:"chart"`
	Table  *Table     `json:"table"`
}

func sp(s string) *string { return &s }

func stat(key, label, value, hint, tone string) StatCard {
	var h *string
	if hint != "" {
		h = &hint
	}
	return StatCard{Key: key, Label: label, Value: value, Hint: h, Tone: tone}
}

// Rupiah formats 12345000 as "Rp 12.345.000".
func Rupiah(v int64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := strconv.FormatInt(v, 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-Rp " + b.String()
	}
	return "Rp " + b.String()
}

type Service struct {
	pool    *pgxpool.Pool
	tenants *tenant.Store
	krs     *krs.Service
	acad    *academic.Service
	Now     func() time.Time
}

func NewService(pool *pgxpool.Pool, tenants *tenant.Store, k *krs.Service, a *academic.Service) *Service {
	return &Service{pool: pool, tenants: tenants, krs: k, acad: a, Now: time.Now}
}

func (s *Service) Handle(c *fiber.Ctx) error {
	d, err := s.Build(c.UserContext())
	if err != nil {
		return err
	}
	return httpx.OK(c, d)
}

func (s *Service) Build(ctx context.Context) (Dashboard, error) {
	p, _ := appctx.PrincipalFrom(ctx)
	var d Dashboard
	var err error
	switch p.ActiveRole {
	case "mahasiswa":
		d, err = s.mahasiswa(ctx)
	case "dosen_wali":
		d, err = s.dosenWali(ctx)
	case "dosen":
		d, err = s.dosen(ctx)
	case "kaprodi":
		d, err = s.kaprodi(ctx, p.ScopeProdi("kaprodi"))
	case "operator":
		d, err = s.operator(ctx)
	case "keuangan":
		d, err = s.keuangan(ctx)
	case "admin_kampus":
		d, err = s.admin(ctx)
	case "super_admin":
		d, err = s.superAdmin(ctx)
	case "calon_mahasiswa":
		d, err = s.calon(ctx)
	}
	if d.Stats == nil {
		d.Stats = []StatCard{}
	}
	if d.Alerts == nil {
		d.Alerts = []Alert{}
	}
	return d, err
}

func (s *Service) count(ctx context.Context, sql string, args ...any) int {
	var n int
	_ = s.pool.QueryRow(ctx, sql, append([]any{appctx.TenantID(ctx)}, args...)...).Scan(&n)
	return n
}

func (s *Service) sum(ctx context.Context, sql string, args ...any) int64 {
	var n int64
	_ = s.pool.QueryRow(ctx, sql, append([]any{appctx.TenantID(ctx)}, args...)...).Scan(&n)
	return n
}

// ---- mahasiswa ----

func (s *Service) mahasiswa(ctx context.Context) (Dashboard, error) {
	var d Dashboard
	m, err := academic.StudentByUser(ctx, s.pool, appctx.UserID(ctx))
	if err != nil {
		return d, err
	}
	mine, err := s.krs.GetMine(ctx)
	if err != nil {
		return d, err
	}
	outstanding := s.sum(ctx, `SELECT coalesce(sum(greatest(jumlah-terbayar,0)),0)::bigint FROM tagihan WHERE tenant_id=$1 AND mahasiswa_id=$2`, m.ID)
	krsStatus, krsHint, tone := "Belum ada", "", "amber"
	if mine.Krs != nil {
		krsStatus = strings.ToUpper(mine.Krs.Status[:1]) + mine.Krs.Status[1:]
		krsHint = fmt.Sprintf("%d / %d SKS", mine.Krs.TotalSks, mine.Krs.MaxSks)
		tone = map[string]string{"draft": "amber", "diajukan": "blue", "disetujui": "green", "ditolak": "red"}[mine.Krs.Status]
	}
	tagTone := "green"
	if outstanding > 0 {
		tagTone = "red"
	}
	d.Stats = []StatCard{
		stat("ipk", "IPK", m.IPK, fmt.Sprintf("Semester %d", m.SemesterKe), "blue"),
		stat("sks_lulus", "SKS Lulus", strconv.Itoa(m.SksLulus), "", "violet"),
		stat("tagihan", "Sisa Tagihan", Rupiah(outstanding), "", tagTone),
		stat("krs", "Status KRS", krsStatus, krsHint, tone),
	}
	if mine.Window.Open {
		d.Alerts = append(d.Alerts, Alert{"info", "Masa " + strings.ToUpper(mine.Window.Phase) + " sedang dibuka sampai " + mine.Window.End, sp("/krs")})
	}
	for _, r := range mine.Eligibility.Reasons {
		switch r {
		case krs.ErrUnpaid:
			d.Alerts = append(d.Alerts, Alert{"error", "Tagihan wajib belum lunas — KRS terkunci. Lunasi atau ajukan dispensasi/cicilan.", sp("/tagihan")})
		case krs.ErrNotActive:
			d.Alerts = append(d.Alerts, Alert{"warning", "Status Anda bukan aktif; lakukan aktivasi sebelum mengisi KRS.", sp("/akademik/profil")})
		}
	}
	if mine.Krs != nil && mine.Krs.Status == "ditolak" && mine.Krs.Catatan != nil {
		d.Alerts = append(d.Alerts, Alert{"warning", "KRS ditolak: " + *mine.Krs.Catatan, sp("/krs")})
	}
	// IPS trend
	hist, err := academic.GradeHistory(ctx, s.pool, m.ID)
	if err != nil {
		return d, err
	}
	var labels []string
	var vals []float64
	seen := map[string]bool{}
	for _, g := range hist {
		if seen[g.PeriodeKode] {
			continue
		}
		seen[g.PeriodeKode] = true
		kode := g.PeriodeKode
		num, den := domain.Sum(academic.Entries(hist, func(x academic.GradeRecord) bool { return x.PeriodeKode == kode }))
		f, _ := domain.IPDecimal(num, den).Round(2).Float64()
		labels = append(labels, kode)
		vals = append(vals, f)
	}
	if len(labels) > 0 {
		d.Chart = &Chart{Title: "IPS per semester", Labels: labels, Series: []Series{{Name: "IPS", Values: vals}}}
	}
	t := &Table{Title: "Jadwal kuliah semester ini", Columns: []Column{{"hari", "Hari"}, {"jam", "Jam"}, {"mk", "Mata kuliah"}, {"kelas", "Kelas"}, {"ruang", "Ruang"}}, Rows: []map[string]string{}, Link: sp("/akademik/jadwal")}
	hari := []string{"", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	if mine.Krs != nil {
		for _, it := range mine.Krs.Items {
			r := ""
			if it.Kelas.RuangNama != nil {
				r = *it.Kelas.RuangNama
			}
			t.Rows = append(t.Rows, map[string]string{"hari": hari[it.Kelas.Slot.Hari], "jam": it.Kelas.Slot.JamMulai + "–" + it.Kelas.Slot.JamSelesai,
				"mk": it.Kelas.MkNama, "kelas": it.Kelas.Nama, "ruang": r})
		}
	}
	d.Table = t
	return d, nil
}

// ---- dosen wali ----

func (s *Service) dosenWali(ctx context.Context) (Dashboard, error) {
	var d Dashboard
	did, err := academic.DosenIDByUser(ctx, s.pool, appctx.UserID(ctx))
	if err != nil {
		return d, err
	}
	ap, err := academic.ActivePeriode(ctx, s.pool)
	if err != nil {
		return d, err
	}
	advisees := s.count(ctx, `SELECT count(*) FROM mahasiswa WHERE tenant_id=$1 AND dosen_wali_id=$2 AND status='aktif'`, did)
	byStatus := map[string]int{}
	rows, err := s.pool.Query(ctx, `SELECT krs.status, count(*) FROM krs JOIN mahasiswa m ON m.id=krs.mahasiswa_id WHERE krs.tenant_id=$1 AND krs.periode_id=$2 AND m.dosen_wali_id=$3 GROUP BY 1`,
		appctx.TenantID(ctx), ap.ID, did)
	if err != nil {
		return d, err
	}
	total := 0
	for rows.Next() {
		var st string
		var n int
		_ = rows.Scan(&st, &n)
		byStatus[st] = n
		total += n
	}
	rows.Close()
	t, _ := s.tenants.Current(ctx)
	deadline := krs.ApprovalDeadline(ap.Tahapan, t.Loc, t.Rules.KRSGraceDays).Add(-time.Second)
	d.Stats = []StatCard{
		stat("advisees", "Mahasiswa bimbingan", strconv.Itoa(advisees), "", "blue"),
		stat("pending", "KRS menunggu persetujuan", strconv.Itoa(byStatus["diajukan"]), "Batas proses "+deadline.In(t.Loc).Format("02 Jan 2006"), "amber"),
		stat("approved", "KRS disetujui", strconv.Itoa(byStatus["disetujui"]), "", "green"),
		stat("no_krs", "Belum mengisi KRS", strconv.Itoa(max(advisees-total, 0)), "", "red"),
	}
	if byStatus["diajukan"] > 0 {
		d.Alerts = append(d.Alerts, Alert{"warning", fmt.Sprintf("%d KRS menunggu persetujuan Anda (grace period H+%d).", byStatus["diajukan"], t.Rules.KRSGraceDays), sp("/krs/advisees")})
	}
	d.Chart = &Chart{Title: "Status KRS bimbingan", Labels: []string{"Draft", "Diajukan", "Disetujui", "Ditolak", "Belum KRS"},
		Series: []Series{{Name: "Mahasiswa", Values: []float64{float64(byStatus["draft"]), float64(byStatus["diajukan"]), float64(byStatus["disetujui"]), float64(byStatus["ditolak"]), float64(max(advisees-total, 0))}}}}
	tbl := &Table{Title: "KRS menunggu persetujuan", Columns: []Column{{"nim", "NIM"}, {"nama", "Nama"}, {"sks", "SKS"}, {"diajukan", "Diajukan"}}, Rows: []map[string]string{}, Link: sp("/krs/advisees")}
	rows, err = s.pool.Query(ctx, `SELECT krs.id, m.nim, m.nama, krs.total_sks, coalesce(iso(krs.submitted_at),'') FROM krs JOIN mahasiswa m ON m.id=krs.mahasiswa_id
		WHERE krs.tenant_id=$1 AND krs.periode_id=$2 AND m.dosen_wali_id=$3 AND krs.status='diajukan' ORDER BY krs.submitted_at LIMIT 10`, appctx.TenantID(ctx), ap.ID, did)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, nim, nama, sub string
		var sks int
		_ = rows.Scan(&id, &nim, &nama, &sks, &sub)
		tbl.Rows = append(tbl.Rows, map[string]string{"id": id, "nim": nim, "nama": nama, "sks": strconv.Itoa(sks), "diajukan": sub})
	}
	d.Table = tbl
	return d, nil
}

// ---- dosen ----

func (s *Service) dosen(ctx context.Context) (Dashboard, error) {
	var d Dashboard
	did, err := academic.DosenIDByUser(ctx, s.pool, appctx.UserID(ctx))
	if err != nil {
		return d, err
	}
	ap, err := academic.ActivePeriode(ctx, s.pool)
	if err != nil {
		return d, err
	}
	ks, err := academic.QueryKelas(ctx, s.pool, `k.periode_id=$2 AND k.dosen_id::text=$3 AND k.status='dibuka' ORDER BY k.hari, k.jam_mulai`, ap.ID, did)
	if err != nil {
		return d, err
	}
	students := 0
	pending := 0
	tbl := &Table{Title: "Kelas yang diampu", Columns: []Column{{"mk", "Mata kuliah"}, {"kelas", "Kelas"}, {"jadwal", "Jadwal"}, {"peserta", "Peserta"}, {"nilai", "Nilai"}}, Rows: []map[string]string{}, Link: sp("/teaching")}
	ch := &Chart{Title: "Peserta per kelas", Series: []Series{{Name: "Peserta", Values: []float64{}}, {Name: "Kuota", Values: []float64{}}}}
	hari := []string{"", "Sen", "Sel", "Rab", "Kam", "Jum", "Sab"}
	for _, k := range ks {
		students += k.Terisi
		final := s.count(ctx, `SELECT count(*) FROM nilai WHERE tenant_id=$1 AND kelas_id=$2 AND final_at IS NOT NULL`, k.ID) > 0
		status := "Final"
		if !final {
			pending++
			status = "Belum final"
		}
		tbl.Rows = append(tbl.Rows, map[string]string{"id": k.ID, "mk": k.MkNama, "kelas": k.Nama, "jadwal": hari[k.Slot.Hari] + " " + k.Slot.JamMulai, "peserta": fmt.Sprintf("%d/%d", k.Terisi, k.Kuota), "nilai": status})
		ch.Labels = append(ch.Labels, k.MkKode+" "+k.Nama)
		ch.Series[0].Values = append(ch.Series[0].Values, float64(k.Terisi))
		ch.Series[1].Values = append(ch.Series[1].Values, float64(k.Kuota))
	}
	pert := s.count(ctx, `SELECT count(*) FROM pertemuan WHERE tenant_id=$1 AND dosen_id::text=$2`, did)
	d.Stats = []StatCard{
		stat("kelas", "Kelas diampu", strconv.Itoa(len(ks)), ap.Kode, "blue"),
		stat("mahasiswa", "Total peserta", strconv.Itoa(students), "", "violet"),
		stat("nilai_pending", "Nilai belum final", strconv.Itoa(pending), "Batas "+ap.Tahapan.NilaiSelesai, "amber"),
		stat("pertemuan", "Pertemuan tercatat", strconv.Itoa(pert), "", "green"),
	}
	if pending > 0 {
		d.Alerts = append(d.Alerts, Alert{"info", fmt.Sprintf("%d kelas belum difinalisasi nilainya (batas %s).", pending, ap.Tahapan.NilaiSelesai), sp("/teaching")})
	}
	if n := s.count(ctx, `SELECT count(*) FROM koreksi_nilai WHERE tenant_id=$1 AND diajukan_oleh=$2 AND status='diajukan'`, appctx.UserID(ctx)); n > 0 {
		d.Alerts = append(d.Alerts, Alert{"info", fmt.Sprintf("%d pengajuan koreksi nilai menunggu kaprodi.", n), sp("/koreksi-nilai")})
	}
	if len(ks) > 0 {
		d.Chart = ch
	}
	d.Table = tbl
	return d, nil
}

// ---- kaprodi ----

func (s *Service) kaprodi(ctx context.Context, scope *string) (Dashboard, error) {
	var d Dashboard
	prodi := ""
	if scope != nil {
		prodi = *scope
	}
	warn, err := s.acad.MasaStudiWarnings(ctx, scope)
	if err != nil {
		return d, err
	}
	doCand := 0
	for _, w := range warn {
		if w.Flag == "do_candidate" {
			doCand++
		}
	}
	aktif := s.count(ctx, `SELECT count(*) FROM mahasiswa WHERE tenant_id=$1 AND status='aktif' AND ($2='' OR prodi_id::text=$2)`, prodi)
	koreksi := s.count(ctx, `SELECT count(*) FROM koreksi_nilai kn JOIN nilai n ON n.id=kn.nilai_id JOIN kelas k ON k.id=n.kelas_id JOIN mata_kuliah mk ON mk.id=k.mk_id
		JOIN kurikulum ku ON ku.id=mk.kurikulum_id WHERE kn.tenant_id=$1 AND kn.status='diajukan' AND ($2='' OR ku.prodi_id::text=$2)`, prodi)
	d.Stats = []StatCard{
		stat("aktif", "Mahasiswa aktif", strconv.Itoa(aktif), "", "blue"),
		stat("masa_studi", "Peringatan masa studi", strconv.Itoa(len(warn)-doCand), "Sisa ≤ 2 semester", "amber"),
		stat("do", "Kandidat DO", strconv.Itoa(doCand), "Masa studi terlampaui", "red"),
		stat("koreksi", "Koreksi nilai pending", strconv.Itoa(koreksi), "", "violet"),
	}
	if koreksi > 0 {
		d.Alerts = append(d.Alerts, Alert{"warning", fmt.Sprintf("%d pengajuan koreksi nilai menunggu persetujuan.", koreksi), sp("/koreksi-nilai")})
	}
	if doCand > 0 {
		d.Alerts = append(d.Alerts, Alert{"error", fmt.Sprintf("%d mahasiswa melampaui batas masa studi (keputusan DO tetap manual).", doCand), sp("/akademik/masa-studi")})
	}
	rows, err := s.pool.Query(ctx, `SELECT angkatan::text, count(*) FROM mahasiswa WHERE tenant_id=$1 AND status IN ('aktif','cuti') AND ($2='' OR prodi_id::text=$2) GROUP BY angkatan ORDER BY angkatan`,
		appctx.TenantID(ctx), prodi)
	if err != nil {
		return d, err
	}
	ch := &Chart{Title: "Mahasiswa per angkatan", Series: []Series{{Name: "Mahasiswa", Values: []float64{}}}}
	for rows.Next() {
		var a string
		var n int
		_ = rows.Scan(&a, &n)
		ch.Labels = append(ch.Labels, a)
		ch.Series[0].Values = append(ch.Series[0].Values, float64(n))
	}
	rows.Close()
	d.Chart = ch
	tbl := &Table{Title: "Peringatan masa studi", Columns: []Column{{"nim", "NIM"}, {"nama", "Nama"}, {"angkatan", "Angkatan"}, {"sisa", "Sisa semester"}}, Rows: []map[string]string{}, Link: sp("/akademik/masa-studi")}
	for i, w := range warn {
		if i >= 10 {
			break
		}
		tbl.Rows = append(tbl.Rows, map[string]string{"id": w.ID, "nim": w.NIM, "nama": w.Nama, "angkatan": strconv.Itoa(w.Angkatan), "sisa": strconv.Itoa(w.SisaSemester)})
	}
	d.Table = tbl
	return d, nil
}

// ---- operator ----

func (s *Service) operator(ctx context.Context) (Dashboard, error) {
	var d Dashboard
	ap, err := academic.ActivePeriode(ctx, s.pool)
	if err != nil {
		return d, err
	}
	rep := reporting.NewService(s.pool)
	last, err := rep.Latest(ctx, ap.ID)
	if err != nil {
		return d, err
	}
	score, hint := "-", "Belum divalidasi"
	if last != nil {
		score = strconv.Itoa(last.Score) + "%"
		hint = fmt.Sprintf("%d error, %d peringatan", last.Errors, last.Warnings)
	}
	kelas := s.count(ctx, `SELECT count(*) FROM kelas WHERE tenant_id=$1 AND periode_id=$2 AND status='dibuka'`, ap.ID)
	noDosen := s.count(ctx, `SELECT count(*) FROM kelas WHERE tenant_id=$1 AND periode_id=$2 AND status='dibuka' AND dosen_id IS NULL`, ap.ID)
	aktif := s.count(ctx, `SELECT count(*) FROM mahasiswa WHERE tenant_id=$1 AND status='aktif'`)
	byStatus := map[string]int{}
	rows, err := s.pool.Query(ctx, `SELECT status, count(*) FROM krs WHERE tenant_id=$1 AND periode_id=$2 GROUP BY 1`, appctx.TenantID(ctx), ap.ID)
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var st string
		var n int
		_ = rows.Scan(&st, &n)
		byStatus[st] = n
	}
	rows.Close()
	done := byStatus["diajukan"] + byStatus["disetujui"]
	pct := 0
	if aktif > 0 {
		pct = done * 100 / aktif
	}
	tone := "green"
	if last == nil || last.Errors > 0 {
		tone = "red"
	}
	d.Stats = []StatCard{
		stat("pddikti", "Skor kesiapan PDDikti", score, hint, tone),
		stat("kelas", "Kelas dibuka", strconv.Itoa(kelas), fmt.Sprintf("%d tanpa dosen", noDosen), "blue"),
		stat("krs_progress", "Progres KRS", strconv.Itoa(pct)+"%", fmt.Sprintf("%d dari %d mahasiswa aktif", done, aktif), "violet"),
		stat("krs_pending", "KRS menunggu wali", strconv.Itoa(byStatus["diajukan"]), "", "amber"),
	}
	if last != nil && last.Errors > 0 {
		d.Alerts = append(d.Alerts, Alert{"error", fmt.Sprintf("Validasi PDDikti: %d error memblokir ekspor.", last.Errors), sp("/pddikti")})
	} else if last == nil {
		d.Alerts = append(d.Alerts, Alert{"info", "Jalankan validasi PDDikti untuk periode " + ap.Kode + ".", sp("/pddikti")})
	}
	if noDosen > 0 {
		d.Alerts = append(d.Alerts, Alert{"warning", fmt.Sprintf("%d kelas belum memiliki dosen pengampu.", noDosen), sp("/kelas")})
	}
	d.Chart = &Chart{Title: "Status KRS " + ap.Kode, Labels: []string{"Draft", "Diajukan", "Disetujui", "Ditolak", "Belum KRS"},
		Series: []Series{{Name: "KRS", Values: []float64{float64(byStatus["draft"]), float64(byStatus["diajukan"]), float64(byStatus["disetujui"]), float64(byStatus["ditolak"]),
			float64(max(aktif-byStatus["draft"]-done-byStatus["ditolak"], 0))}}}}
	tbl := &Table{Title: "Hasil validasi PDDikti per aturan", Columns: []Column{{"rule", "Aturan"}, {"label", "Keterangan"}, {"level", "Tingkat"}, {"count", "Jumlah"}}, Rows: []map[string]string{}, Link: sp("/pddikti")}
	if last != nil {
		for _, r := range last.ByRule {
			tbl.Rows = append(tbl.Rows, map[string]string{"rule": r.Rule, "label": r.Label, "level": r.Level, "count": strconv.Itoa(r.Count)})
		}
	}
	d.Table = tbl
	return d, nil
}

// ---- keuangan ----

func (s *Service) keuangan(ctx context.Context) (Dashboard, error) {
	var d Dashboard
	ap, err := academic.ActivePeriode(ctx, s.pool)
	if err != nil {
		return d, err
	}
	var total, paid, piutang int64
	var n, lunas int
	_ = s.pool.QueryRow(ctx, `SELECT coalesce(sum(jumlah),0)::bigint, coalesce(sum(least(terbayar,jumlah)),0)::bigint, coalesce(sum(greatest(jumlah-terbayar,0)),0)::bigint,
		count(*), count(*) FILTER (WHERE status='lunas') FROM tagihan WHERE tenant_id=$1 AND periode_id=$2`, appctx.TenantID(ctx), ap.ID).Scan(&total, &paid, &piutang, &n, &lunas)
	pct := 0
	if n > 0 {
		pct = lunas * 100 / n
	}
	disp := s.count(ctx, `SELECT count(*) FROM dispensasi WHERE tenant_id=$1 AND status='diajukan'`)
	d.Stats = []StatCard{
		stat("piutang", "Total piutang", Rupiah(piutang), ap.Kode, "red"),
		stat("terbayar", "Terbayar", Rupiah(paid), "dari "+Rupiah(total), "green"),
		stat("lunas_pct", "Tagihan lunas", strconv.Itoa(pct)+"%", fmt.Sprintf("%d dari %d tagihan", lunas, n), "blue"),
		stat("dispensasi", "Dispensasi menunggu", strconv.Itoa(disp), "", "amber"),
	}
	if disp > 0 {
		d.Alerts = append(d.Alerts, Alert{"warning", fmt.Sprintf("%d pengajuan dispensasi menunggu keputusan.", disp), sp("/dispensasi")})
	}
	today := domain.LocalDate(s.Now(), appctx.Tenant(ctx).Loc)
	if overdue := s.count(ctx, `SELECT count(*) FROM tagihan WHERE tenant_id=$1 AND status<>'lunas' AND jatuh_tempo < $2::date`, today); overdue > 0 {
		d.Alerts = append(d.Alerts, Alert{"error", fmt.Sprintf("%d tagihan melewati jatuh tempo.", overdue), sp("/tagihan?status=belum_bayar")})
	}
	rows, err := s.pool.Query(ctx, `SELECT p.kode, coalesce(sum(t.jumlah),0)::bigint, coalesce(sum(least(t.terbayar,t.jumlah)),0)::bigint FROM tagihan t
		JOIN mahasiswa m ON m.id=t.mahasiswa_id JOIN prodi p ON p.id=m.prodi_id WHERE t.tenant_id=$1 AND t.periode_id=$2 GROUP BY p.kode ORDER BY p.kode`, appctx.TenantID(ctx), ap.ID)
	if err != nil {
		return d, err
	}
	ch := &Chart{Title: "Tagihan vs terbayar per prodi (juta Rp)", Series: []Series{{Name: "Tagihan", Values: []float64{}}, {Name: "Terbayar", Values: []float64{}}}}
	for rows.Next() {
		var k string
		var t, p int64
		_ = rows.Scan(&k, &t, &p)
		ch.Labels = append(ch.Labels, k)
		ch.Series[0].Values = append(ch.Series[0].Values, float64(t/100000)/10)
		ch.Series[1].Values = append(ch.Series[1].Values, float64(p/100000)/10)
	}
	rows.Close()
	d.Chart = ch
	tbl := &Table{Title: "Tunggakan terbesar", Columns: []Column{{"nim", "NIM"}, {"nama", "Nama"}, {"jenis", "Jenis"}, {"sisa", "Sisa"}, {"status", "Status"}}, Rows: []map[string]string{}, Link: sp("/tagihan")}
	rows, err = s.pool.Query(ctx, `SELECT t.id, m.nim, m.nama, t.jenis, (t.jumlah-t.terbayar)::bigint, t.status FROM tagihan t JOIN mahasiswa m ON m.id=t.mahasiswa_id
		WHERE t.tenant_id=$1 AND t.periode_id=$2 AND t.status<>'lunas' ORDER BY t.jumlah-t.terbayar DESC, m.nim LIMIT 10`, appctx.TenantID(ctx), ap.ID)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, nim, nama, jenis, st string
		var sisa int64
		_ = rows.Scan(&id, &nim, &nama, &jenis, &sisa, &st)
		tbl.Rows = append(tbl.Rows, map[string]string{"id": id, "nim": nim, "nama": nama, "jenis": jenis, "sisa": Rupiah(sisa), "status": st})
	}
	d.Table = tbl
	return d, nil
}

// ---- admin kampus ----

func (s *Service) admin(ctx context.Context) (Dashboard, error) {
	var d Dashboard
	d.Stats = []StatCard{
		stat("mahasiswa", "Mahasiswa", strconv.Itoa(s.count(ctx, `SELECT count(*) FROM mahasiswa WHERE tenant_id=$1`)), fmt.Sprintf("%d aktif", s.count(ctx, `SELECT count(*) FROM mahasiswa WHERE tenant_id=$1 AND status='aktif'`)), "blue"),
		stat("dosen", "Dosen", strconv.Itoa(s.count(ctx, `SELECT count(*) FROM dosen WHERE tenant_id=$1`)), "", "violet"),
		stat("prodi", "Program studi", strconv.Itoa(s.count(ctx, `SELECT count(*) FROM prodi WHERE tenant_id=$1`)), "", "green"),
		stat("users", "Pengguna", strconv.Itoa(s.count(ctx, `SELECT count(*) FROM users WHERE tenant_id=$1`)), "", "pink"),
	}
	if _, err := academic.ActivePeriode(ctx, s.pool); err != nil {
		d.Alerts = append(d.Alerts, Alert{"warning", "Belum ada periode aktif.", sp("/periode")})
	}
	rows, err := s.pool.Query(ctx, `SELECT p.kode, count(m.id) FROM prodi p LEFT JOIN mahasiswa m ON m.prodi_id=p.id AND m.status='aktif' WHERE p.tenant_id=$1 GROUP BY p.kode ORDER BY p.kode`, appctx.TenantID(ctx))
	if err != nil {
		return d, err
	}
	ch := &Chart{Title: "Mahasiswa aktif per prodi", Series: []Series{{Name: "Mahasiswa", Values: []float64{}}}}
	for rows.Next() {
		var k string
		var n int
		_ = rows.Scan(&k, &n)
		ch.Labels = append(ch.Labels, k)
		ch.Series[0].Values = append(ch.Series[0].Values, float64(n))
	}
	rows.Close()
	d.Chart = ch
	tbl := &Table{Title: "Aktivitas terbaru", Columns: []Column{{"waktu", "Waktu"}, {"actor", "Pengguna"}, {"aksi", "Aksi"}, {"entitas", "Entitas"}}, Rows: []map[string]string{}, Link: sp("/audit-log")}
	rows, err = s.pool.Query(ctx, `SELECT iso(created_at), actor_nama, aksi, entitas FROM audit_log WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT 10`, appctx.TenantID(ctx))
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		var w, a, k, e string
		_ = rows.Scan(&w, &a, &k, &e)
		tbl.Rows = append(tbl.Rows, map[string]string{"waktu": w, "actor": a, "aksi": k, "entitas": e})
	}
	d.Table = tbl
	return d, nil
}

// ---- super admin ----

func (s *Service) superAdmin(ctx context.Context) (Dashboard, error) {
	var d Dashboard
	rows, err := s.pool.Query(ctx, `SELECT t.kode, t.nama, t.timezone, t.plan, (SELECT count(*) FROM mahasiswa m WHERE m.tenant_id=t.id),
		(SELECT count(*) FROM users u WHERE u.tenant_id=t.id) FROM tenant t ORDER BY t.kode`)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	ch := &Chart{Title: "Mahasiswa per tenant", Series: []Series{{Name: "Mahasiswa", Values: []float64{}}}}
	tbl := &Table{Title: "Tenant kampus", Columns: []Column{{"kode", "Kode"}, {"nama", "Nama"}, {"timezone", "Zona waktu"}, {"plan", "Paket"}, {"mahasiswa", "Mahasiswa"}}, Rows: []map[string]string{}, Link: sp("/tenants")}
	var tenants, mhs, users int
	for rows.Next() {
		var k, n, tz, plan string
		var m, u int
		_ = rows.Scan(&k, &n, &tz, &plan, &m, &u)
		tenants++
		mhs += m
		users += u
		ch.Labels = append(ch.Labels, k)
		ch.Series[0].Values = append(ch.Series[0].Values, float64(m))
		tbl.Rows = append(tbl.Rows, map[string]string{"kode": k, "nama": n, "timezone": tz, "plan": plan, "mahasiswa": strconv.Itoa(m)})
	}
	d.Stats = []StatCard{
		stat("tenants", "Tenant kampus", strconv.Itoa(tenants), "", "blue"),
		stat("mahasiswa", "Total mahasiswa", strconv.Itoa(mhs), "Semua tenant", "violet"),
		stat("users", "Total pengguna", strconv.Itoa(users), "", "green"),
	}
	d.Chart, d.Table = ch, tbl
	return d, nil
}

// ---- calon mahasiswa ----

func (s *Service) calon(ctx context.Context) (Dashboard, error) {
	var d Dashboard
	var id, no, status, gel, prodi string
	var nim *string
	var berkas []byte
	err := s.pool.QueryRow(ctx, `SELECT p.id, p.no_daftar, p.status, g.nama, pr.nama, p.nim, p.berkas FROM pendaftar p JOIN gelombang g ON g.id=p.gelombang_id
		JOIN prodi pr ON pr.id=p.prodi_pilihan_id WHERE p.tenant_id=$1 AND p.user_id=$2 ORDER BY p.created_at DESC LIMIT 1`, appctx.TenantID(ctx), appctx.UserID(ctx)).
		Scan(&id, &no, &status, &gel, &prodi, &nim, &berkas)
	if err != nil {
		d.Alerts = append(d.Alerts, Alert{"info", "Anda belum memiliki data pendaftaran.", sp("/pmb")})
		return d, nil
	}
	var tagStatus string
	_ = s.pool.QueryRow(ctx, `SELECT status FROM tagihan WHERE tenant_id=$1 AND pendaftar_id=$2 AND jenis='formulir_pmb'`, appctx.TenantID(ctx), id).Scan(&tagStatus)
	nimS := "-"
	if nim != nil {
		nimS = *nim
	}
	d.Stats = []StatCard{
		stat("status", "Status pendaftaran", status, gel, "blue"),
		stat("no_daftar", "No. pendaftaran", no, prodi, "violet"),
		stat("formulir", "Biaya formulir", tagStatus, "", map[bool]string{true: "green", false: "amber"}[tagStatus == "lunas"]),
		stat("nim", "NIM", nimS, "", "green"),
	}
	next := map[string]string{
		"daftar": "Bayar biaya formulir melalui virtual account.", "bayar_formulir": "Lengkapi berkas pendaftaran.",
		"lengkapi_berkas": "Kirim berkas untuk diverifikasi.", "verifikasi": "Berkas sedang diverifikasi panitia.",
		"lulus": "Selamat! Anda lulus seleksi — lakukan daftar ulang.", "tidak_lulus": "Mohon maaf, Anda belum lulus seleksi.",
		"daftar_ulang": "Daftar ulang selesai. Login menggunakan NIM Anda.",
	}[status]
	d.Alerts = append(d.Alerts, Alert{"info", next, sp("/pmb")})
	steps := []string{"daftar", "bayar_formulir", "lengkapi_berkas", "verifikasi", "lulus", "daftar_ulang"}
	tbl := &Table{Title: "Tahapan PMB", Columns: []Column{{"tahap", "Tahap"}, {"status", "Status"}}, Rows: []map[string]string{}, Link: sp("/pmb")}
	reached := true
	for _, st := range steps {
		state := "Selesai"
		if st == status {
			state = "Saat ini"
			reached = false
		} else if !reached {
			state = "-"
		}
		tbl.Rows = append(tbl.Rows, map[string]string{"tahap": st, "status": state})
	}
	d.Table = tbl
	return d, nil
}
