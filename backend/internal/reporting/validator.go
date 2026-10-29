// Package reporting contains the PDDikti pre-export validator (PV-01..PV-08), readiness
// score, and Feeder-like CSV export (FR-29..31).
package reporting

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Rule metadata (BRD §5.5).
type RuleDef struct {
	ID    string
	Label string
	Level string // error | warning
}

var Rules = []RuleDef{
	{"PV-01", "Mahasiswa aktif tanpa KRS", "error"},
	{"PV-02", "Nilai tanpa KRS yang sesuai", "error"},
	{"PV-03", "Kelas tanpa dosen pengampu", "error"},
	{"PV-04", "SKS kelas berbeda dengan SKS mata kuliah", "error"},
	{"PV-05", "NIK kosong, bukan 16 digit, atau duplikat", "error"},
	{"PV-06", "Total SKS melebihi batas tanpa dispensasi", "warning"},
	{"PV-07", "Mahasiswa cuti/non-aktif memiliki KRS", "error"},
	{"PV-08", "Nilai perlu lapor ulang setelah koreksi", "warning"},
}

func ruleDef(id string) RuleDef {
	for _, r := range Rules {
		if r.ID == id {
			return r
		}
	}
	return RuleDef{}
}

// Snapshot inputs (one period).
type Student struct {
	ID, NIM, Nama  string
	NIK            *string
	Status         string // status in the period (from riwayat_status)
	KrsID          *string
	KrsItems       int
	KrsTotalSks    int
	KrsMaxSks      int
	HasDispensasi  bool // any approved dispensation in the period
}

type NilaiRec struct {
	ID, KelasID, MahasiswaID, NIM, Nama, MkNama string
	MatchingKrs                                 bool // linked to an active krs_detail of the same student & class
	PerluLaporUlang                             bool
}

type KelasRec struct {
	ID, Label string
	DosenID   *string
	Sks       int
	MkSks     int
}

type Snapshot struct {
	PeriodeID string
	Students  []Student
	Nilai     []NilaiRec
	Kelas     []KelasRec
	// AllNIK counts NIKs across the tenant (duplicates are checked tenant-wide).
	NIKCount map[string]int
}

type Issue struct {
	ID          string `json:"id"`
	Rule        string `json:"rule"`
	Level       string `json:"level"`
	Entity      string `json:"entity"`
	EntityID    string `json:"entity_id"`
	EntityLabel string `json:"entity_label"`
	Message     string `json:"message"`
	FixURL      string `json:"fix_url"`
}

type RuleCount struct {
	Rule  string `json:"rule"`
	Label string `json:"label"`
	Level string `json:"level"`
	Count int    `json:"count"`
}

type Result struct {
	PeriodeID string      `json:"periode_id"`
	Score     int         `json:"score"`
	Errors    int         `json:"errors"`
	Warnings  int         `json:"warnings"`
	RanAt     string      `json:"ran_at"`
	ByRule    []RuleCount `json:"by_rule"`
	Items     []Issue     `json:"items"`
}

var nik16 = regexp.MustCompile(`^\d{16}$`)

func reportable(status string) bool { return status == "aktif" || status == "cuti" || status == "non_aktif" }

// Check helpers — each PV rule is a pure function so it can be unit-tested alone.

func PV01(s Student) bool { return s.Status == "aktif" && (s.KrsID == nil || s.KrsItems == 0) }

func PV02(n NilaiRec) bool { return !n.MatchingKrs }

func PV03(k KelasRec) bool { return k.DosenID == nil }

func PV04(k KelasRec) bool { return k.Sks != k.MkSks }

// PV05 returns a reason when the NIK is empty, not 16 digits, or duplicated in the tenant.
func PV05(s Student, nikCount map[string]int) string {
	switch {
	case s.NIK == nil || *s.NIK == "":
		return "NIK kosong"
	case !nik16.MatchString(*s.NIK):
		return "NIK bukan 16 digit"
	case nikCount[*s.NIK] > 1:
		return "NIK duplikat"
	}
	return ""
}

func PV06(s Student) bool { return s.KrsID != nil && s.KrsTotalSks > s.KrsMaxSks && !s.HasDispensasi }

