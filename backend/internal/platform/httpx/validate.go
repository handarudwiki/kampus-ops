package httpx

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// V accumulates field validation errors.
type V struct{ errs map[string]string }

func NewV() *V { return &V{errs: map[string]string{}} }

func (v *V) add(f, m string) {
	if _, ok := v.errs[f]; !ok {
		v.errs[f] = m
	}
}

func (v *V) Required(f, s string) *V {
	if strings.TrimSpace(s) == "" {
		v.add(f, "wajib diisi")
	}
	return v
}

func (v *V) UUID(f, s string) *V {
	if _, err := uuid.Parse(s); err != nil {
		v.add(f, "harus berupa UUID")
	}
	return v
}

func (v *V) OneOf(f, s string, opts ...string) *V {
	for _, o := range opts {
		if s == o {
			return v
		}
	}
	v.add(f, "harus salah satu dari: "+strings.Join(opts, ", "))
	return v
}

func (v *V) Range(f string, n, min, max int) *V {
	if n < min || n > max {
		v.add(f, fmt.Sprintf("harus antara %d dan %d", min, max))
	}
	return v
}

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func (v *V) Date(f, s string) *V {
	if !dateRe.MatchString(s) {
		v.add(f, "format tanggal YYYY-MM-DD")
	}
	return v
}

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func (v *V) Email(f, s string) *V {
	if !emailRe.MatchString(s) {
		v.add(f, "email tidak valid")
	}
	return v
}

func (v *V) Check(cond bool, f, msg string) *V {
	if !cond {
		v.add(f, msg)
	}
	return v
}

func (v *V) Err() error {
	if len(v.errs) == 0 {
		return nil
	}
	return Validation(v.errs)
}
