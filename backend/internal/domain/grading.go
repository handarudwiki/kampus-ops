package domain

import (
	"fmt"
	"sort"

	"github.com/shopspring/decimal"
)

// Letter converts a numeric score (×100, e.g. 7850 = 78.50) to letter + bobot×100 (FR-22).
func (r Rules) Letter(angkaX100 int) (string, int) {
	for _, b := range r.GradeScale {
		if angkaX100 >= b.MinAngkax100 {
			return b.Huruf, b.BobotX100
		}
	}
	last := r.GradeScale[len(r.GradeScale)-1]
	return last.Huruf, last.BobotX100
}

// BobotOf returns the bobot×100 of a letter.
func (r Rules) BobotOf(huruf string) (int, bool) {
	for _, b := range r.GradeScale {
		if b.Huruf == huruf {
			return b.BobotX100, true
		}
	}
	return 0, false
}

// IsLulus reports whether a bobot is a passing grade.
func (r Rules) IsLulus(bobotX100 int) bool { return bobotX100 >= r.LulusMinBobotX100 }

// ComputeAngka returns the weighted score ×100 from component scores ×100 and
// whether every component has a value. Pure integer math with half-up rounding.
func ComputeAngka(komponen []Komponen, scores map[string]*int) (int, bool) {
	total := 0
	complete := true
	for _, k := range komponen {
		v, ok := scores[k.Key]
		if !ok || v == nil {
			complete = false
			continue
		}
		total += *v * k.BobotPersen
	}
	return (total + 50) / 100, complete
}

// FormatX100 renders 7850 as "78.50".
func FormatX100(v int) string {
	sign := ""
	if v < 0 {
		sign, v = "-", -v
	}
	return fmt.Sprintf("%s%d.%02d", sign, v/100, v%100)
}

// GradeEntry is one final grade of a student (input for IPS/IPK).
type GradeEntry struct {
	MkID        string
	MkKode      string
	MkNama      string
	Sks         int
	BobotX100   int
	Huruf       string
	PeriodeKode string
	Semester    int
}

// Sum returns Σ(sks×bobot×100) and Σsks.
func Sum(entries []GradeEntry) (num, den int64) {
	for _, e := range entries {
		num += int64(e.Sks) * int64(e.BobotX100)
		den += int64(e.Sks)
	}
	return
}

// IPDecimal is the exact (unrounded) IP value: num / (den×100).
func IPDecimal(num, den int64) decimal.Decimal {
	if den == 0 {
		return decimal.Zero
	}
	return decimal.NewFromInt(num).Div(decimal.NewFromInt(den * 100))
}

// FormatIP renders an IP rounded to 2 decimals only at display time (EC-10).
func FormatIP(num, den int64) string { return IPDecimal(num, den).StringFixed(2) }

// IPx100 is the displayed IP ×100 (half-up rounded); used for rule thresholds so the
// rule matches what the student sees.
func IPx100(num, den int64) int {
	return int(IPDecimal(num, den).Round(2).Mul(decimal.NewFromInt(100)).IntPart())
}

// IPS computes semester GPA from that semester's grades (repeats count as taken).
func IPS(entries []GradeEntry) string { return FormatIP(Sum(entries)) }

// BestGrades keeps the best grade per course (EC-06 / FR-24). Ties keep the latest period.
// canonical maps a course id to its group id (for equivalent courses); nil = identity.
func BestGrades(entries []GradeEntry, canonical func(string) string) []GradeEntry {
	if canonical == nil {
		canonical = func(s string) string { return s }
	}
	best := map[string]GradeEntry{}
	order := []string{}
	for _, e := range entries {
		k := canonical(e.MkID)
		cur, ok := best[k]
		if !ok {
			order = append(order, k)
			best[k] = e
			continue
		}
		if e.BobotX100 > cur.BobotX100 || (e.BobotX100 == cur.BobotX100 && e.PeriodeKode > cur.PeriodeKode) {
			best[k] = e
		}
	}
	out := make([]GradeEntry, 0, len(order))
	for _, k := range order {
		out = append(out, best[k])
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Semester != out[j].Semester {
			return out[i].Semester < out[j].Semester
		}
		return out[i].MkKode < out[j].MkKode
	})
	return out
}


// Cumulative returns IPK numerator/denominator and passed SKS over best grades.
func (r Rules) Cumulative(entries []GradeEntry, canonical func(string) string) (num, den int64, sksLulus int) {
	best := BestGrades(entries, canonical)
	num, den = Sum(best)
	for _, e := range best {
		if r.IsLulus(e.BobotX100) {
			sksLulus += e.Sks
		}
	}
	return
}
