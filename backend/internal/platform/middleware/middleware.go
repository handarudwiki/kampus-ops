// Package middleware contains Fiber middleware: request id, structured logging,
// metrics, authentication, role checks and idempotency keys (EC-02).
package middleware

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/httpx"
	"github.com/kampusops/backend/internal/platform/token"
	"github.com/kampusops/backend/internal/tenant"
)

var (
	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: []float64{.005, .01, .025, .05, .1, .2, .3, .5, 1, 2, 5},
	}, []string{"method", "route", "status"})
	httpErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_errors_total", Help: "HTTP responses with status >= 400",
	}, []string{"route", "status"})
)

func init() { prometheus.MustRegister(httpDuration, httpErrors) }

// RequestID assigns X-Request-ID and stores it in the request context.
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Get("X-Request-ID")
		if id == "" || len(id) > 100 {
			id = uuid.NewString()
		}
		c.Locals("request_id", id)
		c.Set("X-Request-ID", id)
		c.SetUserContext(appctx.WithRequestID(c.UserContext(), id))
		return c.Next()
	}
}

// Logger writes one JSON line per request with request_id and tenant_id, and records metrics.
func Logger(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		if err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				status = fe.Code
			} else if ae, ok := err.(*httpx.AppError); ok {
				status = ae.Status
			}
		}
		lat := time.Since(start)
		route := c.Route().Path
		st := strconv.Itoa(status)
		httpDuration.WithLabelValues(c.Method(), route, st).Observe(lat.Seconds())
		if status >= 400 {
			httpErrors.WithLabelValues(route, st).Inc()
		}
		ctx := c.UserContext()
		p, _ := appctx.PrincipalFrom(ctx)
		lvl := slog.LevelInfo
		if status >= 500 {
			lvl = slog.LevelError
		}
		log.Log(ctx, lvl, "http_request",
			"request_id", appctx.RequestID(ctx),
			"tenant_id", p.TenantID,
			"user_id", p.UserID,
			"role", p.ActiveRole,
			"method", c.Method(),
			"path", c.Path(),
			"route", route,
			"status", status,
			"latency_ms", float64(lat.Microseconds())/1000,
		)
		return err
	}
}

// Auth validates the bearer token, resolves the active role (X-Active-Role) and tenant.
func Auth(iss *token.Issuer, tenants *tenant.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		h := c.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			return httpx.Unauthorized("UNAUTHORIZED", "Token tidak ditemukan")
		}
		claims, err := iss.Parse(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			return httpx.Unauthorized("AUTH_TOKEN_EXPIRED", "Token kedaluwarsa atau tidak valid")
		}
		p := appctx.Principal{UserID: claims.Subject, TenantID: claims.TenantID, Name: claims.Name, Roles: claims.Roles}
		active := strings.TrimSpace(c.Get("X-Active-Role"))
		if active == "" {
			if len(p.Roles) > 0 {
				active = p.Roles[0].Role
			}
		} else if !p.HasRole(active) {
			return httpx.Forbidden("FORBIDDEN_ROLE", "Anda tidak memiliki peran "+active)
		}
		p.ActiveRole = active
		t, err := tenants.Get(c.UserContext(), p.TenantID)
		if err != nil {
			return httpx.Unauthorized("AUTH_TOKEN_EXPIRED", "Tenant tidak ditemukan")
		}
		ctx := appctx.WithPrincipal(c.UserContext(), p)
		ctx = appctx.WithTenant(ctx, t.Info())
		c.SetUserContext(ctx)
		c.Locals("principal", p)
		return c.Next()
	}
}

// OptionalAuth runs Auth when a bearer token is present, otherwise resolves the tenant
// from ?tenant_code= (public PMB pages). Without either the request is rejected.
func OptionalAuth(iss *token.Issuer, tenants *tenant.Store) fiber.Handler {
	auth := Auth(iss, tenants)
	return func(c *fiber.Ctx) error {
		if c.Get("Authorization") != "" {
			return auth(c)
		}
		code := c.Query("tenant_code")
		if code == "" {
			return httpx.Unauthorized("UNAUTHORIZED", "Token tidak ditemukan")
		}
		t, err := tenants.ByCode(c.UserContext(), code)
		if err != nil {
			return httpx.NotFound("Kampus tidak ditemukan")
		}
		c.SetUserContext(appctx.WithTenant(c.UserContext(), t.Info()))
		return c.Next()
	}
}

// Roles allows the request only when the active role is one of roles.
func Roles(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		p, ok := appctx.PrincipalFrom(c.UserContext())
		if !ok {
			// public (OptionalAuth) access: role-gated routes are not available
			return httpx.Unauthorized("UNAUTHORIZED", "Token tidak ditemukan")
		}
		for _, r := range roles {
			if p.ActiveRole == r {
				return c.Next()
			}
		}
		return httpx.Forbidden("FORBIDDEN_ROLE", "Peran aktif "+p.ActiveRole+" tidak diizinkan")
	}
}
