package krs

import (
	"testing"
	"time"

	"github.com/kampusops/backend/internal/academic"
	"github.com/kampusops/backend/internal/domain"
)

func ip(v int) *int { return &v }

func TestMaxSKSByPreviousIPS(t *testing.T) {
	r := domain.DefaultRules()
	cases := []struct {
		ips  *int
		want int
	}{
		{ip(400), 24}, {ip(300), 24}, {ip(299), 21}, {ip(250), 21}, {ip(249), 18}, {ip(200), 18}, {ip(199), 15}, {ip(0), 15}, {nil, 20},
	}
	for _, c := range cases {
		if got := MaxSKS(r, "Ganjil", c.ips); got != c.want {
			t.Errorf("MaxSKS(%v) = %d want %d", c.ips, got, c.want)
		}
	}
	// IPS 2.995 displays as "3.00" and gets 24 SKS (rule uses the displayed value)
	num, den := int64(3594), int64(12) // 3594/(12*100) = 2.995
	if got := MaxSKS(r, "Genap", ip(domain.IPx100(num, den))); got != 24 {
		t.Fatalf("2.995 → %d", got)
	}
	// tenant-configurable
	r2 := domain.ParseRules([]byte(`{"sks_rules":[{"min_ips_x100":350,"max_sks":22},{"min_ips_x100":0,"max_sks":12}]}`))
	if MaxSKS(r2, "Ganjil", ip(349)) != 12 || MaxSKS(r2, "Ganjil", ip(360)) != 22 {
		t.Fatal("custom tenant rules")
	}
}

func TestPendekIgnoresIPSRule(t *testing.T) { // EC-07
	r := domain.DefaultRules()
	for _, v := range []*int{ip(400), ip(100), nil} {
		if MaxSKS(r, "Pendek", v) != 9 {
			t.Fatal("Pendek must be capped at 9 regardless of IPS")
		}
	}
}

func tahapan() domain.Tahapan {
	rm, rs := "2026-11-10", "2026-11-14"
	return domain.Tahapan{KrsMulai: "2026-11-01", KrsSelesai: "2026-11-05", RevisiMulai: &rm, RevisiSelesai: &rs}
}

func TestWindowEvaluatedInTenantTimezone(t *testing.T) { // EC-17
	jkt, mks := domain.LoadLocation("Asia/Jakarta"), domain.LoadLocation("Asia/Makassar")
	th := tahapan()
	// 2026-11-05 16:30 UTC: 23:30 WIB (open) but 00:30 WITA Nov 6 (closed)
	now := time.Date(2026, 11, 5, 16, 30, 0, 0, time.UTC)
	if w := EvaluateWindow(th, now, jkt); !w.Open || w.Phase != "krs" {
		t.Fatalf("Jakarta: %+v", w)
	}
	if w := EvaluateWindow(th, now, mks); w.Open || w.Phase != "closed" {
		t.Fatalf("Makassar: %+v", w)
	}
	// window opens at 00:00 local: 2026-10-31 17:00 UTC = Nov 1 00:00 WIB
	if !EvaluateWindow(th, time.Date(2026, 10, 31, 17, 0, 0, 0, time.UTC), jkt).Open {
		t.Fatal("should open at local midnight")
	}
	if EvaluateWindow(th, time.Date(2026, 10, 31, 16, 59, 59, 0, time.UTC), jkt).Open {
		t.Fatal("should be closed one second before local midnight")
	}
	// revision phase (FR-20)
	if w := EvaluateWindow(th, time.Date(2026, 11, 12, 3, 0, 0, 0, time.UTC), jkt); !w.Open || w.Phase != "revisi" {
		t.Fatalf("revisi: %+v", w)
	}
	// between KRS and revisi: closed, shows the upcoming revision window
	w := EvaluateWindow(th, time.Date(2026, 11, 7, 3, 0, 0, 0, time.UTC), jkt)
	if w.Open || w.Start.In(jkt).Format("2006-01-02") != "2026-11-10" {
		t.Fatalf("gap: %+v", w)
	}
}

func TestGracePeriodForAdvisorApproval(t *testing.T) { // EC-14
	jkt := domain.LoadLocation("Asia/Jakarta")
	th := tahapan() // last window ends 2026-11-14 → deadline end of 2026-11-17 WIB
	at := func(s string) time.Time { x, _ := time.ParseInLocation("2006-01-02 15:04", s, jkt); return x }
	if !CanDecide(th, at("2026-11-17 23:59"), jkt, 3) {
		t.Fatal("H+3 must still be allowed")
	}
	if CanDecide(th, at("2026-11-18 00:00"), jkt, 3) {
		t.Fatal("H+4 must be rejected")
	}
	noRev := domain.Tahapan{KrsMulai: "2026-11-01", KrsSelesai: "2026-11-05"}
	if !CanDecide(noRev, at("2026-11-08 12:00"), jkt, 3) || CanDecide(noRev, at("2026-11-09 00:01"), jkt, 3) {
		t.Fatal("grace without revision window")
	}
}

