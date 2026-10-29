// Package notification stores in-app notifications (FR-33).
package notification

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/httpx"
)

// Send inserts a notification for userID (no-op when userID is empty).
func Send(ctx context.Context, q db.Querier, userID, judul, isi string, link *string) error {
	if userID == "" {
		return nil
	}
	_, err := q.Exec(ctx, `INSERT INTO notifikasi (tenant_id, user_id, judul, isi, link) VALUES ($1,$2,$3,$4,$5)`,
		appctx.TenantID(ctx), userID, judul, isi, link)
	return err
}

// Link is a helper for optional links.
func Link(s string) *string { return &s }

type Notifikasi struct {
	ID        string  `json:"id"`
	Judul     string  `json:"judul"`
	Isi       string  `json:"isi"`
	Link      *string `json:"link"`
	Read      bool    `json:"read"`
	CreatedAt string  `json:"created_at"`
}

type Handler struct{ pool *pgxpool.Pool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

// List: GET /notifikasi (own notifications, newest first).
func (h *Handler) List(c *fiber.Ctx) error {
	ctx := c.UserContext()
	pg := httpx.ParsePage(c)
	var total int
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM notifikasi WHERE tenant_id=$1 AND user_id=$2`,
		appctx.TenantID(ctx), appctx.UserID(ctx)).Scan(&total); err != nil {
		return err
	}
	rows, err := h.pool.Query(ctx, `SELECT id, judul, isi, link, read_at IS NOT NULL, iso(created_at)
		FROM notifikasi WHERE tenant_id=$1 AND user_id=$2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		appctx.TenantID(ctx), appctx.UserID(ctx), pg.PerPage, pg.Offset())
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []Notifikasi{}
	for rows.Next() {
		var n Notifikasi
		if err := rows.Scan(&n.ID, &n.Judul, &n.Isi, &n.Link, &n.Read, &n.CreatedAt); err != nil {
			return err
		}
		out = append(out, n)
	}
	return httpx.List(c, out, pg, total)
}

// Read: POST /notifikasi/:id/read
func (h *Handler) Read(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	ctx := c.UserContext()
	var n Notifikasi
	err = h.pool.QueryRow(ctx, `UPDATE notifikasi SET read_at = coalesce(read_at, now()), updated_at = now()
		WHERE id=$1 AND tenant_id=$2 AND user_id=$3 RETURNING id, judul, isi, link, true, iso(created_at)`,
		id, appctx.TenantID(ctx), appctx.UserID(ctx)).Scan(&n.ID, &n.Judul, &n.Isi, &n.Link, &n.Read, &n.CreatedAt)
	if err != nil {
		return err
	}
	return httpx.OK(c, n)
}
