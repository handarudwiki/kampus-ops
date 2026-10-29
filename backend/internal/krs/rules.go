// Package krs implements study-plan (KRS) selection, submission and advisor approval
// (FR-16..20, EC-01, EC-02, EC-05, EC-07, EC-14, EC-15). Pure rules live in this file.
package krs

import (
	"time"

	"github.com/kampusops/backend/internal/academic"
	"github.com/kampusops/backend/internal/domain"
)

// Error codes (contract).
const (
	ErrWindowClosed = "KRS_WINDOW_CLOSED"
	ErrNotActive    = "KRS_STUDENT_NOT_ACTIVE"
	ErrUnpaid       = "KRS_UNPAID"
	ErrQuotaFull    = "KRS_QUOTA_FULL"
	ErrSKSLimit     = "KRS_SKS_LIMIT"
	ErrPrereq       = "KRS_PREREQ_NOT_MET"
	ErrClash        = "KRS_SCHEDULE_CLASH"
	ErrInvalidState = "KRS_INVALID_STATE"
	ErrDuplicate    = "KRS_DUPLICATE_COURSE"
)

// MaxSKS implements FR-17 and EC-07. Pendek periods are capped at SKSPendekMax and do
// not use the IPS rule; without a previous IPS (new student) SKSDefaultBaru applies.
func MaxSKS(r domain.Rules, periodeTipe string, prevIPSx100 *int) int {
	if periodeTipe == "Pendek" {
		return r.SKSPendekMax
	}
	if prevIPSx100 == nil {
		return r.SKSDefaultBaru
	}
	for _, rule := range r.SKSRules { // sorted by MinIPSx100 desc
		if *prevIPSx100 >= rule.MinIPSx100 {
			return rule.MaxSKS
		}
	}
	return r.SKSRules[len(r.SKSRules)-1].MaxSKS
}

// Window is the KRS window state evaluated in the tenant's timezone (EC-17).
type Window struct {
	Open  bool      `json:"open"`
	Phase string    `json:"phase"` // krs | revisi | closed
	Start time.Time `json:"start"`
	End   time.Time `json:"end"` // exclusive end instant (00:00 next day, tenant tz)
}

// EvaluateWindow decides whether KRS (or revision, FR-20) is open at now.
func EvaluateWindow(t domain.Tahapan, now time.Time, loc *time.Location) Window {
	ks, _ := domain.DayStart(t.KrsMulai, loc)
	ke, _ := domain.DayEnd(t.KrsSelesai, loc)
	w := Window{Phase: "closed", Start: ks, End: ke}
	if domain.InRange(now, t.KrsMulai, t.KrsSelesai, loc) {
		return Window{Open: true, Phase: "krs", Start: ks, End: ke}
	}
	if t.RevisiMulai != nil && t.RevisiSelesai != nil {
		rs, ok1 := domain.DayStart(*t.RevisiMulai, loc)
		re, ok2 := domain.DayEnd(*t.RevisiSelesai, loc)
		if ok1 && ok2 {
			if domain.InRange(now, *t.RevisiMulai, *t.RevisiSelesai, loc) {
				return Window{Open: true, Phase: "revisi", Start: rs, End: re}
			}
			if !now.Before(ke) { // after KRS: show the revision window (upcoming or past)
				w.Start, w.End = rs, re
			}
		}
	}
	return w
}

// ApprovalDeadline is the last instant (exclusive) a dosen wali may still decide a KRS
// that was submitted in time: the end of the latest KRS/revision window + grace days (EC-14).
func ApprovalDeadline(t domain.Tahapan, loc *time.Location, graceDays int) time.Time {
	end, _ := domain.DayEnd(t.KrsSelesai, loc)
	if t.RevisiSelesai != nil {
		if re, ok := domain.DayEnd(*t.RevisiSelesai, loc); ok && re.After(end) {
			end = re
		}
	}
	return end.AddDate(0, 0, graceDays)
}

// CanDecide reports whether approval/rejection is still allowed at now (EC-14).
func CanDecide(t domain.Tahapan, now time.Time, loc *time.Location, graceDays int) bool {
	return now.Before(ApprovalDeadline(t, loc, graceDays))
}

// Bill is the billing info relevant for KRS eligibility.
type Bill struct {
	Jenis       string
	Status      string // belum_bayar | sebagian | lunas
	Terbayar    int64
	CicilanDue  int64 // sum of installments due up to today (0 when no plan)
	HasCicilan  bool
}

