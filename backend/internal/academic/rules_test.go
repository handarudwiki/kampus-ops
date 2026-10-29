package academic

import "testing"

func sp(s string) *string { return &s }

func TestOverlaps(t *testing.T) {
	a := Slot{Hari: 1, JamMulai: "07:30", JamSelesai: "10:00"}
	if !Overlaps(a, Slot{1, "09:59", "11:00"}) || Overlaps(a, Slot{1, "10:00", "11:00"}) || Overlaps(a, Slot{2, "07:30", "10:00"}) {
		t.Fatal("Overlaps")
	}
	if _, _, err := (Slot{1, "10:00", "09:00"}).Minutes(); err == nil {
		t.Fatal("end before start must fail")
	}
	if _, err := ParseHHMM("25:00"); err == nil {
		t.Fatal("invalid hour")
	}
}

func TestFindClashes(t *testing.T) { // FR-15
	others := []ScheduledClass{
		{ID: "k1", RuangID: sp("R1"), DosenID: sp("D1"), Slot: Slot{1, "07:30", "10:00"}},
		{ID: "k2", RuangID: sp("R2"), DosenID: sp("D2"), Slot: Slot{1, "08:00", "09:40"}},
		{ID: "k3", RuangID: sp("R1"), DosenID: sp("D3"), Slot: Slot{2, "07:30", "10:00"}},
	}
	room, lect := FindClashes(ScheduledClass{ID: "new", RuangID: sp("R1"), DosenID: sp("D2"), Slot: Slot{1, "09:00", "10:40"}}, others)
	if len(room) != 1 || room[0] != "k1" || len(lect) != 1 || lect[0] != "k2" {
		t.Fatalf("room=%v lect=%v", room, lect)
	}
	// updating k1 itself never clashes with itself; no room/lecturer → no clash
	room, lect = FindClashes(ScheduledClass{ID: "k1", RuangID: sp("R1"), DosenID: sp("D1"), Slot: Slot{1, "07:30", "10:00"}}, others)
	if len(room)+len(lect) != 0 {
		t.Fatal("self clash")
	}
	room, lect = FindClashes(ScheduledClass{ID: "x", Slot: Slot{1, "07:30", "10:00"}}, others)
	if len(room)+len(lect) != 0 {
		t.Fatal("nil room/lecturer")
	}
}

func TestDosenOnEffectiveDate(t *testing.T) { // EC-12
	hist := []DosenAssignment{{DosenID: "old", EfektifMulai: "2000-01-01", EfektifSelesai: sp("2026-10-14")}, {DosenID: "new", EfektifMulai: "2026-10-15"}}
	if d, _ := DosenOn(hist, "2026-10-14"); d != "old" {
		t.Fatal("day before effective date belongs to old lecturer")
	}
	if d, _ := DosenOn(hist, "2026-10-15"); d != "new" {
		t.Fatal("effective date belongs to new lecturer")
	}
	if _, ok := DosenOn(nil, "2026-10-15"); ok {
		t.Fatal("no history")
	}
	if !ShouldCancelForLowEnrollment(3, 5) || ShouldCancelForLowEnrollment(5, 5) {
		t.Fatal("EC-13 minimum")
	}
}
