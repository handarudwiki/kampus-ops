package krs

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kampusops/backend/internal/academic"
	"github.com/kampusops/backend/internal/audit"
	"github.com/kampusops/backend/internal/domain"
	"github.com/kampusops/backend/internal/notification"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/httpx"
	"github.com/kampusops/backend/internal/tenant"
)

var (
	quotaFull = prometheus.NewCounter(prometheus.CounterOpts{Name: "krs_quota_full_total", Help: "KRS add rejected because the class was full (EC-01)"})
	itemsAdded = prometheus.NewCounter(prometheus.CounterOpts{Name: "krs_items_added_total", Help: "KRS items added"})
)

func init() { prometheus.MustRegister(quotaFull, itemsAdded) }

type Service struct {
	pool    *pgxpool.Pool
	tenants *tenant.Store
	Now     func() time.Time
}

func NewService(pool *pgxpool.Pool, tenants *tenant.Store) *Service {
	return &Service{pool: pool, tenants: tenants, Now: time.Now}
}

// ---- views (contract shapes) ----

type MhsBrief struct {
	ID        string `json:"id"`
	NIM       string `json:"nim"`
	Nama      string `json:"nama"`
	ProdiNama string `json:"prodi_nama"`
	Angkatan  int    `json:"angkatan"`
	IPK       string `json:"ipk"`
}

type Item struct {
	Kelas   academic.Kelas `json:"kelas"`
	IsUlang bool           `json:"is_ulang"`
}

type Krs struct {
	ID          string   `json:"id"`
	Mahasiswa   MhsBrief `json:"mahasiswa"`
	PeriodeID   string   `json:"periode_id"`
	Status      string   `json:"status"`
	TotalSks    int      `json:"total_sks"`
	MaxSks      int      `json:"max_sks"`
	IpsLalu     *string  `json:"ips_lalu"`
	Catatan     *string  `json:"catatan"`
	SubmittedAt *string  `json:"submitted_at"`
	DecidedAt   *string  `json:"decided_at"`
	Items       []Item   `json:"items"`
}