// EligibilityInput collects everything FR-16 / FR-27 / EC-08 / EC-15 need.
type EligibilityInput struct {
	Window              Window
	StudentStatus       string   // from riwayat_status
	Bills               []Bill   // mandatory bills of the period (ukt, praktikum)
	ApprovedDispensasi  []string // approved dispensasi jenis for the period
}

func has(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// Eligibility returns the list of blocking reasons (empty = eligible).
//   - the window must be open, unless an approved krs_susulan dispensation exists (EC-15:
//     a late payment never reopens KRS by itself);
//   - the student must be aktif per riwayat_status (cuti blocks KRS, EC-08);
//   - mandatory bills must be lunas, or covered by an approved cicilan/penundaan
//     dispensation, or by an installment plan that is up to date (FR-27).
func Eligibility(in EligibilityInput) []string {
	reasons := []string{}
	if !in.Window.Open && !has(in.ApprovedDispensasi, "krs_susulan") {
		reasons = append(reasons, ErrWindowClosed)
	}
	if in.StudentStatus != domain.StatusAktif {
		reasons = append(reasons, ErrNotActive)
	}
	if !billsOK(in.Bills, in.ApprovedDispensasi) {
		reasons = append(reasons, ErrUnpaid)
	}
	return reasons
}

func billsOK(bills []Bill, disp []string) bool {
	if has(disp, "cicilan") || has(disp, "penundaan") {
		return true
	}
	for _, b := range bills {
		if b.Status == "lunas" {
			continue
		}
		if b.HasCicilan && b.Terbayar >= b.CicilanDue {
			continue
		}
		return false
	}
	return true
}

// Prereq is one prerequisite of a course (FR-12). MinBobot nil = "sudah lulus".
type Prereq struct {
	MkID     string  `json:"mk_id"`
	Kode     string  `json:"kode"`
	Nama     string  `json:"nama"`
	NilaiMin *string `json:"nilai_min"`
	MinBobot *int    `json:"-"`
}

// MissingPrereqs implements FR-12/FR-18 with EC-05: a prerequisite is satisfied by the
// course itself or any course mapped to it in ekuivalensi (either direction), with a
// best grade ≥ the minimum (or ≥ the passing grade when no minimum is set).
func MissingPrereqs(prereqs []Prereq, best map[string]int, equiv map[string][]string, lulusMin int) []Prereq {
	missing := []Prereq{}
	for _, p := range prereqs {
		min := lulusMin
		if p.MinBobot != nil {
			min = *p.MinBobot
		}
		ok := false
		for _, id := range append([]string{p.MkID}, equiv[p.MkID]...) {
			if b, found := best[id]; found && b >= min {
				ok = true
				break
			}
		}
		if !ok {
			missing = append(missing, p)
		}
	}
	return missing
}

// BuildEquiv turns ekuivalensi pairs into a symmetric adjacency map.
func BuildEquiv(pairs [][2]string) map[string][]string {
	m := map[string][]string{}
	for _, p := range pairs {
		m[p[0]] = append(m[p[0]], p[1])
		m[p[1]] = append(m[p[1]], p[0])
	}
	return m
}

// BestBobot maps course id → best final bobot×100.
func BestBobot(entries []domain.GradeEntry) map[string]int {
	m := map[string]int{}
	for _, e := range entries {
		if cur, ok := m[e.MkID]; !ok || e.BobotX100 > cur {
			m[e.MkID] = e.BobotX100
		}
	}
	return m
}

// IsUlang reports whether a course (or an equivalent) was already taken (EC-06).
func IsUlang(mkID string, best map[string]int, equiv map[string][]string) bool {
	if _, ok := best[mkID]; ok {
		return true
	}
	for _, e := range equiv[mkID] {
		if _, ok := best[e]; ok {
			return true
		}
	}
	return false
}

// Clashes returns the classes in current whose slot overlaps candidate (FR-18).
func Clashes(candidate academic.Kelas, current []academic.Kelas) []academic.Kelas {
	out := []academic.Kelas{}
	for _, k := range current {
		if k.ID != candidate.ID && academic.Overlaps(candidate.Slot, k.Slot) {
			out = append(out, k)
		}
	}
	return out
}

// Modifiable reports whether the student may change KRS items in this state: draft or
// ditolak normally; an already submitted/approved KRS during the revision phase (FR-20)
// or with a krs_susulan dispensation (EC-13/EC-15) — the change resets it to draft.
func Modifiable(status, phase string, susulan bool) bool {
	switch status {
	case "draft", "ditolak":
		return true
	case "diajukan", "disetujui":
		return phase == "revisi" || susulan
	}
	return false
}
