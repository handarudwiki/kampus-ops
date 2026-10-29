package academic

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kampusops/backend/internal/audit"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/httpx"
	"github.com/kampusops/backend/internal/tenant"
)

// Handler serves all academic endpoints; business rules are in the Service.
type Handler struct {
	pool    *pgxpool.Pool
	tenants *tenant.Store
	svc     *Service
}

func NewHandler(pool *pgxpool.Pool, tenants *tenant.Store) *Handler {
	return &Handler{pool: pool, tenants: tenants, svc: NewService(pool, tenants)}
}

func (h *Handler) Service() *Service { return h.svc }

// deleteRow deletes a tenant-scoped row with audit; FK violations → 409 IN_USE.
func (h *Handler) deleteRow(c *fiber.Ctx, table, entitas string) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	ctx := c.UserContext()
	err = db.WithTx(ctx, h.pool, func(tx pgx.Tx) error {
		var before []byte
		err := tx.QueryRow(ctx, `DELETE FROM `+table+` t WHERE t.tenant_id=$1 AND t.id=$2 RETURNING to_jsonb(t)`, appctx.TenantID(ctx), id).Scan(&before)
		if err != nil {
			return err
		}
		return audit.Write(ctx, tx, entitas, id, "delete", before, nil)
	})
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23503" {
		return httpx.Conflict("IN_USE", "Data masih dipakai oleh data lain")
	}
	if err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func txAudit(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	return db.WithTx(ctx, pool, fn)
}

// ===== Fakultas =====

func (h *Handler) ListFakultas(c *fiber.Ctx) error {
	ctx := c.UserContext()
	pg := httpx.ParsePage(c)
	var total int
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM fakultas WHERE tenant_id=$1 AND (nama ILIKE $2 OR kode ILIKE $2)`, appctx.TenantID(ctx), pg.Like()).Scan(&total); err != nil {
		return err
	}
	rows, err := h.pool.Query(ctx, `SELECT id, kode, nama FROM fakultas WHERE tenant_id=$1 AND (nama ILIKE $2 OR kode ILIKE $2) ORDER BY kode LIMIT $3 OFFSET $4`,
		appctx.TenantID(ctx), pg.Like(), pg.PerPage, pg.Offset())
	if err != nil {
		return err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Fakultas])
	if err != nil {
		return err
	}
	return httpx.List(c, nonNil(out), pg, total)
}

func (h *Handler) getFakultas(ctx context.Context, id string) (Fakultas, error) {
	var f Fakultas
	err := h.pool.QueryRow(ctx, `SELECT id, kode, nama FROM fakultas WHERE tenant_id=$1 AND id=$2`, appctx.TenantID(ctx), id).Scan(&f.ID, &f.Kode, &f.Nama)
	return f, err
}

func (h *Handler) GetFakultas(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	f, err := h.getFakultas(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, f)
}

func (h *Handler) SaveFakultas(c *fiber.Ctx) error {
	ctx := c.UserContext()
	var in Fakultas
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := httpx.NewV().Required("kode", in.Kode).Required("nama", in.Nama).Err(); err != nil {
		return err
	}
	id := c.Params("id")
	var before any
	if id != "" {
		var err error
		if id, err = httpx.ParamID(c, "id"); err != nil {
			return err
		}
		b, err := h.getFakultas(ctx, id)
		if err != nil {
			return err
		}
		before = b
	}
	err := txAudit(ctx, h.pool, func(tx pgx.Tx) error {
		var err error
		if id == "" {
			err = tx.QueryRow(ctx, `INSERT INTO fakultas (tenant_id, kode, nama) VALUES ($1,$2,$3) RETURNING id`, appctx.TenantID(ctx), in.Kode, in.Nama).Scan(&id)
		} else {
			_, err = tx.Exec(ctx, `UPDATE fakultas SET kode=$3, nama=$4, updated_at=now() WHERE tenant_id=$1 AND id=$2`, appctx.TenantID(ctx), id, in.Kode, in.Nama)
		}
		if err != nil {
			return dup(err, "kode")
		}
		in.ID = id
		return audit.Write(ctx, tx, "fakultas", id, saveAksi(before), before, in)
	})
	if err != nil {
		return err
	}
	return respondSaved(c, before, in)
}

func (h *Handler) DeleteFakultas(c *fiber.Ctx) error { return h.deleteRow(c, "fakultas", "fakultas") }

// ===== Prodi =====

const prodiSel = `SELECT p.id, p.fakultas_id, f.nama, p.kode, p.nama, p.jenjang, p.masa_studi_maks FROM prodi p JOIN fakultas f ON f.id=p.fakultas_id `

func (h *Handler) ListProdi(c *fiber.Ctx) error {
	ctx := c.UserContext()
	pg := httpx.ParsePage(c)
	where := `WHERE p.tenant_id=$1 AND (p.nama ILIKE $2 OR p.kode ILIKE $2) AND ($3='' OR p.fakultas_id::text=$3)`
	args := []any{appctx.TenantID(ctx), pg.Like(), c.Query("fakultas_id")}
	var total int
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM prodi p `+where, args...).Scan(&total); err != nil {
		return err
	}
	rows, err := h.pool.Query(ctx, prodiSel+where+` ORDER BY p.kode LIMIT $4 OFFSET $5`, append(args, pg.PerPage, pg.Offset())...)
	if err != nil {
		return err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Prodi])
	if err != nil {
		return err
	}
	return httpx.List(c, nonNil(out), pg, total)
}