type WindowView struct {
	Open  bool   `json:"open"`
	Phase string `json:"phase"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type EligibilityView struct {
	OK      bool     `json:"ok"`
	Reasons []string `json:"reasons"`
}

type MyKrs struct {
	Krs         *Krs        `json:"krs"`
	Window      WindowView  `json:"window"`
	Eligibility EligibilityView `json:"eligibility"`
}

type Offering struct {
	academic.Kelas
	Eligible bool     `json:"eligible"`
	Reasons  []string `json:"reasons"`
	InKrs    bool     `json:"in_krs"`
}

func brief(m academic.Mahasiswa) MhsBrief {
	return MhsBrief{ID: m.ID, NIM: m.NIM, Nama: m.Nama, ProdiNama: m.ProdiNama, Angkatan: m.Angkatan, IPK: m.IPK}
}

func viewWindow(w Window) WindowView {
	end := w.End
	if !end.IsZero() {
		end = end.Add(-time.Second) // present the inclusive last second
	}
	return WindowView{Open: w.Open, Phase: w.Phase, Start: w.Start.UTC().Format(time.RFC3339), End: end.UTC().Format(time.RFC3339)}
}

func (s *Service) view(ctx context.Context, q db.Querier, k krsRow, m academic.Mahasiswa) (Krs, error) {
	items, err := krsItems(ctx, q, k.ID)
	if err != nil {
		return Krs{}, err
	}
	out := Krs{ID: k.ID, Mahasiswa: brief(m), PeriodeID: k.PeriodeID, Status: k.Status, TotalSks: k.TotalSks, MaxSks: k.MaxSks,
		IpsLalu: k.IpsLalu, Catatan: k.Catatan, SubmittedAt: k.SubmittedAt, DecidedAt: k.DecidedAt, Items: []Item{}}
	for _, it := range items {
		out.Items = append(out.Items, Item{Kelas: it.Kelas, IsUlang: it.IsUlang})
	}
	return out, nil
}

// ---- student context ----

type studentCtx struct {
	M       academic.Mahasiswa
	P       academic.Periode
	T       tenant.Full
	Window  Window
	Status  string
	Disp    []string
	Reasons []string
	Best    map[string]int
	Equiv   map[string][]string
	MaxSKS  int
	IpsLalu *string
}

func (sc *studentCtx) susulan() bool { return has(sc.Disp, "krs_susulan") }

// load gathers every input the KRS rules need (reads go through q so they see the tx).
func (s *Service) load(ctx context.Context, q db.Querier, m academic.Mahasiswa) (*studentCtx, error) {
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return nil, err
	}
	p, err := academic.ActivePeriode(ctx, q)
	if err != nil {
		return nil, err
	}
	now := s.Now()
	sc := &studentCtx{M: m, P: p, T: t, Window: EvaluateWindow(p.Tahapan, now, t.Loc)}
	if sc.Status, err = academic.CurrentStatus(ctx, q, m.ID); err != nil {
		return nil, err
	}
	if sc.Disp, err = approvedDispensasi(ctx, q, m.ID, p.ID); err != nil {
		return nil, err
	}
	bs, err := bills(ctx, q, m.ID, p.ID, domain.LocalDate(now, t.Loc))
	if err != nil {
		return nil, err
	}
	sc.Reasons = Eligibility(EligibilityInput{Window: sc.Window, StudentStatus: sc.Status, Bills: bs, ApprovedDispensasi: sc.Disp})
	hist, err := academic.GradeHistory(ctx, q, m.ID)
	if err != nil {
		return nil, err
	}
	prior := academic.Entries(hist, func(g academic.GradeRecord) bool { return g.PeriodeKode < p.Kode })
	sc.Best = BestBobot(prior)
	if sc.Equiv, err = equivPairs(ctx, q); err != nil {
		return nil, err
	}
	// previous IPS = latest earlier regular period with final grades (FR-17)
	last := ""
	for _, g := range hist {
		if g.PeriodeKode < p.Kode && g.PeriodeTipe != "Pendek" && g.PeriodeKode > last {
			last = g.PeriodeKode
		}
	}
	var prevX100 *int
	if last != "" {
		num, den := domain.Sum(academic.Entries(hist, func(g academic.GradeRecord) bool { return g.PeriodeKode == last }))
		v := domain.IPx100(num, den)
		prevX100 = &v
		str := domain.FormatIP(num, den)
		sc.IpsLalu = &str
	}
	sc.MaxSKS = MaxSKS(t.Rules, p.Tipe, prevX100)
	return sc, nil
}

func reasonError(code string) *httpx.AppError {
	msgs := map[string]string{
		ErrWindowClosed: "Jendela KRS sedang ditutup",
		ErrNotActive:    "Status mahasiswa tidak aktif pada periode ini",
		ErrUnpaid:       "Tagihan wajib belum lunas dan tidak ada dispensasi/cicilan yang disetujui",
	}
	return httpx.Rule(code, msgs[code])
}

func (s *Service) me(ctx context.Context) (academic.Mahasiswa, error) {
	return academic.StudentByUser(ctx, s.pool, appctx.UserID(ctx))
}

// GetMine: GET /krs/me
func (s *Service) GetMine(ctx context.Context) (MyKrs, error) {
	m, err := s.me(ctx)
	if err != nil {
		return MyKrs{}, err
	}
	sc, err := s.load(ctx, s.pool, m)
	if err != nil {
		return MyKrs{}, err
	}
	out := MyKrs{Window: viewWindow(sc.Window), Eligibility: EligibilityView{OK: len(sc.Reasons) == 0, Reasons: sc.Reasons}}
	k, err := findKrs(ctx, s.pool, m.ID, sc.P.ID, false)
	if err != nil && !db.IsNoRows(err) {
		return out, err
	}
	if err == nil {
		k.MaxSks = sc.MaxSKS
		if k.IpsLalu == nil {
			k.IpsLalu = sc.IpsLalu
		}
		v, err := s.view(ctx, s.pool, k, m)
		if err != nil {
			return out, err
		}
		out.Krs = &v
	}
	return out, nil
}

// Offerings: GET /krs/me/offerings — classes of the student's prodi with eligibility.
func (s *Service) Offerings(ctx context.Context, q string) ([]Offering, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	sc, err := s.load(ctx, s.pool, m)
	if err != nil {
		return nil, err
	}
	ks, err := academic.QueryKelas(ctx, s.pool, `k.periode_id=$2 AND k.status='dibuka'
		AND mk.kurikulum_id IN (SELECT id FROM kurikulum WHERE tenant_id=$1 AND prodi_id=$3)
		AND (mk.nama ILIKE $4 OR mk.kode ILIKE $4) ORDER BY mk.semester, mk.kode, k.nama`, sc.P.ID, m.ProdiID, "%"+strings.TrimSpace(q)+"%")
	if err != nil {
		return nil, err
	}
	var current []academic.Kelas
	total := 0
	if k, err := findKrs(ctx, s.pool, m.ID, sc.P.ID, false); err == nil {
		items, err := krsItems(ctx, s.pool, k.ID)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			current = append(current, it.Kelas)
			total += it.Kelas.Sks
		}
	}
	mkIDs := make([]string, 0, len(ks))
	for _, k := range ks {
		mkIDs = append(mkIDs, k.MkID)
	}
	pre, err := prereqsFor(ctx, s.pool, sc.T.Rules, mkIDs)
	if err != nil {
		return nil, err
	}
	out := make([]Offering, 0, len(ks))
	for _, k := range ks {
		o := Offering{Kelas: k, Reasons: []string{}}
		for _, c := range current {
			if c.ID == k.ID {
				o.InKrs = true
			}
		}
		if !o.InKrs {
			o.Reasons = append(o.Reasons, sc.Reasons...)
			for _, c := range current {
				if c.MkID == k.MkID {
					o.Reasons = append(o.Reasons, ErrDuplicate)
					break
				}
			}
			if k.Terisi >= k.Kuota {
				o.Reasons = append(o.Reasons, ErrQuotaFull)
			}
			if total+k.Sks > sc.MaxSKS {
				o.Reasons = append(o.Reasons, ErrSKSLimit)
			}
			if len(MissingPrereqs(pre[k.MkID], sc.Best, sc.Equiv, sc.T.Rules.LulusMinBobotX100)) > 0 {
				o.Reasons = append(o.Reasons, ErrPrereq)
			}
			if len(Clashes(k, current)) > 0 {
				o.Reasons = append(o.Reasons, ErrClash)
			}
		}
		o.Eligible = !o.InKrs && len(o.Reasons) == 0
		out = append(out, o)
	}
	return out, nil
}

// AddItem: POST /krs/me/items — EC-01 atomic seat reservation in the same transaction
// as the krs_detail insert. All FR-16/17/18 checks run inside the tx under the
// student's KRS row lock.
func (s *Service) AddItem(ctx context.Context, kelasID string) (Krs, error) {
	m, err := s.me(ctx)
	if err != nil {
		return Krs{}, err
	}
	var out Krs
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		sc, err := s.load(ctx, tx, m)
		if err != nil {
			return err
		}
		if len(sc.Reasons) > 0 {
			return reasonError(sc.Reasons[0]).With(map[string]any{"reasons": sc.Reasons})
		}
		if _, err := tx.Exec(ctx, `INSERT INTO krs (tenant_id, mahasiswa_id, periode_id, max_sks, ips_lalu) VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (tenant_id, mahasiswa_id, periode_id) DO NOTHING`, tid, m.ID, sc.P.ID, sc.MaxSKS, sc.IpsLalu); err != nil {
			return err
		}
		k, err := findKrs(ctx, tx, m.ID, sc.P.ID, true)
		if err != nil {
			return err
		}
		if !Modifiable(k.Status, sc.Window.Phase, sc.susulan()) {
			return httpx.Rule(ErrInvalidState, "KRS berstatus "+k.Status+" dan tidak dapat diubah")
		}
		kelas, err := academic.GetKelas(ctx, tx, kelasID)
		if err != nil {
			return httpx.NotFound("Kelas tidak ditemukan")
		}
		if kelas.PeriodeID != sc.P.ID || kelas.Status != "dibuka" {
			return httpx.NotFound("Kelas tidak tersedia pada periode aktif")
		}
		items, err := krsItems(ctx, tx, k.ID)
		if err != nil {
			return err
		}
		total := 0
		current := make([]academic.Kelas, 0, len(items))
		for _, it := range items {
			if it.Kelas.ID == kelas.ID || it.Kelas.MkID == kelas.MkID {
				return httpx.Conflict(ErrDuplicate, "Mata kuliah "+kelas.MkNama+" sudah ada di KRS")
			}
			total += it.Kelas.Sks
			current = append(current, it.Kelas)
		}
		if total+kelas.Sks > sc.MaxSKS {
			return httpx.Rule(ErrSKSLimit, fmt.Sprintf("Melebihi batas %d SKS", sc.MaxSKS)).
				With(map[string]int{"total_sks": total, "tambah_sks": kelas.Sks, "max_sks": sc.MaxSKS})
		}
		pre, err := prereqsFor(ctx, tx, sc.T.Rules, []string{kelas.MkID})
		if err != nil {
			return err
		}
		if miss := MissingPrereqs(pre[kelas.MkID], sc.Best, sc.Equiv, sc.T.Rules.LulusMinBobotX100); len(miss) > 0 {
			return httpx.Rule(ErrPrereq, "Prasyarat belum terpenuhi").With(map[string]any{"missing": miss})
		}
		if cl := Clashes(kelas, current); len(cl) > 0 {
			return httpx.Conflict(ErrClash, "Jadwal bentrok dengan kelas lain di KRS").With(map[string]any{"conflicts": cl})
		}
		// EC-01: atomic conditional increment; 0 rows = full.
		tag, err := tx.Exec(ctx, `UPDATE kelas SET terisi = terisi + 1, updated_at = now()
			WHERE id = $1 AND tenant_id = $2 AND terisi < kuota AND status = 'dibuka'`, kelas.ID, tid)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			quotaFull.Inc()
			return httpx.Rule(ErrQuotaFull, "Kelas sudah penuh")
		}
		ulang := IsUlang(kelas.MkID, sc.Best, sc.Equiv)
		var detailID string
		if err := tx.QueryRow(ctx, `INSERT INTO krs_detail (tenant_id, krs_id, kelas_id, mk_id, sks, is_ulang) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
			tid, k.ID, kelas.ID, kelas.MkID, kelas.Sks, ulang).Scan(&detailID); err != nil {
			if db.IsUniqueViolation(err) {
				return httpx.Conflict(ErrDuplicate, "Mata kuliah sudah ada di KRS")
			}
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE krs SET total_sks=$3, max_sks=$4, ips_lalu=$5, status='draft', updated_at=now() WHERE id=$1 AND tenant_id=$2`,
			k.ID, tid, total+kelas.Sks, sc.MaxSKS, sc.IpsLalu); err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, "krs", k.ID, "add_item", map[string]any{"total_sks": total, "status": k.Status},
			map[string]any{"kelas_id": kelas.ID, "mk_kode": kelas.MkKode, "kelas": kelas.Nama, "total_sks": total + kelas.Sks, "is_ulang": ulang}); err != nil {
			return err
		}
		k, err = getKrs(ctx, tx, k.ID, false)
		if err != nil {
			return err
		}
		out, err = s.view(ctx, tx, k, m)
		return err
	})
	if err == nil {
		itemsAdded.Inc()
	}
	return out, err
}

// RemoveItem: DELETE /krs/me/items/:kelas_id — releases the seat in the same tx.
func (s *Service) RemoveItem(ctx context.Context, kelasID string) (Krs, error) {
	m, err := s.me(ctx)
	if err != nil {
		return Krs{}, err
	}
	var out Krs
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		sc, err := s.load(ctx, tx, m)
		if err != nil {
			return err
		}
		if !sc.Window.Open && !sc.susulan() {
			return reasonError(ErrWindowClosed)
		}
		k, err := findKrs(ctx, tx, m.ID, sc.P.ID, true)
		if err != nil {
			return httpx.NotFound("KRS belum dibuat")
		}
		if !Modifiable(k.Status, sc.Window.Phase, sc.susulan()) {
			return httpx.Rule(ErrInvalidState, "KRS berstatus "+k.Status+" dan tidak dapat diubah")
		}
		var detailID string
		var sks int
		err = tx.QueryRow(ctx, `DELETE FROM krs_detail WHERE tenant_id=$1 AND krs_id=$2 AND kelas_id=$3 AND status='aktif' RETURNING id, sks`,
			tid, k.ID, kelasID).Scan(&detailID, &sks)
		if db.IsNoRows(err) {
			return httpx.NotFound("Kelas tidak ada di KRS")
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE kelas SET terisi = terisi - 1, updated_at=now() WHERE id=$1 AND tenant_id=$2 AND terisi > 0`, kelasID, tid); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE krs SET total_sks = (SELECT coalesce(sum(sks),0) FROM krs_detail WHERE krs_id=$1 AND status='aktif'),
			max_sks=$3, status='draft', updated_at=now() WHERE id=$1 AND tenant_id=$2`, k.ID, tid, sc.MaxSKS); err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, "krs", k.ID, "remove_item", map[string]any{"kelas_id": kelasID, "sks": sks, "status": k.Status}, map[string]any{"status": "draft"}); err != nil {
			return err
		}
		k, err = getKrs(ctx, tx, k.ID, false)
		if err != nil {
			return err
		}
		out, err = s.view(ctx, tx, k, m)
		return err
	})
	return out, err
}

// Submit: POST /krs/me/submit — draft/ditolak → diajukan, re-validating FR-16..18.
func (s *Service) Submit(ctx context.Context) (Krs, error) {
	m, err := s.me(ctx)
	if err != nil {
		return Krs{}, err
	}
	var out Krs
	var waliUser *string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		sc, err := s.load(ctx, tx, m)
		if err != nil {
			return err
		}
		if len(sc.Reasons) > 0 {
			return reasonError(sc.Reasons[0]).With(map[string]any{"reasons": sc.Reasons})
		}
		k, err := findKrs(ctx, tx, m.ID, sc.P.ID, true)
		if err != nil {
			return httpx.Rule(ErrInvalidState, "KRS masih kosong")
		}
		if k.Status != "draft" && k.Status != "ditolak" {
			return httpx.Rule(ErrInvalidState, "KRS berstatus "+k.Status+" tidak dapat diajukan")
		}
		items, err := krsItems(ctx, tx, k.ID)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return httpx.Rule(ErrInvalidState, "KRS masih kosong")
		}
		total := 0
		var ks []academic.Kelas
		var mkIDs []string
		for _, it := range items {
			total += it.Kelas.Sks
			ks = append(ks, it.Kelas)
			mkIDs = append(mkIDs, it.Kelas.MkID)
		}
		if total > sc.MaxSKS {
			return httpx.Rule(ErrSKSLimit, fmt.Sprintf("Total %d SKS melebihi batas %d SKS", total, sc.MaxSKS))
		}
		for i, a := range ks {
			if cl := Clashes(a, ks[i+1:]); len(cl) > 0 {
				return httpx.Conflict(ErrClash, "Jadwal bentrok di dalam KRS").With(map[string]any{"conflicts": append(cl, a)})
			}
		}
		pre, err := prereqsFor(ctx, tx, sc.T.Rules, mkIDs)
		if err != nil {
			return err
		}
		for _, a := range ks {
			if miss := MissingPrereqs(pre[a.MkID], sc.Best, sc.Equiv, sc.T.Rules.LulusMinBobotX100); len(miss) > 0 {
				return httpx.Rule(ErrPrereq, "Prasyarat "+a.MkNama+" belum terpenuhi").With(map[string]any{"missing": miss})
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE krs SET status='diajukan', total_sks=$3, max_sks=$4, submitted_at=now(), decided_at=NULL, decided_by=NULL, updated_at=now()
			WHERE id=$1 AND tenant_id=$2`, k.ID, tid, total, sc.MaxSKS); err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, "krs", k.ID, "submit", map[string]string{"status": k.Status}, map[string]any{"status": "diajukan", "total_sks": total}); err != nil {
			return err
		}
		if m.DosenWaliID != nil {
			_ = tx.QueryRow(ctx, `SELECT user_id FROM dosen WHERE tenant_id=$1 AND id=$2`, tid, *m.DosenWaliID).Scan(&waliUser)
			if waliUser != nil {
				if err := notification.Send(ctx, tx, *waliUser, "KRS menunggu persetujuan",
					fmt.Sprintf("%s (%s) mengajukan KRS %d SKS.", m.Nama, m.NIM, total), notification.Link("/krs/"+k.ID)); err != nil {
					return err
				}
			}
		}
		k, err = getKrs(ctx, tx, k.ID, false)
		if err != nil {
			return err
		}
		out, err = s.view(ctx, tx, k, m)
		return err
	})
	return out, err
}

