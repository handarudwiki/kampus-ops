// Package lecture covers meetings, attendance, grading, finalisation and grade
// corrections (FR-21..23, EC-11, EC-12). Pure rules live in this file.
package lecture

import (
	"fmt"
	"math"
	"time"

	"github.com/kampusops/backend/internal/academic"
	"github.com/kampusops/backend/internal/domain"
)

// ScoresToX100 validates and converts component scores (0..100, decimals allowed) to ×100 ints.
func ScoresToX100(komponen []domain.Komponen, in map[string]*float64) (map[string]*int, error) {
	out := map[string]*int{}
	known := map[string]bool{}
	for _, k := range komponen {
		known[k.Key] = true
	}
	for key, v := range in {
		if !known[key] {
			return nil, fmt.Errorf("komponen %q tidak dikenal", key)
		}
		if v == nil {
			out[key] = nil
			continue
		}
		if *v < 0 || *v > 100 || math.IsNaN(*v) {
			return nil, fmt.Errorf("nilai %s harus 0..100", key)
		}
		x := int(math.Round(*v * 100))
		out[key] = &x
	}
	return out, nil
}

// Grade is the computed result of a grade row.
type Grade struct {
	AngkaX100 *int
	Huruf     *string
	BobotX100 *int
	Complete  bool
}

// ComputeGrade applies FR-22: weighted components → numeric score → letter + bobot.
func ComputeGrade(r domain.Rules, komponen []domain.Komponen, scores map[string]*int) Grade {
	angka, complete := domain.ComputeAngka(komponen, scores)
	if !complete {
		return Grade{}
	}
	h, b := r.Letter(angka)
	return Grade{AngkaX100: &angka, Huruf: &h, BobotX100: &b, Complete: true}
}

// GradeEditable implements the FR-23 lock: no edits after finalisation or after the
// grade-input deadline (end of nilai_selesai in tenant tz, EC-17).
func GradeEditable(finalized bool, nilaiSelesai string, now time.Time, loc *time.Location) bool {
	if finalized {
		return false
	}
	end, ok := domain.DayEnd(nilaiSelesai, loc)
	if !ok {
		return true
	}
	return now.Before(end)
}

// CanGrade implements EC-12: grading rights belong to the lecturer assigned on `today`.
func CanGrade(hist []academic.DosenAssignment, dosenID, today string) bool {
	if dosenID == "" {
		return false
	}
	cur, ok := academic.DosenOn(hist, today)
	return ok && cur == dosenID
}

// Teaches reports whether dosenID was ever assigned to the class (read access).
func Teaches(hist []academic.DosenAssignment, dosenID string) bool {
	for _, h := range hist {
		if h.DosenID == dosenID {
			return true
		}
	}
	return false
}

// NeedsReReport is EC-11: a correction after the period was exported must be re-reported.
func NeedsReReport(periodeExported bool) bool { return periodeExported }
