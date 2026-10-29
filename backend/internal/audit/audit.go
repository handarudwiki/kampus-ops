// Package audit writes the append-only audit_log (FR-32) inside the caller's
// transaction and exposes the read endpoint.
package audit

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/httpx"
)

// Write appends an audit entry using q (pass the tx so the entry commits atomically
// with the mutation). before/after are marshalled to JSONB (nil → NULL).
func Write(ctx context.Context, q db.Querier, entitas, entitasID, aksi string, before, after any) error {
	p, ok := appctx.PrincipalFrom(ctx)
	nama, role := "system", "system"
	var actor *string
	if ok {
		nama, role = p.Name, p.ActiveRole
		uid := p.UserID
		actor = &uid
	}
	_, err := q.Exec(ctx, `INSERT INTO audit_log (tenant_id, actor_id, actor_nama, actor_role, entitas, entitas_id, aksi, before, after, request_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		appctx.TenantID(ctx), actor, nama, role, entitas, entitasID, aksi, jsonOrNil(before), jsonOrNil(after), appctx.RequestID(ctx))
	return err
}

func jsonOrNil(v any) []byte {
	if v == nil {
		return nil
	}
	if b, ok := v.([]byte); ok {
		return b
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

// Entry is the contract's AuditEntry.
type Entry struct {
	ID        string          `json:"id"`
	ActorNama string          `json:"actor_nama"`
	ActorRole string          `json:"actor_role"`
	Entitas   string          `json:"entitas"`
	EntitasID string          `json:"entitas_id"`
	Aksi      string          `json:"aksi"`
	Before    json.RawMessage `json:"before"`
	After     json.RawMessage `json:"after"`
	CreatedAt string          `json:"created_at"`
	RequestID string          `json:"request_id"`
}

type Handler struct{ pool *pgxpool.Pool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

// List: GET /audit-log?entitas=&actor=&q=&entitas_id=
func (h *Handler) List(c *fiber.Ctx) error {
	ctx := c.UserContext()
	pg := httpx.ParsePage(c)
	where := `tenant_id = $1 AND ($2 = '' OR entitas = $2) AND ($3 = '' OR actor_nama ILIKE '%'||$3||'%')
		AND ($4 = '' OR aksi ILIKE '%'||$4||'%' OR entitas ILIKE '%'||$4||'%' OR entitas_id::text = $4)
		AND ($5 = '' OR entitas_id::text = $5)`
	args := []any{appctx.TenantID(ctx), c.Query("entitas"), c.Query("actor"), pg.Q, c.Query("entitas_id")}
	var total int
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE `+where, args...).Scan(&total); err != nil {
		return err
	}
	rows, err := h.pool.Query(ctx, `SELECT id, actor_nama, actor_role, entitas, entitas_id, aksi,
		coalesce(before,'null'::jsonb), coalesce(after,'null'::jsonb), iso(created_at), request_id
		FROM audit_log WHERE `+where+` ORDER BY created_at DESC LIMIT $6 OFFSET $7`, append(args, pg.PerPage, pg.Offset())...)
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		var e Entry
		var b, a []byte
		if err := rows.Scan(&e.ID, &e.ActorNama, &e.ActorRole, &e.Entitas, &e.EntitasID, &e.Aksi, &b, &a, &e.CreatedAt, &e.RequestID); err != nil {
			return err
		}
		e.Before, e.After = b, a
		out = append(out, e)
	}
	return httpx.List(c, out, pg, total)
}
