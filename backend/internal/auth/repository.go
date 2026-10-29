package auth

import (
	"context"
	"time"

	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/db"
)

// User is the contract's User.
type User struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Email      string            `json:"email"`
	Identifier string            `json:"identifier"`
	Roles      []appctx.UserRole `json:"roles"`
}

type credRow struct {
	ID          string
	Hash        string
	Failed      int
	LockedUntil *time.Time
}

type Repo struct{}

func (Repo) findCred(ctx context.Context, q db.Querier, identifier string) (credRow, error) {
	var r credRow
	err := q.QueryRow(ctx, `SELECT id, password_hash, failed_attempts, locked_until FROM users
		WHERE tenant_id = $1 AND (lower(email) = lower($2) OR identifier = $2)
		ORDER BY (identifier = $2) DESC LIMIT 1`, appctx.TenantID(ctx), identifier).
		Scan(&r.ID, &r.Hash, &r.Failed, &r.LockedUntil)
	return r, err
}

func (Repo) setFailure(ctx context.Context, q db.Querier, id string, failed int, lock *time.Time) error {
	_, err := q.Exec(ctx, `UPDATE users SET failed_attempts=$3, locked_until=$4, updated_at=now() WHERE id=$1 AND tenant_id=$2`,
		id, appctx.TenantID(ctx), failed, lock)
	return err
}

func (Repo) resetFailures(ctx context.Context, q db.Querier, id string) error {
	_, err := q.Exec(ctx, `UPDATE users SET failed_attempts=0, locked_until=NULL WHERE id=$1 AND tenant_id=$2 AND (failed_attempts<>0 OR locked_until IS NOT NULL)`,
		id, appctx.TenantID(ctx))
	return err
}

// LoadUser returns a user with roles (and scope prodi names), tenant-scoped.
func (Repo) LoadUser(ctx context.Context, q db.Querier, id string) (User, error) {
	var u User
	err := q.QueryRow(ctx, `SELECT id, nama, email, coalesce(identifier, email) FROM users WHERE id=$1 AND tenant_id=$2`,
		id, appctx.TenantID(ctx)).Scan(&u.ID, &u.Name, &u.Email, &u.Identifier)
	if err != nil {
		return u, err
	}
	u.Roles, err = loadRoles(ctx, q, id)
	return u, err
}

var roleOrder = `array_position(ARRAY['super_admin','admin_kampus','operator','kaprodi','dosen_wali','dosen','keuangan','mahasiswa','calon_mahasiswa'], ur.role)`

func loadRoles(ctx context.Context, q db.Querier, userID string) ([]appctx.UserRole, error) {
	rows, err := q.Query(ctx, `SELECT ur.role, ur.scope_prodi_id, p.nama FROM user_role ur
		LEFT JOIN prodi p ON p.id = ur.scope_prodi_id AND p.tenant_id = ur.tenant_id
		WHERE ur.user_id=$1 AND ur.tenant_id=$2 ORDER BY `+roleOrder, userID, appctx.TenantID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []appctx.UserRole{}
	for rows.Next() {
		var r appctx.UserRole
		if err := rows.Scan(&r.Role, &r.ScopeProdiID, &r.ScopeProdiName); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (Repo) insertRefresh(ctx context.Context, q db.Querier, userID, hash string, exp time.Time) error {
	_, err := q.Exec(ctx, `INSERT INTO refresh_token (tenant_id, user_id, token_hash, expires_at) VALUES ($1,$2,$3,$4)`,
		appctx.TenantID(ctx), userID, hash, exp)
	return err
}

type refreshRow struct {
	ID, TenantID, UserID string
	ExpiresAt            time.Time
	RevokedAt            *time.Time
}

// findRefreshForUpdate looks a refresh token up by hash. The tenant is not known yet
// at this point (the token itself identifies it), so this is the one lookup by hash only.
func (Repo) findRefreshForUpdate(ctx context.Context, q db.Querier, hash string) (refreshRow, error) {
	var r refreshRow
	err := q.QueryRow(ctx, `SELECT id, tenant_id, user_id, expires_at, revoked_at FROM refresh_token WHERE token_hash=$1 FOR UPDATE`, hash).
		Scan(&r.ID, &r.TenantID, &r.UserID, &r.ExpiresAt, &r.RevokedAt)
	return r, err
}

func (Repo) revoke(ctx context.Context, q db.Querier, id string) error {
	_, err := q.Exec(ctx, `UPDATE refresh_token SET revoked_at=now(), updated_at=now() WHERE id=$1 AND tenant_id=$2 AND revoked_at IS NULL`, id, appctx.TenantID(ctx))
	return err
}

func (Repo) revokeAll(ctx context.Context, q db.Querier, userID string) error {
	_, err := q.Exec(ctx, `UPDATE refresh_token SET revoked_at=now(), updated_at=now() WHERE user_id=$1 AND tenant_id=$2 AND revoked_at IS NULL`, userID, appctx.TenantID(ctx))
	return err
}

// ReplaceRoles sets a user's roles.
func (Repo) ReplaceRoles(ctx context.Context, q db.Querier, userID string, roles []appctx.UserRole) error {
	tid := appctx.TenantID(ctx)
	if _, err := q.Exec(ctx, `DELETE FROM user_role WHERE user_id=$1 AND tenant_id=$2`, userID, tid); err != nil {
		return err
	}
	for _, r := range roles {
		if _, err := q.Exec(ctx, `INSERT INTO user_role (tenant_id, user_id, role, scope_prodi_id) VALUES ($1,$2,$3,$4)`, tid, userID, r.Role, r.ScopeProdiID); err != nil {
			return err
		}
	}
	return nil
}
