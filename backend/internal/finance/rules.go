// Package finance handles bills, installments, simulated VA payments with webhook
// dedup, dispensations and receivables (FR-25..28, EC-03, EC-15).
package finance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/kampusops/backend/internal/domain"
)

// StatusFor derives the bill status from the paid total (never from the last event).
func StatusFor(jumlah, terbayar int64) string {
	switch {
	case terbayar <= 0:
		return "belum_bayar"
	case terbayar >= jumlah:
		return "lunas"
	default:
		return "sebagian"
	}
}

// Payment is one webhook event.
type Payment struct {
	TransactionID string
	Amount        int64
	PaidAt        time.Time
}

// Reconcile implements EC-03: events are deduplicated by transaction_id and the total is
// a sum, so duplicates and out-of-order delivery always yield the same state.
func Reconcile(jumlah int64, events []Payment) (int64, string) {
	seen := map[string]bool{}
	var total int64
	for _, e := range events {
		if seen[e.TransactionID] || e.Amount <= 0 {
			continue
		}
		seen[e.TransactionID] = true
		total += e.Amount
	}
	return total, StatusFor(jumlah, total)
}

// Installment is one cicilan.
type Installment struct {
	Ke         int    `json:"ke"`
	Jumlah     int64  `json:"jumlah"`
	JatuhTempo string `json:"jatuh_tempo"`
	Lunas      bool   `json:"lunas"`
}

// SplitCicilan splits jumlah into n monthly installments; the remainder goes to the last
// one so the sum is exact. firstDue is YYYY-MM-DD.
func SplitCicilan(jumlah int64, n int, firstDue string) []Installment {
	base := jumlah / int64(n)
	start, err := time.Parse(domain.DateLayout, firstDue)
	if err != nil {
		start = time.Now().UTC()
	}
	out := make([]Installment, n)
	for i := 0; i < n; i++ {
		amt := base
		if i == n-1 {
			amt = jumlah - base*int64(n-1)
		}
		out[i] = Installment{Ke: i + 1, Jumlah: amt, JatuhTempo: start.AddDate(0, i, 0).Format(domain.DateLayout)}
	}
	return out
}

// MarkLunas flags installments covered by the cumulative paid amount.
func MarkLunas(inst []Installment, terbayar int64) []Installment {
	var cum int64
	for i := range inst {
		cum += inst[i].Jumlah
		inst[i].Lunas = terbayar >= cum
	}
	return inst
}

// Sign returns the hex HMAC-SHA256 of body (X-Signature header).
func Sign(secret string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

// VerifySignature checks X-Signature in constant time.
func VerifySignature(secret string, body []byte, sig string) bool {
	want, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return hmac.Equal(m.Sum(nil), want)
}