func TestEligibility(t *testing.T) {
	open := Window{Open: true, Phase: "krs"}
	closed := Window{Phase: "closed"}
	lunas := []Bill{{Jenis: "ukt", Status: "lunas"}}
	unpaid := []Bill{{Jenis: "ukt", Status: "belum_bayar"}}
	cases := []struct {
		name string
		in   EligibilityInput
		want []string
	}{
		{"ok", EligibilityInput{Window: open, StudentStatus: "aktif", Bills: lunas}, nil},
		{"no bills generated", EligibilityInput{Window: open, StudentStatus: "aktif"}, nil},
		{"closed", EligibilityInput{Window: closed, StudentStatus: "aktif", Bills: lunas}, []string{ErrWindowClosed}},
		{"cuti blocks KRS (EC-08)", EligibilityInput{Window: open, StudentStatus: "cuti", Bills: lunas}, []string{ErrNotActive}},
		{"unpaid", EligibilityInput{Window: open, StudentStatus: "aktif", Bills: unpaid}, []string{ErrUnpaid}},
		{"partial", EligibilityInput{Window: open, StudentStatus: "aktif", Bills: []Bill{{Status: "sebagian", Terbayar: 10}}}, []string{ErrUnpaid}},
		{"cicilan dispensation (FR-27)", EligibilityInput{Window: open, StudentStatus: "aktif", Bills: unpaid, ApprovedDispensasi: []string{"cicilan"}}, nil},
		{"penundaan dispensation", EligibilityInput{Window: open, StudentStatus: "aktif", Bills: unpaid, ApprovedDispensasi: []string{"penundaan"}}, nil},
		{"installment plan up to date", EligibilityInput{Window: open, StudentStatus: "aktif", Bills: []Bill{{Status: "sebagian", Terbayar: 500, CicilanDue: 500, HasCicilan: true}}}, nil},
		{"installment plan overdue", EligibilityInput{Window: open, StudentStatus: "aktif", Bills: []Bill{{Status: "sebagian", Terbayar: 500, CicilanDue: 1000, HasCicilan: true}}}, []string{ErrUnpaid}},
		{"paid after close does not reopen (EC-15)", EligibilityInput{Window: closed, StudentStatus: "aktif", Bills: lunas}, []string{ErrWindowClosed}},
		{"krs_susulan reopens for the student (EC-15)", EligibilityInput{Window: closed, StudentStatus: "aktif", Bills: lunas, ApprovedDispensasi: []string{"krs_susulan"}}, nil},
		{"all reasons", EligibilityInput{Window: closed, StudentStatus: "non_aktif", Bills: unpaid}, []string{ErrWindowClosed, ErrNotActive, ErrUnpaid}},
	}
	for _, c := range cases {
		got := Eligibility(c.in)
		if len(got) != len(c.want) {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: got %v want %v", c.name, got, c.want)
			}
		}
	}
}

func TestPrerequisitesThroughEquivalence(t *testing.T) { // FR-12, EC-05
	r := domain.DefaultRules()
	c, _ := r.BobotOf("C")
	pre := []Prereq{{MkID: "alg-new", Kode: "IF11", MinBobot: &c}, {MkID: "mat"}}
	equiv := BuildEquiv([][2]string{{"alg-old", "alg-new"}})
	// old-curriculum student passed the replaced course with B
	best := map[string]int{"alg-old": 300, "mat": 100}
	if miss := MissingPrereqs(pre, best, equiv, r.LulusMinBobotX100); len(miss) != 0 {
		t.Fatalf("equivalence should satisfy: %v", miss)
	}
	// D in the old course is below the C minimum
	best = map[string]int{"alg-old": 100, "mat": 100}
	if miss := MissingPrereqs(pre, best, equiv, r.LulusMinBobotX100); len(miss) != 1 || miss[0].MkID != "alg-new" {
		t.Fatalf("min grade: %v", miss)
	}
	// E is not "lulus"
	best = map[string]int{"alg-new": 400, "mat": 0}
	if miss := MissingPrereqs(pre, best, equiv, r.LulusMinBobotX100); len(miss) != 1 || miss[0].MkID != "mat" {
		t.Fatalf("lulus: %v", miss)
	}
	// without the mapping the old course does not count
	if miss := MissingPrereqs(pre, map[string]int{"alg-old": 400, "mat": 300}, nil, 100); len(miss) != 1 {
		t.Fatal("code-only check must fail without ekuivalensi")
	}
	if !IsUlang("alg-new", map[string]int{"alg-old": 100}, equiv) || IsUlang("x", map[string]int{}, equiv) {
		t.Fatal("IsUlang")
	}
	if got := BestBobot([]domain.GradeEntry{{MkID: "a", BobotX100: 100}, {MkID: "a", BobotX100: 300}, {MkID: "a", BobotX100: 200}}); got["a"] != 300 {
		t.Fatal("BestBobot")
	}
}

func TestScheduleClash(t *testing.T) { // FR-18
	k := func(id string, hari int, a, b string) academic.Kelas {
		return academic.Kelas{ID: id, Slot: academic.Slot{Hari: hari, JamMulai: a, JamSelesai: b}}
	}
	cur := []academic.Kelas{k("1", 1, "07:30", "09:10"), k("2", 2, "10:00", "12:30")}
	if len(Clashes(k("x", 1, "09:00", "10:40"), cur)) != 1 {
		t.Fatal("overlap on Monday")
	}
	if len(Clashes(k("x", 1, "09:10", "10:50"), cur)) != 0 {
		t.Fatal("back-to-back is not a clash")
	}
	if len(Clashes(k("x", 3, "07:30", "09:10"), cur)) != 0 {
		t.Fatal("different day")
	}
	if len(Clashes(k("x", 2, "08:00", "13:00"), cur)) != 1 {
		t.Fatal("containing interval")
	}
}

func TestModifiable(t *testing.T) {
	if !Modifiable("draft", "krs", false) || !Modifiable("ditolak", "krs", false) {
		t.Fatal("draft/ditolak editable")
	}
	if Modifiable("diajukan", "krs", false) || Modifiable("disetujui", "krs", false) {
		t.Fatal("submitted KRS locked during KRS phase")
	}
	if !Modifiable("disetujui", "revisi", false) || !Modifiable("disetujui", "closed", true) {
		t.Fatal("revision phase / susulan allow changes")
	}
}
