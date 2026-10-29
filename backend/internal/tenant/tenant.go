// Package tenant loads tenants (with their rules config and timezone) and caches them
// in-process; tenant rows are the only rows not keyed by tenant_id.
package tenant

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kampusops/backend/internal/domain"
	"github.com/kampusops/backend/internal/platform/appctx"
)

// Tenant is the contract's Tenant shape.
type Tenant struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
}

// Full is a tenant with parsed rules and location.
type Full struct {
	Tenant
	Plan  string
	Rules domain.Rules
	Loc   *time.Location
}

func (f Full) Info() appctx.TenantInfo {
	return appctx.TenantInfo{ID: f.ID, Code: f.Code, Name: f.Name, Timezone: f.Timezone, Loc: f.Loc}
}

type Store struct {
	pool *pgxpool.Pool
	mu   sync.RWMutex
	byID map[string]cached
	ttl  time.Duration
}

type cached struct {
	t   Full
	exp time.Time
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, byID: map[string]cached{}, ttl: time.Minute}
}

const sel = `SELECT id, kode, nama, timezone, plan, konfigurasi_aturan FROM tenant `

func scan(row interface{ Scan(...any) error }) (Full, error) {
	var f Full
	var raw []byte
	if err := row.Scan(&f.ID, &f.Code, &f.Name, &f.Timezone, &f.Plan, &raw); err != nil {
		return f, err
	}
	f.Rules = domain.ParseRules(raw)
	f.Loc = domain.LoadLocation(f.Timezone)
	return f, nil
}

// Get returns a tenant by id (cached ~1 minute).
func (s *Store) Get(ctx context.Context, id string) (Full, error) {
	s.mu.RLock()
	c, ok := s.byID[id]
	s.mu.RUnlock()
	if ok && time.Now().Before(c.exp) {
		return c.t, nil
	}
	f, err := scan(s.pool.QueryRow(ctx, sel+`WHERE id = $1`, id))
	if err != nil {
		return f, err
	}
	s.mu.Lock()
	s.byID[id] = cached{t: f, exp: time.Now().Add(s.ttl)}
	s.mu.Unlock()
	return f, nil
}

// ByCode returns a tenant by its code (case-insensitive).
func (s *Store) ByCode(ctx context.Context, code string) (Full, error) {
	return scan(s.pool.QueryRow(ctx, sel+`WHERE upper(kode) = upper($1)`, code))
}

// Current returns the tenant of the request context.
func (s *Store) Current(ctx context.Context) (Full, error) { return s.Get(ctx, appctx.TenantID(ctx)) }

// Invalidate drops a cached tenant.
func (s *Store) Invalidate(id string) {
	s.mu.Lock()
	delete(s.byID, id)
	s.mu.Unlock()
}

// List returns all tenants (public picker / super admin).
func (s *Store) List(ctx context.Context) ([]Full, error) {
	rows, err := s.pool.Query(ctx, sel+`ORDER BY kode`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Full
	for rows.Next() {
		f, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
