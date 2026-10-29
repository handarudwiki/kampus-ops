// Package academic covers master data, curriculum, class scheduling, student status
// and the student's academic views (KHS, transcript). Pure rules live in this file.
package academic

import (
	"fmt"
	"strconv"
	"strings"
)

// Slot is a weekly time slot (hari 1=Senin .. 6=Sabtu).
type Slot struct {
	Hari       int    `json:"hari"`
	JamMulai   string `json:"jam_mulai"`
	JamSelesai string `json:"jam_selesai"`
}

// ParseHHMM converts "07:30" to minutes since midnight.
func ParseHHMM(s string) (int, error) {
	parts := strings.Split(s, ":")
	if len(parts) < 2 {
		return 0, fmt.Errorf("format jam HH:MM")
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("format jam HH:MM")
	}
	return h*60 + m, nil
}

// Minutes returns the slot's [start, end) in minutes.
func (s Slot) Minutes() (int, int, error) {
	a, err := ParseHHMM(s.JamMulai)
	if err != nil {
		return 0, 0, err
	}
	b, err := ParseHHMM(s.JamSelesai)
	if err != nil {
		return 0, 0, err
	}
	if b <= a {
		return 0, 0, fmt.Errorf("jam selesai harus setelah jam mulai")
	}
	return a, b, nil
}

// Overlaps reports whether two slots are on the same day with intersecting time ranges.
// Back-to-back slots (08:00-09:40 and 09:40-11:20) do not overlap.
func Overlaps(a, b Slot) bool {
	if a.Hari != b.Hari {
		return false
	}
	as, ae, err1 := a.Minutes()
	bs, be, err2 := b.Minutes()
	if err1 != nil || err2 != nil {
		return false
	}
	return as < be && bs < ae
}

// ScheduledClass is the minimum needed for clash detection.
type ScheduledClass struct {
	ID      string
	RuangID *string
	DosenID *string
	Slot    Slot
}

// FindClashes implements FR-15: same room at an overlapping slot, or same lecturer at
// an overlapping slot. others must already be filtered to the same period and active.
func FindClashes(c ScheduledClass, others []ScheduledClass) (room, lecturer []string) {
	for _, o := range others {
		if o.ID == c.ID || !Overlaps(c.Slot, o.Slot) {
			continue
		}
		if c.RuangID != nil && o.RuangID != nil && *c.RuangID == *o.RuangID {
			room = append(room, o.ID)
		}
		if c.DosenID != nil && o.DosenID != nil && *c.DosenID == *o.DosenID {
			lecturer = append(lecturer, o.ID)
		}
	}
	return
}

// DosenAssignment is a row of kelas_dosen (EC-12).
type DosenAssignment struct {
	DosenID        string
	EfektifMulai   string  // YYYY-MM-DD
	EfektifSelesai *string // inclusive, nil = open
}

// DosenOn returns the lecturer holding the class on date (YYYY-MM-DD). Grading rights
// move to the new lecturer from the effective date; earlier attendance keeps the old one.
func DosenOn(hist []DosenAssignment, date string) (string, bool) {
	best := ""
	bestStart := ""
	for _, h := range hist {
		if h.EfektifMulai <= date && (h.EfektifSelesai == nil || date <= *h.EfektifSelesai) {
			if h.EfektifMulai >= bestStart {
				best, bestStart = h.DosenID, h.EfektifMulai
			}
		}
	}
	return best, best != ""
}

// ShouldCancelForLowEnrollment is the EC-13 trigger rule (used by operators/UI hints).
func ShouldCancelForLowEnrollment(terisi, minPeserta int) bool { return terisi < minPeserta }
