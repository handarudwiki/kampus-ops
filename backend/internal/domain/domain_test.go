package domain

import (
	"testing"
	"time"
)

func TestLetterConversion(t *testing.T) {
	r := DefaultRules()
	cases := []struct {
		angka int
		huruf string
		bobot int
	}{
		{10000, "A", 400}, {8500, "A", 400}, {8499, "A-", 370}, {8000, "A-", 370}, {7999, "B+", 330}, {7500, "B+", 330},
		{7000, "B", 300}, {6500, "B-", 270}, {6000, "C+", 230}, {5500, "C", 200}, {5499, "D", 100}, {4000, "D", 100}, {3999, "E", 0}, {0, "E", 0},
	}
	for _, c := range cases {
		h, b := r.Letter(c.angka)
		if h != c.huruf || b != c.bobot {
			t.Errorf("Letter(%d) = %s/%d, want %s/%d", c.angka, h, b, c.huruf, c.bobot)
		}
	}
	if b, ok := r.BobotOf("B+"); !ok || b != 330 {
		t.Fatalf("BobotOf(B+) = %d %v", b, ok)
	}
	if _, ok := r.BobotOf("Z"); ok {
		t.Fatal("unknown letter must not resolve")
	}
}

func TestComputeAngkaWeightedIntegerMath(t *testing.T) {
	k := DefaultRules().KomponenDefault // 20/30/50
	v := func(x int) *int { return &x }
	angka, complete := ComputeAngka(k, map[string]*int{"tugas": v(8000), "uts": v(7550), "uas": v(9000)})
	// 0.2*80 + 0.3*75.5 + 0.5*90 = 83.65
	if !complete || angka != 8365 {
		t.Fatalf("got %d complete=%v", angka, complete)
	}
	// half-up rounding of the third decimal: 0.2*80.01+0.3*0+0.5*0 = 16.002 → 16.00
	angka, _ = ComputeAngka(k, map[string]*int{"tugas": v(8001), "uts": v(0), "uas": v(0)})
	if angka != 1600 {
		t.Fatalf("rounding: got %d", angka)
	}
	if _, complete := ComputeAngka(k, map[string]*int{"tugas": v(8000), "uts": nil}); complete {
		t.Fatal("missing components must be incomplete")
	}
	if FormatX100(8365) != "83.65" || FormatX100(5) != "0.05" {
		t.Fatal("FormatX100")
	}
}

func TestIPSRoundingOnlyAtDisplay(t *testing.T) {
	// 3 SKS A (400) + 3 SKS B+ (330) + 2 SKS A- (370) + 2 SKS B (300) + 2 SKS C (200)
	e := []GradeEntry{{MkID: "a", Sks: 3, BobotX100: 400}, {MkID: "b", Sks: 3, BobotX100: 330}, {MkID: "c", Sks: 2, BobotX100: 370},
		{MkID: "d", Sks: 2, BobotX100: 300}, {MkID: "e", Sks: 2, BobotX100: 200}}
	// (1200+990+740+600+400)/12 = 3930/12 = 327.5 → 3.275 → "3.28" (half away from zero)
	if got := IPS(e); got != "3.28" {
		t.Fatalf("IPS = %s", got)
	}
	num, den := Sum(e)
	if IPx100(num, den) != 328 {
		t.Fatalf("IPx100 = %d", IPx100(num, den))
	}
	// classic float trap: 1/3 style values stay exact until display
	e2 := []GradeEntry{{Sks: 3, BobotX100: 400}, {Sks: 3, BobotX100: 300}, {Sks: 3, BobotX100: 300}}
	if got := IPS(e2); got != "3.33" {
		t.Fatalf("IPS thirds = %s", got)
	}
	if IPS(nil) != "0.00" {
		t.Fatal("empty IPS must be 0.00")
	}
}

