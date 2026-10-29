package auth

import (
	"testing"
	"time"
)

func TestLockAfterFiveFailures(t *testing.T) { // FR-04
	now := time.Date(2026, 10, 29, 8, 0, 0, 0, time.UTC)
	failed := 0
	var lock *time.Time
	var locked bool
	for i := 1; i <= 5; i++ {
		failed, lock, locked = RegisterFailure(failed, now)
		if i < 5 && (locked || lock != nil || failed != i) {
			t.Fatalf("attempt %d locked too early", i)
		}
	}
	if !locked || lock == nil || failed != 0 {
		t.Fatal("5th failure must lock and reset the counter")
	}
	if !IsLocked(lock, now.Add(14*time.Minute)) || IsLocked(lock, now.Add(15*time.Minute)) || IsLocked(nil, now) {
		t.Fatal("lock duration")
	}
	if !IsValidRole("dosen_wali") || IsValidRole("root") {
		t.Fatal("roles")
	}
}