func (h *Handler) getProdi(ctx context.Context, id string) (Prodi, error) {
	rows, err := h.pool.Query(ctx, prodiSel+`WHERE p.tenant_id=$1 AND p.id=$2`, appctx.TenantID(ctx), id)
	if err != nil {
		return Prodi{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByPos[Prodi])
}

func (h *Handler) GetProdi(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	p, err := h.getProdi(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, p)
}

func (h *Handler) SaveProdi(c *fiber.Ctx) error {
	ctx := c.UserContext()
	var in Prodi
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if in.MasaStudiMaks == 0 {
		switch in.Jenjang {
		case "D3":
			in.MasaStudiMaks = 10
		case "S2":
			in.MasaStudiMaks = 8
		default:
			in.MasaStudiMaks = 14
		}
	}
	if err := httpx.NewV().Required("kode", in.Kode).Required("nama", in.Nama).OneOf("jenjang", in.Jenjang, "D3", "S1", "S2").
		Range("masa_studi_maks", in.MasaStudiMaks, 1, 20).Err(); err != nil {
		return err
	}
	if err := MustExist(ctx, h.pool, "fakultas", "fakultas_id", in.FakultasID); err != nil {
		return err
	}
	id := c.Params("id")
	var before any
	if id != "" {
		var err error
		if id, err = httpx.ParamID(c, "id"); err != nil {
			return err
		}
		if before, err = h.getProdi(ctx, id); err != nil {
			return err
		}
	}
	err := txAudit(ctx, h.pool, func(tx pgx.Tx) error {
		var err error
		if id == "" {
			err = tx.QueryRow(ctx, `INSERT INTO prodi (tenant_id, fakultas_id, kode, nama, jenjang, masa_studi_maks) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
				appctx.TenantID(ctx), in.FakultasID, in.Kode, in.Nama, in.Jenjang, in.MasaStudiMaks).Scan(&id)
		} else {
			_, err = tx.Exec(ctx, `UPDATE prodi SET fakultas_id=$3, kode=$4, nama=$5, jenjang=$6, masa_studi_maks=$7, updated_at=now() WHERE tenant_id=$1 AND id=$2`,
				appctx.TenantID(ctx), id, in.FakultasID, in.Kode, in.Nama, in.Jenjang, in.MasaStudiMaks)
		}
		if err != nil {
			return dup(err, "kode")
		}
		return audit.Write(ctx, tx, "prodi", id, saveAksi(before), before, in)
	})
	if err != nil {
		return err
	}
	p, err := h.getProdi(ctx, id)
	if err != nil {
		return err
	}
	return respondSaved(c, before, p)
}

func (h *Handler) DeleteProdi(c *fiber.Ctx) error { return h.deleteRow(c, "prodi", "prodi") }

// ===== Dosen =====

const dosenSel = `SELECT d.id, d.nidn, d.nama, d.email, d.prodi_id, p.nama FROM dosen d JOIN prodi p ON p.id=d.prodi_id `

func (h *Handler) ListDosen(c *fiber.Ctx) error {
	ctx := c.UserContext()
	pg := httpx.ParsePage(c)
	where := `WHERE d.tenant_id=$1 AND (d.nama ILIKE $2 OR d.nidn ILIKE $2) AND ($3='' OR d.prodi_id::text=$3)`
	args := []any{appctx.TenantID(ctx), pg.Like(), c.Query("prodi_id")}
	var total int
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM dosen d `+where, args...).Scan(&total); err != nil {
		return err
	}
	rows, err := h.pool.Query(ctx, dosenSel+where+` ORDER BY d.nama LIMIT $4 OFFSET $5`, append(args, pg.PerPage, pg.Offset())...)
	if err != nil {
		return err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Dosen])
	if err != nil {
		return err
	}
	return httpx.List(c, nonNil(out), pg, total)
}

func (h *Handler) getDosen(ctx context.Context, id string) (Dosen, error) {
	rows, err := h.pool.Query(ctx, dosenSel+`WHERE d.tenant_id=$1 AND d.id=$2`, appctx.TenantID(ctx), id)
	if err != nil {
		return Dosen{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByPos[Dosen])
}

func (h *Handler) GetDosen(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	d, err := h.getDosen(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, d)
}

// SaveDosen creates/updates a lecturer. On create a login account (identifier = NIDN,
// password = NIDN) with role dosen is created as well.
func (h *Handler) SaveDosen(c *fiber.Ctx) error {
	ctx := c.UserContext()
	var in Dosen
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := httpx.NewV().Required("nidn", in.NIDN).Required("nama", in.Nama).Email("email", in.Email).Err(); err != nil {
		return err
	}
	if err := MustExist(ctx, h.pool, "prodi", "prodi_id", in.ProdiID); err != nil {
		return err
	}
	id := c.Params("id")
	var before any
	if id != "" {
		var err error
		if id, err = httpx.ParamID(c, "id"); err != nil {
			return err
		}
		if before, err = h.getDosen(ctx, id); err != nil {
			return err
		}
	}
	err := txAudit(ctx, h.pool, func(tx pgx.Tx) error {
		var err error
		if id == "" {
			userID, err := h.svc.createAccount(ctx, tx, in.Nama, in.Email, in.NIDN, in.NIDN, "dosen")
			if err != nil {
				return err
			}
			err = tx.QueryRow(ctx, `INSERT INTO dosen (tenant_id, user_id, nidn, nama, email, prodi_id) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
				appctx.TenantID(ctx), userID, in.NIDN, in.Nama, in.Email, in.ProdiID).Scan(&id)
			if err != nil {
				return dup(err, "nidn")
			}
		} else {
			_, err = tx.Exec(ctx, `UPDATE dosen SET nidn=$3, nama=$4, email=$5, prodi_id=$6, updated_at=now() WHERE tenant_id=$1 AND id=$2`,
				appctx.TenantID(ctx), id, in.NIDN, in.Nama, in.Email, in.ProdiID)
			if err != nil {
				return dup(err, "nidn")
			}
		}
		return audit.Write(ctx, tx, "dosen", id, saveAksi(before), before, in)
	})
	if err != nil {
		return err
	}
	d, err := h.getDosen(ctx, id)
	if err != nil {
		return err
	}
	return respondSaved(c, before, d)
}

func (h *Handler) DeleteDosen(c *fiber.Ctx) error { return h.deleteRow(c, "dosen", "dosen") }

// ===== Ruang =====

func (h *Handler) ListRuang(c *fiber.Ctx) error {
	ctx := c.UserContext()
	pg := httpx.ParsePage(c)
	var total int
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM ruang WHERE tenant_id=$1 AND (nama ILIKE $2 OR kode ILIKE $2)`, appctx.TenantID(ctx), pg.Like()).Scan(&total); err != nil {
		return err
	}
	rows, err := h.pool.Query(ctx, `SELECT id, kode, nama, kapasitas FROM ruang WHERE tenant_id=$1 AND (nama ILIKE $2 OR kode ILIKE $2) ORDER BY kode LIMIT $3 OFFSET $4`,
		appctx.TenantID(ctx), pg.Like(), pg.PerPage, pg.Offset())
	if err != nil {
		return err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Ruang])
	if err != nil {
		return err
	}
	return httpx.List(c, nonNil(out), pg, total)
}

func (h *Handler) getRuang(ctx context.Context, id string) (Ruang, error) {
	var r Ruang
	err := h.pool.QueryRow(ctx, `SELECT id, kode, nama, kapasitas FROM ruang WHERE tenant_id=$1 AND id=$2`, appctx.TenantID(ctx), id).Scan(&r.ID, &r.Kode, &r.Nama, &r.Kapasitas)
	return r, err
}

func (h *Handler) GetRuang(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	r, err := h.getRuang(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) SaveRuang(c *fiber.Ctx) error {
	ctx := c.UserContext()
	var in Ruang
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := httpx.NewV().Required("kode", in.Kode).Required("nama", in.Nama).Range("kapasitas", in.Kapasitas, 1, 1000).Err(); err != nil {
		return err
	}
	id := c.Params("id")
	var before any
	if id != "" {
		var err error
		if id, err = httpx.ParamID(c, "id"); err != nil {
			return err
		}
		if before, err = h.getRuang(ctx, id); err != nil {
			return err
		}
	}
	err := txAudit(ctx, h.pool, func(tx pgx.Tx) error {
		var err error
		if id == "" {
			err = tx.QueryRow(ctx, `INSERT INTO ruang (tenant_id, kode, nama, kapasitas) VALUES ($1,$2,$3,$4) RETURNING id`, appctx.TenantID(ctx), in.Kode, in.Nama, in.Kapasitas).Scan(&id)
		} else {
			_, err = tx.Exec(ctx, `UPDATE ruang SET kode=$3, nama=$4, kapasitas=$5, updated_at=now() WHERE tenant_id=$1 AND id=$2`, appctx.TenantID(ctx), id, in.Kode, in.Nama, in.Kapasitas)
		}
		if err != nil {
			return dup(err, "kode")
		}
		in.ID = id
		return audit.Write(ctx, tx, "ruang", id, saveAksi(before), before, in)
	})
	if err != nil {
		return err
	}
	return respondSaved(c, before, in)
}

func (h *Handler) DeleteRuang(c *fiber.Ctx) error { return h.deleteRow(c, "ruang", "ruang") }

// ===== helpers =====

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func dup(err error, field string) error {
	if db.IsUniqueViolation(err) {
		return httpx.Conflict("DUPLICATE", field+" sudah dipakai").With(map[string]string{field: "sudah dipakai"})
	}
	return err
}

func saveAksi(before any) string {
	if before == nil {
		return "create"
	}
	return "update"
}

func respondSaved(c *fiber.Ctx, before, v any) error {
	if before == nil {
		return httpx.Created(c, v)
	}
	return httpx.OK(c, v)
}