// ---- dosen wali / staff ----

// accessible returns the KRS's student if the caller may see it; otherwise 404 (never 403).
func (s *Service) accessible(ctx context.Context, q db.Querier, k krsRow) (academic.Mahasiswa, error) {
	m, err := academic.GetMahasiswa(ctx, q, k.MahasiswaID)
	if err != nil {
		return m, err
	}
	p, _ := appctx.PrincipalFrom(ctx)
	switch p.ActiveRole {
	case "operator", "admin_kampus":
		return m, nil
	case "kaprodi":
		if sc := p.ScopeProdi("kaprodi"); sc == nil || *sc == m.ProdiID {
			return m, nil
		}
	case "dosen_wali":
		did, err := academic.DosenIDByUser(ctx, q, p.UserID)
		if err != nil {
			return m, err
		}
		if did != "" && m.DosenWaliID != nil && *m.DosenWaliID == did {
			return m, nil
		}
	case "mahasiswa":
		if m.UserID != nil && *m.UserID == p.UserID {
			return m, nil
		}
	}
	return m, httpx.NotFound("KRS tidak ditemukan")
}

func (s *Service) Get(ctx context.Context, id string) (Krs, error) {
	k, err := getKrs(ctx, s.pool, id, false)
	if err != nil {
		return Krs{}, httpx.NotFound("KRS tidak ditemukan")
	}
	m, err := s.accessible(ctx, s.pool, k)
	if err != nil {
		return Krs{}, err
	}
	return s.view(ctx, s.pool, k, m)
}

