package auth

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/kampusops/backend/internal/audit"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/httpx"
	"github.com/kampusops/backend/internal/platform/token"
	"github.com/kampusops/backend/internal/tenant"
)

type Service struct {
	pool    *pgxpool.Pool
	tenants *tenant.Store
	iss     *token.Issuer
	repo    Repo
	now     func() time.Time
}

func NewService(pool *pgxpool.Pool, tenants *tenant.Store, iss *token.Issuer) *Service {
	return &Service{pool: pool, tenants: tenants, iss: iss, now: time.Now}
}

type LoginResult struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    int           `json:"expires_in"`
	User         User          `json:"user"`
	Tenant       tenant.Tenant `json:"tenant"`
}

var errInvalid = httpx.Unauthorized("AUTH_INVALID_CREDENTIALS", "Identitas atau kata sandi salah")

// dummyHash equalises timing for unknown users.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

func (s *Service) Login(ctx context.Context, tenantCode, identifier, password string) (*LoginResult, error) {
	t, err := s.tenants.ByCode(ctx, tenantCode)
	if err != nil {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return nil, errInvalid
	}
	ctx = appctx.WithTenant(ctx, t.Info())
	cred, err := s.repo.findCred(ctx, s.pool, strings.TrimSpace(identifier))
	if err != nil {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return nil, errInvalid
	}
	now := s.now()
	if IsLocked(cred.LockedUntil, now) {
		return nil, httpx.E(423, "AUTH_LOCKED", "Akun terkunci karena 5 kali gagal login. Coba lagi nanti.")
	}
	if bcrypt.CompareHashAndPassword([]byte(cred.Hash), []byte(password)) != nil {
		failed, lock, locked := RegisterFailure(cred.Failed, now)
		if err := s.repo.setFailure(ctx, s.pool, cred.ID, failed, lock); err != nil {
			return nil, err
		}
		if locked {
			return nil, httpx.E(423, "AUTH_LOCKED", "Akun terkunci karena 5 kali gagal login. Coba lagi dalam 15 menit.")
		}
		return nil, errInvalid
	}
	if err := s.repo.resetFailures(ctx, s.pool, cred.ID); err != nil {
		return nil, err
	}
	return s.issue(ctx, t, cred.ID)
}

func (s *Service) issue(ctx context.Context, t tenant.Full, userID string) (*LoginResult, error) {
	u, err := s.repo.LoadUser(ctx, s.pool, userID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	access, err := s.iss.Access(appctx.Principal{UserID: u.ID, TenantID: t.ID, Name: u.Name, Roles: u.Roles}, now)
	if err != nil {
		return nil, err
	}
	raw, hash := token.NewRefresh()
	if err := s.repo.insertRefresh(ctx, s.pool, u.ID, hash, now.Add(token.RefreshTTL)); err != nil {
		return nil, err
	}
	return &LoginResult{AccessToken: access, RefreshToken: raw, ExpiresIn: int(token.AccessTTL.Seconds()), User: u, Tenant: t.Tenant}, nil
}

// Refresh rotates a refresh token. Reusing a revoked token revokes the whole family.
func (s *Service) Refresh(ctx context.Context, raw string) (*LoginResult, error) {
	var userID string
	var t tenant.Full
	reused := false
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		r, err := s.repo.findRefreshForUpdate(ctx, tx, token.HashRefresh(raw))
		if err != nil {
			return httpx.Unauthorized("AUTH_TOKEN_EXPIRED", "Refresh token tidak valid")
		}
		t, err = s.tenants.Get(ctx, r.TenantID)
		if err != nil {
			return err
		}
		tctx := appctx.WithTenant(ctx, t.Info())
		if r.RevokedAt != nil {
			reused = true
			return s.repo.revokeAll(tctx, tx, r.UserID)
		}
		if s.now().After(r.ExpiresAt) {
			return httpx.Unauthorized("AUTH_TOKEN_EXPIRED", "Refresh token kedaluwarsa")
		}
		userID = r.UserID
		return s.repo.revoke(tctx, tx, r.ID)
	})
	if err != nil {
		return nil, err
	}
	if reused {
		return nil, httpx.Unauthorized("AUTH_TOKEN_EXPIRED", "Refresh token sudah dipakai")
	}
	return s.issue(appctx.WithTenant(ctx, t.Info()), t, userID)
}

func (s *Service) Logout(ctx context.Context, raw string) error {
	return db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		r, err := s.repo.findRefreshForUpdate(ctx, tx, token.HashRefresh(raw))
		if err != nil {
			return nil
		}
		return s.repo.revoke(appctx.WithTenant(ctx, appctx.TenantInfo{ID: r.TenantID}), tx, r.ID)
	})
}

func (s *Service) Me(ctx context.Context) (User, tenant.Tenant, error) {
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return User{}, tenant.Tenant{}, err
	}
	u, err := s.repo.LoadUser(ctx, s.pool, appctx.UserID(ctx))
	return u, t.Tenant, err
}

// HashPassword is exported for seeders / admission.
func HashPassword(p string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	return string(b), err
}

// --- user administration (admin_kampus) ---

type CreateUserInput struct {
	Name       string            `json:"name"`
	Email      string            `json:"email"`
	Identifier string            `json:"identifier"`
	Password   string            `json:"password"`
	Roles      []appctx.UserRole `json:"roles"`
}

