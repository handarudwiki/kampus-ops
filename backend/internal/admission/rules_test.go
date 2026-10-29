package admission

import (
	"testing"
	"time"

	"github.com/kampusops/backend/internal/domain"
)

func TestPMBStateMachine(t *testing.T) { // FR-09
	happy := []string{Daftar, BayarFormulir, LengkapiBerkas, Verifikasi, Lulus, DaftarUlang}
	for i := 0; i+1 < len(happy); i++ {
		if !CanTransition(happy[i], happy[i+1]) {
			t.Fatalf("%s → %s", happy[i], happy[i+1])
		}
	}
	for _, c := range [][2]string{{Daftar, Lulus}, {Verifikasi, DaftarUlang}, {TidakLulus, DaftarUlang}, {DaftarUlang, Daftar}, {Lulus, TidakLulus}, {Daftar, Daftar}} {
		if CanTransition(c[0], c[1]) {
			t.Errorf("%s → %s must be invalid", c[0], c[1])
		}
	}
	if !CanTransition(Verifikasi, TidakLulus) || !CanTransition(Verifikasi, LengkapiBerkas) {
		t.Fatal("verifikasi branches")
	}
	if !ApplicantMay(BayarFormulir, LengkapiBerkas) || ApplicantMay(Verifikasi, Lulus) {
		t.Fatal("applicant permissions")
	}
}

func TestFormatNIM(t *testing.T) { // FR-10
	if got := FormatNIM("{angkatan}{prodi}{seq:4}", 2024, "11", 1); got != "2024110001" {
		t.Fatal(got)
	}
	if got := FormatNIM("{yy}.{prodi}.{seq:3}", 2026, "TI", 42); got != "26.TI.042" {
		t.Fatal(got)
	}
	if got := FormatNIM("{angkatan}{prodi}{seq}", 2026, "21", 7); got != "2026210007" {
		t.Fatal(got)
	}
}

func TestGelombangOpenAndNIK(t *testing.T) {
	loc := domain.LoadLocation("Asia/Makassar")
	now := time.Date(2026, 3, 31, 15, 30, 0, 0, time.UTC) // 23:30 WITA Mar 31
	if !GelombangOpen("2026-01-05", "2026-03-31", now, loc) || GelombangOpen("2026-01-05", "2026-03-31", now.Add(time.Hour), loc) {
		t.Fatal("wave window in tenant tz")
	}
	if !ValidNIK("3201010101010101") || ValidNIK("12345") || ValidNIK("320101010101010A") {
		t.Fatal("ValidNIK")
	}
}