// Advisees: GET /krs/advisees?status=&periode_id=
func (s *Service) Advisees(ctx context.Context, status, periodeID string) ([]Krs, error) {
	p, _ := appctx.PrincipalFrom(ctx)
	if periodeID == "" {
		ap, err := academic.ActivePeriode(ctx, s.pool)
		if err != nil {
			return nil, err
		}
		periodeID = ap.ID
	}
	dosenID := ""
	if p.ActiveRole == "dosen_wali" {
		var err error
		if dosenID, err = academic.DosenIDByUser(ctx, s.pool, p.UserID); err != nil {
			return nil, err
		}
		if dosenID == "" {
			return []Krs{}, nil
		}
	}
	rows, err := s.pool.Query(ctx, `SELECT krs.id, krs.mahasiswa_id, krs.periode_id, krs.status, krs.total_sks, krs.max_sks, krs.ips_lalu, krs.catatan,
		iso(krs.submitted_at), iso(krs.decided_at) FROM krs JOIN mahasiswa m ON m.id=krs.mahasiswa_id
		WHERE krs.tenant_id=$1 AND krs.periode_id::text=$2 AND ($3='' OR krs.status=$3) AND ($4='' OR m.dosen_wali_id::text=$4)
		ORDER BY krs.submitted_at NULLS LAST, m.nim LIMIT 500`, appctx.TenantID(ctx), periodeID, status, dosenID)
	if err != nil {
		return nil, err
	}
	var ks []krsRow
	for rows.Next() {
		k, err := scanKrs(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		ks = append(ks, k)
	}
	rows.Close()
	out := []Krs{}
	for _, k := range ks {
		m, err := academic.GetMahasiswa(ctx, s.pool, k.MahasiswaID)
		if err != nil {
			return nil, err
		}
		v, err := s.view(ctx, s.pool, k, m)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Status == "diajukan" && out[j].Status != "diajukan" })
	return out, nil
}

