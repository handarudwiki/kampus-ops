package lecture

import (
	"testing"
	"time"

	"github.com/kampusops/backend/internal/academic"
	"github.com/kampusops/backend/internal/domain"
)

func f(v float64) *float64 { return &v }

func TestComputeGrade(t *testing.T) { // FR-22
	r := domain.DefaultRules()
	k := r.KomponenDefault
	sc, err := ScoresToX100(k, map[string]*float64{"tugas": f(80), "uts": f(75.5), "uas": f(90)})
	if err != nil {
		t.Fatal(err)
	}
	g := ComputeGrade(r, k, sc)
	if !g.Complete || *g.AngkaX100 != 8365 || *g.Huruf != "A-" || *g.BobotX100 != 370 {
		t.Fatalf("%+v", g)
	}
	if _, err := ScoresToX100(k, map[string]*float64{"uts": f(101)}); err == nil {
		t.Fatal("> 100 must fail")
	}
	if _, err := ScoresToX100(k, map[string]*float64{"kuis": f(50)}); err == nil {
		t.Fatal("unknown component must fail")
	}
	if g := ComputeGrade(r, k, map[string]*int{}); g.Complete || g.Huruf != nil {
		t.Fatal("incomplete grade has no letter")
	}
}

func TestGradeLockDeadlineInTenantTZ(t *testing.T) { // FR-23, EC-17
	jkt := domain.LoadLocation("Asia/Jakarta")
	end := time.Date(2026, 12, 20, 16, 59, 0, 0, time.UTC) // 23:59 WIB Dec 20
	if !GradeEditable(false, "2026-12-20", end, jkt) || GradeEditable(false, "2026-12-20", end.Add(2*time.Minute), jkt) {
		t.Fatal("deadline")
	}
	if GradeEditable(true, "2099-01-01", end, jkt) {
		t.Fatal("finalized grades are locked")
	}
}

func TestGradingRightsMoveOnEffectiveDate(t *testing.T) { // EC-12
	sel := "2026-10-14"
	hist := []academic.DosenAssignment{{DosenID: "old", EfektifMulai: "2000-01-01", EfektifSelesai: &sel}, {DosenID: "new", EfektifMulai: "2026-10-15"}}
	if !CanGrade(hist, "old", "2026-10-14") || CanGrade(hist, "new", "2026-10-14") {
		t.Fatal("before effective date")
	}
	if CanGrade(hist, "old", "2026-10-15") || !CanGrade(hist, "new", "2026-10-15") {
		t.Fatal("from effective date")
	}
	if !Teaches(hist, "old") || CanGrade(hist, "", "2026-10-15") {
		t.Fatal("Teaches/empty")
	}
	if !NeedsReReport(true) || NeedsReReport(false) {
		t.Fatal("EC-11 re-report flag")
	}
}
