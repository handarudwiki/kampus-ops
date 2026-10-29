// Package appctx carries request-scoped identity (tenant, user, active role,
// request id) through context.Context so repositories can always filter by tenant.
package appctx

import (
	"context"
	"time"
)

type ctxKey int

const (
	principalKey ctxKey = iota
	requestIDKey
	tenantKey
)

// UserRole mirrors the API contract's UserRole.
type UserRole struct {
	Role           string  `json:"role"`
	ScopeProdiID   *string `json:"scope_prodi_id"`
	ScopeProdiName *string `json:"scope_prodi_name"`
}

// Principal is the authenticated caller.
type Principal struct {
	UserID     string
	TenantID   string
	Name       string
	Roles      []UserRole
	ActiveRole string
}

// HasRole reports whether the principal holds role r.
func (p Principal) HasRole(r string) bool {
	for _, x := range p.Roles {
		if x.Role == r {
			return true
		}
	}
	return false
}

// ScopeProdi returns the prodi scope of the given role, if any.
func (p Principal) ScopeProdi(role string) *string {
	for _, x := range p.Roles {
		if x.Role == role {
			return x.ScopeProdiID
		}
	}
	return nil
}

// TenantInfo is the minimal tenant info the request needs (timezone for EC-17).
type TenantInfo struct {
	ID       string
	Code     string
	Name     string
	Timezone string
	Loc      *time.Location
}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

// TenantID returns the tenant from context. Every repository query uses it; an
// empty value matches no rows, so a missing tenant can never leak data.
func TenantID(ctx context.Context) string {
	if t, ok := ctx.Value(tenantKey).(TenantInfo); ok {
		return t.ID
	}
	if p, ok := PrincipalFrom(ctx); ok {
		return p.TenantID
	}
	return "00000000-0000-0000-0000-000000000000"
}

func WithTenant(ctx context.Context, t TenantInfo) context.Context {
	return context.WithValue(ctx, tenantKey, t)
}

func Tenant(ctx context.Context) TenantInfo {
	t, _ := ctx.Value(tenantKey).(TenantInfo)
	if t.Loc == nil {
		t.Loc = time.UTC
	}
	return t
}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestID(ctx context.Context) string {
	s, _ := ctx.Value(requestIDKey).(string)
	return s
}

// UserID returns the caller's user id or "".
func UserID(ctx context.Context) string {
	p, _ := PrincipalFrom(ctx)
	return p.UserID
}