// Decide approves or rejects a submitted KRS (FR-19). EC-14: allowed until the window
// end + grace days. Rejection requires a note.
func (s *Service) Decide(ctx context.Context, id string, approve bool, catatan string) (Krs, error) {
	if !approve && strings.TrimSpace(catatan) == "" {
		return Krs{}, httpx.ValidationField("catatan", "wajib diisi saat menolak KRS")
	}
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return Krs{}, err
	}
	var out Krs
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		k, err := getKrs(ctx, tx, id, true)
		if err != nil {
			return httpx.NotFound("KRS tidak ditemukan")
		}
		m, err := s.accessible(ctx, tx, k)
		if err != nil {
			return err
		}
		if k.Status != "diajukan" {
			return httpx.Rule(ErrInvalidState, "Hanya KRS berstatus diajukan yang dapat diproses")
		}
		p, err := academic.GetPeriode(ctx, tx, k.PeriodeID)
		if err != nil {
			return err
		}
		if !CanDecide(p.Tahapan, s.Now(), t.Loc, t.Rules.KRSGraceDays) {
			return httpx.Rule(ErrWindowClosed, fmt.Sprintf("Batas persetujuan KRS (H+%d setelah jendela tutup) sudah lewat", t.Rules.KRSGraceDays))
		}
		status := "ditolak"
		if approve {
			status = "disetujui"
		}
		var note *string
		if strings.TrimSpace(catatan) != "" {
			note = &catatan
		}
		if _, err := tx.Exec(ctx, `UPDATE krs SET status=$3, catatan=$4, decided_at=now(), decided_by=$5, updated_at=now() WHERE id=$1 AND tenant_id=$2`,
			id, tid, status, note, appctx.UserID(ctx)); err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, "krs", id, map[bool]string{true: "approve", false: "reject"}[approve],
			map[string]string{"status": k.Status}, map[string]any{"status": status, "catatan": catatan}); err != nil {
			return err
		}
		if m.UserID != nil {
			judul := "KRS disetujui"
			if !approve {
				judul = "KRS ditolak"
			}
			isi := judul + " oleh dosen wali."
			if note != nil {
				isi += " Catatan: " + catatan
			}
			if err := notification.Send(ctx, tx, *m.UserID, judul, isi, notification.Link("/krs")); err != nil {
				return err
			}
		}
		k, err = getKrs(ctx, tx, id, false)
		if err != nil {
			return err
		}
		out, err = s.view(ctx, tx, k, m)
		return err
	})
	return out, err
}
