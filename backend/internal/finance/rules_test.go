package finance

import (
	"math/rand"
	"testing"
	"time"
)

func TestStatusFor(t *testing.T) {
	if StatusFor(1000, 0) != "belum_bayar" || StatusFor(1000, 1) != "sebagian" || StatusFor(1000, 1000) != "lunas" || StatusFor(1000, 1500) != "lunas" {
		t.Fatal("StatusFor")
	}
}

func TestReconcileDuplicateAndOutOfOrder(t *testing.T) { // EC-03
	t0 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	events := []Payment{
		{TransactionID: "A", Amount: 3_000_000, PaidAt: t0},
		{TransactionID: "B", Amount: 2_000_000, PaidAt: t0.Add(time.Hour)},
		{TransactionID: "C", Amount: 2_500_000, PaidAt: t0.Add(2 * time.Hour)},
	}
	want, wantStatus := Reconcile(7_500_000, events)
	if want != 7_500_000 || wantStatus != "lunas" {
		t.Fatalf("baseline %d %s", want, wantStatus)
	}
	// duplicates + every permutation of order must produce the same balance
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		ev := append([]Payment{}, events...)
		for j := 0; j < 1+rng.Intn(4); j++ {
			ev = append(ev, events[rng.Intn(len(events))]) // re-delivery
		}
		rng.Shuffle(len(ev), func(a, b int) { ev[a], ev[b] = ev[b], ev[a] })
		if got, st := Reconcile(7_500_000, ev); got != want || st != wantStatus {
			t.Fatalf("iteration %d: %d %s", i, got, st)
		}
	}
	// the "last event" would say sebagian — status is from the sum, not the last event
	if _, st := Reconcile(7_500_000, events[:2]); st != "sebagian" {
		t.Fatal("partial")
	}
	if got, _ := Reconcile(100, []Payment{{TransactionID: "x", Amount: -5}}); got != 0 {
		t.Fatal("non-positive amounts ignored")
	}
}

func TestCicilanSplitAndMarkLunas(t *testing.T) {
	inst := SplitCicilan(1_000_000, 3, "2026-11-05")
	if len(inst) != 3 || inst[0].Jumlah != 333_333 || inst[2].Jumlah != 333_334 || inst[1].JatuhTempo != "2026-12-05" || inst[2].JatuhTempo != "2027-01-05" {
		t.Fatalf("%+v", inst)
	}
	var sum int64
	for _, c := range inst {
		sum += c.Jumlah
	}
	if sum != 1_000_000 {
		t.Fatal("sum must be exact")
	}
	m := MarkLunas(inst, 666_666)
	if !m[0].Lunas || !m[1].Lunas || m[2].Lunas {
		t.Fatalf("%+v", m)
	}
}

func TestWebhookSignature(t *testing.T) {
	body := []byte(`{"transaction_id":"T1"}`)
	sig := Sign("secret", body)
	if !VerifySignature("secret", body, sig) || VerifySignature("other", body, sig) || VerifySignature("secret", []byte(`{}`), sig) || VerifySignature("secret", body, "zz") {
		t.Fatal("signature")
	}
}
