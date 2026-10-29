// Package admission implements PMB: waves, registration, the applicant state machine
// and re-registration that issues a NIM without duplicates (FR-08..10, EC-04).
package admission

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kampusops/backend/internal/domain"
)

// PMB statuses (FR-09).
const (
	Daftar         = "daftar"
	BayarFormulir  = "bayar_formulir"
	LengkapiBerkas = "lengkapi_berkas"
	Verifikasi     = "verifikasi"
	Lulus          = "lulus"
	TidakLulus     = "tidak_lulus"
	DaftarUlang    = "daftar_ulang"
)

var transitions = map[string][]string{
	Daftar:         {BayarFormulir},
	BayarFormulir:  {LengkapiBerkas},
	LengkapiBerkas: {Verifikasi},
	Verifikasi:     {Lulus, TidakLulus, LengkapiBerkas}, // back to berkas when documents are rejected
	Lulus:          {DaftarUlang},
}

// CanTransition reports whether the PMB state machine allows from → to.
func CanTransition(from, to string) bool {
	for _, t := range transitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

// ApplicantMay restricts what the calon_mahasiswa may do on their own record.
func ApplicantMay(from, to string) bool {
	return (from == BayarFormulir && to == LengkapiBerkas) || (from == LengkapiBerkas && to == Verifikasi)
}

var seqRe = regexp.MustCompile(`\{seq(?::(\d+))?\}`)

// FormatNIM renders a NIM from the tenant format (FR-10), e.g. "{angkatan}{prodi}{seq:4}"
// → 2026110001. Supported tokens: {angkatan}, {yy}, {prodi}, {seq[:width]}.
func FormatNIM(format string, angkatan int, prodiKode string, seq int) string {
	s := strings.ReplaceAll(format, "{angkatan}", strconv.Itoa(angkatan))
	s = strings.ReplaceAll(s, "{yy}", fmt.Sprintf("%02d", angkatan%100))
	s = strings.ReplaceAll(s, "{prodi}", prodiKode)
	return seqRe.ReplaceAllStringFunc(s, func(m string) string {
		w := 4
		if sub := seqRe.FindStringSubmatch(m); sub[1] != "" {
			w, _ = strconv.Atoi(sub[1])
		}
		return fmt.Sprintf("%0*d", w, seq)
	})
}

// GelombangOpen reports whether a wave accepts registrations at now (tenant tz).
func GelombangOpen(mulai, selesai string, now time.Time, loc *time.Location) bool {
	return domain.InRange(now, mulai, selesai, loc)
}

// DefaultBerkas is the document checklist for a new applicant.
func DefaultBerkas() []Berkas {
	return []Berkas{{Nama: "KTP", OK: false}, {Nama: "Ijazah / SKL", OK: false}, {Nama: "Pas foto", OK: false}}
}

type Berkas struct {
	Nama string `json:"nama"`
	OK   bool   `json:"ok"`
}

var nikRe = regexp.MustCompile(`^\d{16}$`)

// ValidNIK checks the 16-digit NIK format (also used by PV-05).
func ValidNIK(nik string) bool { return nikRe.MatchString(nik) }
