package domain

import "time"

const DateLayout = "2006-01-02"

// LoadLocation returns the tenant's location, falling back to UTC.
func LoadLocation(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	return time.UTC
}

// LocalDate is the calendar date of instant t in loc (EC-17).
func LocalDate(t time.Time, loc *time.Location) string { return t.In(loc).Format(DateLayout) }

// DayStart is 00:00 of date in loc.
func DayStart(date string, loc *time.Location) (time.Time, bool) {
	t, err := time.ParseInLocation(DateLayout, date, loc)
	return t, err == nil
}

// DayEnd is the exclusive end (00:00 of the next day) of date in loc.
func DayEnd(date string, loc *time.Location) (time.Time, bool) {
	t, ok := DayStart(date, loc)
	if !ok {
		return t, false
	}
	return t.AddDate(0, 0, 1), true
}

// Tahapan is the period stage schedule (FR-06); dates are YYYY-MM-DD in tenant tz.
type Tahapan struct {
	KrsMulai      string  `json:"krs_mulai"`
	KrsSelesai    string  `json:"krs_selesai"`
	RevisiMulai   *string `json:"revisi_mulai"`
	RevisiSelesai *string `json:"revisi_selesai"`
	KuliahMulai   string  `json:"kuliah_mulai"`
	KuliahSelesai string  `json:"kuliah_selesai"`
	UTS           string  `json:"uts"`
	UAS           string  `json:"uas"`
	NilaiSelesai  string  `json:"nilai_selesai"`
}

// InRange reports start ≤ now < end-of-day(endDate) in loc.
func InRange(now time.Time, startDate, endDate string, loc *time.Location) bool {
	s, ok1 := DayStart(startDate, loc)
	e, ok2 := DayEnd(endDate, loc)
	if !ok1 || !ok2 {
		return false
	}
	return !now.Before(s) && now.Before(e)
}
