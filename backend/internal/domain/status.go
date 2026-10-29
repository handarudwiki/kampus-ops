package domain

// Student statuses (BRD §5.6).
const (
	StatusAktif    = "aktif"
	StatusCuti     = "cuti"
	StatusNonAktif = "non_aktif"
	StatusLulus    = "lulus"
	StatusKeluar   = "keluar"
	StatusDO       = "do"
)

var statusTransitions = map[string][]string{
	StatusAktif:    {StatusCuti, StatusNonAktif, StatusLulus, StatusKeluar, StatusDO},
	StatusCuti:     {StatusAktif, StatusNonAktif, StatusKeluar, StatusDO},
	StatusNonAktif: {StatusAktif, StatusCuti, StatusKeluar, StatusDO},
}

// CanTransition reports whether a student may move from one status to another.
// lulus, keluar and do are terminal.
func CanTransition(from, to string) bool {
	for _, s := range statusTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// IsValidStatus reports whether s is a StatusMhs value.
func IsValidStatus(s string) bool {
	switch s {
	case StatusAktif, StatusCuti, StatusNonAktif, StatusLulus, StatusKeluar, StatusDO:
		return true
	}
	return false
}

// CutiAllowed implements EC-08: at most max semesters of cuti during the study.
// usedCuti counts distinct periods already spent on cuti (excluding the target period).
func CutiAllowed(usedCuti, max int) bool { return usedCuti < max }

// SemesterKe returns the student's semester number in a period. Pendek periods do not
// add a regular semester (EC-07).
func SemesterKe(angkatan, tahun int, tipe string) int {
	n := (tahun - angkatan) * 2
	switch tipe {
	case "Ganjil":
		n++
	default: // Genap and Pendek (which belongs to the even semester's year)
		n += 2
	}
	if n < 1 {
		n = 1
	}
	return n
}

// MasaStudi computes the semesters consumed (cuti does not count) and remaining (EC-09).
func MasaStudi(semesterKe, cutiCount, masaStudiMaks int) (used, sisa int) {
	used = semesterKe - cutiCount
	if used < 0 {
		used = 0
	}
	return used, masaStudiMaks - used
}

// MasaStudiFlag: "warning" when sisa ≤ warnAt, "do_candidate" when exceeded, "" otherwise.
func MasaStudiFlag(sisa, warnAt int) string {
	switch {
	case sisa < 0:
		return "do_candidate"
	case sisa <= warnAt:
		return "warning"
	}
	return ""
}