func PV07(s Student) bool { return (s.Status == "cuti" || s.Status == "non_aktif") && s.KrsID != nil && s.KrsItems > 0 }

func PV08(n NilaiRec) bool { return n.PerluLaporUlang }

// Validate runs PV-01..PV-08 over a snapshot. Every issue carries the SPA route where
// the operator can fix it (FR-31).
func Validate(s Snapshot, now time.Time) Result {
	var items []Issue
	add := func(rule, entity, id, label, msg, fix string) {
		d := ruleDef(rule)
		items = append(items, Issue{ID: uuid.NewString(), Rule: rule, Level: d.Level, Entity: entity, EntityID: id, EntityLabel: label, Message: msg, FixURL: fix})
	}
	checks := 0
	for _, st := range s.Students {
		if !reportable(st.Status) {
			continue
		}
		checks += 4
		label := st.NIM + " — " + st.Nama
		if PV01(st) {
			add("PV-01", "mahasiswa", st.ID, label, "Mahasiswa berstatus aktif tetapi tidak memiliki KRS pada periode ini", "/mahasiswa/"+st.ID)
		}
		if r := PV05(st, s.NIKCount); r != "" {
			add("PV-05", "mahasiswa", st.ID, label, r, "/mahasiswa/"+st.ID)
		}
		if PV06(st) {
			add("PV-06", "krs", *st.KrsID, label, fmt.Sprintf("Total %d SKS melebihi batas %d SKS tanpa dispensasi", st.KrsTotalSks, st.KrsMaxSks), "/krs/"+*st.KrsID)
		}
		if PV07(st) {
			add("PV-07", "krs", *st.KrsID, label, fmt.Sprintf("Mahasiswa berstatus %s tetapi memiliki KRS", st.Status), "/krs/"+*st.KrsID)
		}
	}
	for _, n := range s.Nilai {
		checks += 2
		label := n.NIM + " — " + n.MkNama
		if PV02(n) {
			add("PV-02", "nilai", n.ID, label, "Nilai tidak memiliki KRS yang sesuai (mahasiswa/kelas tidak cocok atau KRS dibatalkan)", "/kelas/"+n.KelasID+"/nilai")
		}
		if PV08(n) {
			add("PV-08", "nilai", n.ID, label, "Nilai dikoreksi setelah periode diekspor; perlu lapor ulang", "/kelas/"+n.KelasID+"/nilai")
		}
	}
	for _, k := range s.Kelas {
		checks += 2
		if PV03(k) {
			add("PV-03", "kelas", k.ID, k.Label, "Kelas belum memiliki dosen pengampu", "/kelas/"+k.ID)
		}
		if PV04(k) {
			add("PV-04", "kelas", k.ID, k.Label, fmt.Sprintf("SKS kelas %d berbeda dengan SKS mata kuliah %d", k.Sks, k.MkSks), "/kelas/"+k.ID)
		}
	}
	res := Result{PeriodeID: s.PeriodeID, RanAt: now.UTC().Format(time.RFC3339), Items: items}
	if res.Items == nil {
		res.Items = []Issue{}
	}
	counts := map[string]int{}
	for _, it := range items {
		counts[it.Rule]++
		if it.Level == "error" {
			res.Errors++
		} else {
			res.Warnings++
		}
	}
	for _, r := range Rules {
		res.ByRule = append(res.ByRule, RuleCount{Rule: r.ID, Label: r.Label, Level: r.Level, Count: counts[r.ID]})
	}
	sort.SliceStable(res.Items, func(i, j int) bool {
		if res.Items[i].Level != res.Items[j].Level {
			return res.Items[i].Level == "error"
		}
		return res.Items[i].Rule < res.Items[j].Rule
	})
	res.Score = Score(checks, res.Errors, res.Warnings)
	return res
}

// Score is the readiness percentage: each error costs a full check, each warning half.
func Score(checks, errors, warnings int) int {
	if checks == 0 {
		return 100
	}
	pen := float64(errors) + 0.5*float64(warnings)
	v := 100 * (1 - pen/float64(checks))
	if v < 0 {
		v = 0
	}
	// an export-blocking error never shows as 100%
	sc := int(math.Floor(v))
	if errors > 0 && sc >= 100 {
		sc = 99
	}
	return sc
}