func validateRoles(roles []appctx.UserRole, actor appctx.Principal) error {
	for _, r := range roles {
		if !IsValidRole(r.Role) {
			return httpx.ValidationField("roles", "peran tidak dikenal: "+r.Role)
		}
		if r.Role == "super_admin" && actor.ActiveRole != "super_admin" {
			return httpx.ValidationField("roles", "super_admin hanya dapat diberikan oleh super_admin")
		}
	}
	return nil
}

func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (User, error) {
	v := httpx.NewV().Required("name", in.Name).Email("email", in.Email).Check(len(in.Password) >= 8, "password", "minimal 8 karakter").
		Check(len(in.Roles) > 0, "roles", "minimal satu peran")
	if err := v.Err(); err != nil {
		return User{}, err
	}
	p, _ := appctx.PrincipalFrom(ctx)
	if err := validateRoles(in.Roles, p); err != nil {
		return User{}, err
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		return User{}, err
	}
	var ident *string
	if strings.TrimSpace(in.Identifier) != "" {
		x := strings.TrimSpace(in.Identifier)
		ident = &x
	}
	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `INSERT INTO users (tenant_id, nama, email, identifier, password_hash) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			appctx.TenantID(ctx), in.Name, strings.ToLower(in.Email), ident, hash).Scan(&id)
		if db.IsUniqueViolation(err) {
			return httpx.Conflict("DUPLICATE", "Email atau identifier sudah dipakai")
		}
		if err != nil {
			return err
		}
		if err := s.repo.ReplaceRoles(ctx, tx, id, in.Roles); err != nil {
			return err
		}
		return audit.Write(ctx, tx, "user", id, "create", nil, map[string]any{"name": in.Name, "email": in.Email, "roles": in.Roles})
	})
	if err != nil {
		return User{}, err
	}
	return s.repo.LoadUser(ctx, s.pool, id)
}

func (s *Service) SetRoles(ctx context.Context, userID string, roles []appctx.UserRole) (User, error) {
	p, _ := appctx.PrincipalFrom(ctx)
	if len(roles) == 0 {
		return User{}, httpx.ValidationField("roles", "minimal satu peran")
	}
	if err := validateRoles(roles, p); err != nil {
		return User{}, err
	}
	before, err := s.repo.LoadUser(ctx, s.pool, userID)
	if err != nil {
		return User{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		if err := s.repo.ReplaceRoles(ctx, tx, userID, roles); err != nil {
			return err
		}
		return audit.Write(ctx, tx, "user", userID, "set_roles", before.Roles, roles)
	})
	if err != nil {
		return User{}, err
	}
	return s.repo.LoadUser(ctx, s.pool, userID)
}

func (s *Service) ListUsers(ctx context.Context, pg httpx.Page) ([]User, int, error) {
	tid := appctx.TenantID(ctx)
	var total int
	where := `tenant_id=$1 AND (nama ILIKE $2 OR email ILIKE $2 OR coalesce(identifier,'') ILIKE $2)`
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE `+where, tid, pg.Like()).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id FROM users WHERE `+where+` ORDER BY nama LIMIT $3 OFFSET $4`, tid, pg.Like(), pg.PerPage, pg.Offset())
	if err != nil {
		return nil, 0, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, 0, err
	}
	out := []User{}
	for _, id := range ids {
		u, err := s.repo.LoadUser(ctx, s.pool, id)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, nil
}

// --- tenants ---

type TenantWithStats struct {
	tenant.Tenant
	MahasiswaCount int    `json:"mahasiswa_count"`
	Plan           string `json:"plan"`
}

// ListTenantsWithStats is the super_admin cross-tenant view (no grade data).
func (s *Service) ListTenantsWithStats(ctx context.Context) ([]TenantWithStats, error) {
	rows, err := s.pool.Query(ctx, `SELECT t.id, t.kode, t.nama, t.timezone, t.plan,
		(SELECT count(*) FROM mahasiswa m WHERE m.tenant_id = t.id) FROM tenant t ORDER BY t.kode`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TenantWithStats{}
	for rows.Next() {
		var t TenantWithStats
		if err := rows.Scan(&t.ID, &t.Code, &t.Name, &t.Timezone, &t.Plan, &t.MahasiswaCount); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

type CreateTenantInput struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
	Plan     string `json:"plan"`
}

func (s *Service) CreateTenant(ctx context.Context, in CreateTenantInput) (TenantWithStats, error) {
	if in.Timezone == "" {
		in.Timezone = "Asia/Jakarta"
	}
	if in.Plan == "" {
		in.Plan = "basic"
	}
	v := httpx.NewV().Required("code", in.Code).Required("name", in.Name).
		OneOf("timezone", in.Timezone, "Asia/Jakarta", "Asia/Makassar", "Asia/Jayapura")
	if err := v.Err(); err != nil {
		return TenantWithStats{}, err
	}
	var out TenantWithStats
	err := s.pool.QueryRow(ctx, `INSERT INTO tenant (kode, nama, timezone, plan) VALUES (upper($1),$2,$3,$4)
		RETURNING id, kode, nama, timezone, plan`, in.Code, in.Name, in.Timezone, in.Plan).
		Scan(&out.ID, &out.Code, &out.Name, &out.Timezone, &out.Plan)
	if db.IsUniqueViolation(err) {
		return out, httpx.Conflict("DUPLICATE", "Kode tenant sudah dipakai")
	}
	return out, err
}
