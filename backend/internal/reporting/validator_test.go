package reporting

import (
	"testing"
	"time"
)

func s(v string) *string { return &v }

func TestEachPVRule(t *testing.T) {
	krs := s("krs-1")
	// PV-01
	if !PV01(Student{Status: "aktif"}) || !PV01(Student{Status: "aktif", KrsID: krs}) || PV01(Student{Status: "aktif", KrsID: krs, KrsItems: 2}) || PV01(Student{Status: "cuti"}) {
		t.Fatal("PV-01")
	}
	// PV-02
	if !PV02(NilaiRec{MatchingKrs: false}) || PV02(NilaiRec{MatchingKrs: true}) {
		t.Fatal("PV-02")
	}
	// PV-03 / PV-04
	if !PV03(KelasRec{}) || PV03(KelasRec{DosenID: s("d")}) || !PV04(KelasRec{Sks: 2, MkSks: 3}) || PV04(KelasRec{Sks: 3, MkSks: 3}) {
		t.Fatal("PV-03/04")
	}
	// PV-05
	counts := map[string]int{"3201010101010101": 2, "3201010101010102": 1}
	if PV05(Student{}, counts) != "NIK kosong" || PV05(Student{NIK: s("")}, counts) != "NIK kosong" ||
		PV05(Student{NIK: s("12345")}, counts) != "NIK bukan 16 digit" || PV05(Student{NIK: s("3201010101010101")}, counts) != "NIK duplikat" ||
		PV05(Student{NIK: s("3201010101010102")}, counts) != "" {
		t.Fatal("PV-05")
	}
	// PV-06 (warning)
	if !PV06(Student{KrsID: krs, KrsTotalSks: 25, KrsMaxSks: 24}) || PV06(Student{KrsID: krs, KrsTotalSks: 25, KrsMaxSks: 24, HasDispensasi: true}) ||
		PV06(Student{KrsID: krs, KrsTotalSks: 24, KrsMaxSks: 24}) {
		t.Fatal("PV-06")
	}
	// PV-07
	if !PV07(Student{Status: "cuti", KrsID: krs, KrsItems: 1}) || !PV07(Student{Status: "non_aktif", KrsID: krs, KrsItems: 3}) || PV07(Student{Status: "aktif", KrsID: krs, KrsItems: 3}) || PV07(Student{Status: "cuti"}) {
		t.Fatal("PV-07")
	}
	// PV-08 (warning)
	if !PV08(NilaiRec{PerluLaporUlang: true}) || PV08(NilaiRec{}) {
		t.Fatal("PV-08")
	}
}

func TestValidateSnapshotFixURLsAndScore(t *testing.T) {
	snap := Snapshot{
		PeriodeID: "p",
		Students: []Student{
			{ID: "m1", NIM: "1", Nama: "A", NIK: s("3201010101010101"), Status: "aktif"},                                  // PV-01
			{ID: "m2", NIM: "2", Nama: "B", NIK: s("123"), Status: "aktif", KrsID: s("k2"), KrsItems: 5, KrsTotalSks: 26, KrsMaxSks: 24}, // PV-05, PV-06
			{ID: "m3", NIM: "3", Nama: "C", NIK: s("3201010101010103"), Status: "cuti", KrsID: s("k3"), KrsItems: 1},       // PV-07
			{ID: "m4", NIM: "4", Nama: "D", Status: "lulus"},                                                              // ignored
		},
		Nilai: []NilaiRec{{ID: "n1", KelasID: "c1", MatchingKrs: false}, {ID: "n2", KelasID: "c1", MatchingKrs: true, PerluLaporUlang: true}},
		Kelas: []KelasRec{{ID: "c1", Sks: 3, MkSks: 3, DosenID: s("d")}, {ID: "c2", Sks: 2, MkSks: 3}},
		NIKCount: map[string]int{"3201010101010101": 1, "123": 1, "3201010101010103": 1},
	}
	r := Validate(snap, time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC))
	want := map[string]int{"PV-01": 1, "PV-02": 1, "PV-03": 1, "PV-04": 1, "PV-05": 1, "PV-06": 1, "PV-07": 1, "PV-08": 1}
	for _, b := range r.ByRule {
		if b.Count != want[b.Rule] {
			t.Errorf("%s = %d", b.Rule, b.Count)
		}
	}
	if len(r.ByRule) != 8 || r.Errors != 6 || r.Warnings != 2 {
		t.Fatalf("errors=%d warnings=%d", r.Errors, r.Warnings)
	}
	fix := map[string]string{}
	for _, it := range r.Items {
		fix[it.Rule] = it.FixURL
	}
	exp := map[string]string{"PV-01": "/mahasiswa/m1", "PV-02": "/kelas/c1/nilai", "PV-03": "/kelas/c2", "PV-04": "/kelas/c2", "PV-05": "/mahasiswa/m2",
		"PV-06": "/krs/k2", "PV-07": "/krs/k3", "PV-08": "/kelas/c1/nilai"}
	for k, v := range exp {
		if fix[k] != v {
			t.Errorf("%s fix_url %s want %s", k, fix[k], v)
		}
	}
	if r.Items[0].Level != "error" {
		t.Fatal("errors first")
	}
	if r.Score >= 100 || r.Score < 0 {
		t.Fatalf("score %d", r.Score)
	}
}

func TestScore(t *testing.T) {
	if Score(0, 0, 0) != 100 || Score(100, 0, 0) != 100 || Score(100, 10, 0) != 90 || Score(100, 0, 10) != 95 || Score(10, 50, 0) != 0 {
		t.Fatal("Score")
	}
	if Score(10000, 1, 0) != 99 {
		t.Fatal("any error keeps score below 100")
	}
}
