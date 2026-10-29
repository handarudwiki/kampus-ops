// Package auth implements login with lockout, refresh-token rotation, /me,
// tenant listing and user/role administration (FR-01..04).
package auth

import "time"

const (
	MaxFailedLogins = 5
	LockDuration    = 15 * time.Minute
)

// ValidRoles is the contract's Role enum.
var ValidRoles = []string{"super_admin", "admin_kampus", "operator", "kaprodi", "dosen_wali", "dosen", "keuangan", "mahasiswa", "calon_mahasiswa"}

func IsValidRole(r string) bool {
	for _, v := range ValidRoles {
		if v == r {
			return true
		}
	}
	return false
}

// IsLocked reports whether an account is locked at now.
func IsLocked(lockedUntil *time.Time, now time.Time) bool {
	return lockedUntil != nil && now.Before(*lockedUntil)
}

// RegisterFailure returns the new failure counter and lock deadline after a failed
// login (FR-04): the 5th consecutive failure locks the account for LockDuration.
func RegisterFailure(failed int, now time.Time) (int, *time.Time, bool) {
	failed++
	if failed >= MaxFailedLogins {
		t := now.Add(LockDuration)
		return 0, &t, true
	}
	return failed, nil, false
}