func TestIPKUsesBestGradeForRepeats(t *testing.T) {
	r := DefaultRules()
	e := []GradeEntry{
		{MkID: "alg", Sks: 3, BobotX100: 100, Huruf: "D", PeriodeKode: "20241"},
		{MkID: "alg", Sks: 3, BobotX100: 330, Huruf: "B+", PeriodeKode: "20251"}, // repeat, better
		{MkID: "mat", Sks: 3, BobotX100: 300, Huruf: "B", PeriodeKode: "20241"},
		{MkID: "eng", Sks: 2, BobotX100: 0, Huruf: "E", PeriodeKode: "20241"},
	}
	best := BestGrades(e, nil)
	if len(best) != 3 {
		t.Fatalf("best = %d rows", len(best))
	}
	num, den, lulus := r.Cumulative(e, nil)
	// (3*330 + 3*300 + 2*0) / 8 = 1890/8 = 236.25 → "2.36"
	if FormatIP(num, den) != "2.36" {
		t.Fatalf("IPK = %s", FormatIP(num, den))
	}
	if lulus != 6 {
		t.Fatalf("sks lulus = %d (E must not count)", lulus)
	}
	// worse repeat must not lower the transcript grade
	e = append(e, GradeEntry{MkID: "alg", Sks: 3, BobotX100: 200, PeriodeKode: "20252"})
	if n2, d2, _ := r.Cumulative(e, nil); FormatIP(n2, d2) != "2.36" {
		t.Fatal("worse repeat changed IPK")
	}
	// KHS (semester view) shows that semester's own grade: IPS of 20241 includes the D
	sem := []GradeEntry{}
	for _, x := range e {
		if x.PeriodeKode == "20241" {
			sem = append(sem, x)
		}
	}
	if IPS(sem) != "1.50" { // (300+900+0)/8
		t.Fatalf("IPS 20241 = %s", IPS(sem))
	}
	// equivalent courses can be grouped through canonical()
	canon := func(id string) string {
		if id == "alg-old" {
			return "alg"
		}
		return id
	}
	e3 := []GradeEntry{{MkID: "alg-old", Sks: 3, BobotX100: 400}, {MkID: "alg", Sks: 3, BobotX100: 200}}
	if len(BestGrades(e3, canon)) != 1 {
		t.Fatal("canonical grouping failed")
	}
}

func TestParseRulesOverridesAndDefaults(t *testing.T) {
	r := ParseRules([]byte(`{"cuti_max":1,"sks_rules":[{"min_ips_x100":0,"max_sks":12},{"min_ips_x100":350,"max_sks":24}]}`))
	if r.CutiMax != 1 || r.SKSPendekMax != 9 || r.KRSGraceDays != 3 {
		t.Fatalf("%+v", r)
	}
	if r.SKSRules[0].MinIPSx100 != 350 {
		t.Fatal("rules must be sorted desc")
	}
	if ParseRules([]byte(`not json`)).CutiMax != 2 {
		t.Fatal("bad json must fall back to defaults")
	}
}

func TestStudentStatusTransitions(t *testing.T) {
	ok := [][2]string{{"aktif", "cuti"}, {"cuti", "aktif"}, {"aktif", "lulus"}, {"non_aktif", "aktif"}, {"aktif", "do"}}
	bad := [][2]string{{"lulus", "aktif"}, {"do", "aktif"}, {"keluar", "aktif"}, {"cuti", "lulus"}, {"aktif", "aktif"}}
	for _, c := range ok {
		if !CanTransition(c[0], c[1]) {
			t.Errorf("%s→%s should be allowed", c[0], c[1])
		}
	}
	for _, c := range bad {
		if CanTransition(c[0], c[1]) {
			t.Errorf("%s→%s should be rejected", c[0], c[1])
		}
	}
	if !CutiAllowed(1, 2) || CutiAllowed(2, 2) || CutiAllowed(1, 1) {
		t.Fatal("CutiAllowed (EC-08)")
	}
}

func TestMasaStudi(t *testing.T) {
	if SemesterKe(2024, 2026, "Ganjil") != 5 || SemesterKe(2024, 2025, "Genap") != 4 || SemesterKe(2024, 2025, "Pendek") != 4 {
		t.Fatal("SemesterKe (Pendek does not add a semester, EC-07)")
	}
	// S1 max 14: angkatan 2020 in 20261 = semester 13, one cuti → used 12, sisa 2 → warning
	used, sisa := MasaStudi(SemesterKe(2020, 2026, "Ganjil"), 1, 14)
	if used != 12 || sisa != 2 || MasaStudiFlag(sisa, 2) != "warning" {
		t.Fatalf("used=%d sisa=%d", used, sisa)
	}
	_, sisa = MasaStudi(15, 0, 14)
	if MasaStudiFlag(sisa, 2) != "do_candidate" {
		t.Fatal("exceeded must be a DO candidate")
	}
	if MasaStudiFlag(5, 2) != "" {
		t.Fatal("no warning expected")
	}
}

func TestTimeHelpersTenantZone(t *testing.T) {
	jkt, mks := LoadLocation("Asia/Jakarta"), LoadLocation("Asia/Makassar")
	// 2026-11-05 16:30 UTC = 23:30 WIB (still Nov 5) = 00:30 WITA (Nov 6)
	now := time.Date(2026, 11, 5, 16, 30, 0, 0, time.UTC)
	if LocalDate(now, jkt) != "2026-11-05" || LocalDate(now, mks) != "2026-11-06" {
		t.Fatal("LocalDate")
	}
	if !InRange(now, "2026-11-01", "2026-11-05", jkt) || InRange(now, "2026-11-01", "2026-11-05", mks) {
		t.Fatal("InRange must be evaluated in tenant tz (EC-17)")
	}
	if LoadLocation("Nope/Zone") != time.UTC {
		t.Fatal("fallback")
	}
}
