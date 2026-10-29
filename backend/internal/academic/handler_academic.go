package academic

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/httpx"
)

// ===== Periode =====

func (h *Handler) ListPeriode(c *fiber.Ctx) error {
	ctx := c.UserContext()
	pg := httpx.ParsePage(c)
	where := ` WHERE p.tenant_id=$1 AND (p.kode ILIKE $2) AND ($3='' OR p.status=$3)`
	args := []any{appctx.TenantID(ctx), pg.Like(), c.Query("status")}
	var total int
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM periode p`+where, args...).Scan(&total); err != nil {
		return err
	}
	rows, err := h.pool.Query(ctx, `SELECT `+periodeCols+` FROM periode p`+where+` ORDER BY p.kode DESC LIMIT $4 OFFSET $5`, append(args, pg.PerPage, pg.Offset())...)
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []Periode{}
	for rows.Next() {
		p, err := scanPeriode(rows)
		if err != nil {
			return err
		}
		out = append(out, p)
	}
	return httpx.List(c, out, pg, total)
}

func (h *Handler) ActivePeriode(c *fiber.Ctx) error {
	p, err := ActivePeriode(c.UserContext(), h.pool)
	if err != nil {
		return err
	}
	return httpx.OK(c, p)
}

func (h *Handler) GetPeriode(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	p, err := GetPeriode(c.UserContext(), h.pool, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, p)
}

func (h *Handler) SavePeriode(c *fiber.Ctx) error {
	var in PeriodeInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	id := ""
	if c.Params("id") != "" {
		var err error
		if id, err = httpx.ParamID(c, "id"); err != nil {
			return err
		}
	}
	p, err := h.svc.SavePeriode(c.UserContext(), id, in)
	if err != nil {
		return err
	}
	if id == "" {
		return httpx.Created(c, p)
	}
	return httpx.OK(c, p)
}

func (h *Handler) DeletePeriode(c *fiber.Ctx) error { return h.deleteRow(c, "periode", "periode") }

func (h *Handler) ActivatePeriode(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	p, err := h.svc.ActivatePeriode(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, p)
}

// ===== Mahasiswa =====

func (h *Handler) ListMahasiswa(c *fiber.Ctx) error {
	ctx := c.UserContext()
	pg := httpx.ParsePage(c)
	ang, _ := strconv.Atoi(c.Query("angkatan"))
	where := `(m.nama ILIKE $2 OR m.nim ILIKE $2) AND ($3='' OR m.prodi_id::text=$3) AND ($4=0 OR m.angkatan=$4) AND ($5='' OR m.status=$5)
		AND ($6='' OR m.dosen_wali_id::text=$6)`
	args := []any{pg.Like(), c.Query("prodi_id"), ang, c.Query("status"), c.Query("dosen_wali_id")}
	var total int
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM mahasiswa m WHERE m.tenant_id=$1 AND `+where, append([]any{appctx.TenantID(ctx)}, args...)...).Scan(&total); err != nil {
		return err
	}
	ms, err := QueryMahasiswa(ctx, h.pool, where, `ORDER BY m.nim LIMIT $7 OFFSET $8`, append(args, pg.PerPage, pg.Offset())...)
	if err != nil {
		return err
	}
	return httpx.List(c, ms, pg, total)
}

func (h *Handler) GetMahasiswa(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	m, err := GetMahasiswa(c.UserContext(), h.pool, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, m)
}

func (h *Handler) SaveMahasiswa(c *fiber.Ctx) error {
	var in MahasiswaInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	id := ""
	if c.Params("id") != "" {
		var err error
		if id, err = httpx.ParamID(c, "id"); err != nil {
			return err
		}
	}
	m, err := h.svc.SaveMahasiswa(c.UserContext(), id, in)
	if err != nil {
		return err
	}
	if id == "" {
		return httpx.Created(c, m)
	}
	return httpx.OK(c, m)
}

func (h *Handler) DeleteMahasiswa(c *fiber.Ctx) error { return h.deleteRow(c, "mahasiswa", "mahasiswa") }

func (h *Handler) RiwayatStatus(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	ctx := c.UserContext()
	if _, err := GetMahasiswa(ctx, h.pool, id); err != nil {
		return err
	}
	r, err := Riwayat(ctx, h.pool, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) ChangeStatus(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	var in struct {
		Status string `json:"status"`
		Alasan string `json:"alasan"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	r, err := h.svc.ChangeStatus(c.UserContext(), id, in.Status, in.Alasan)
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

// ===== Kurikulum =====

func (h *Handler) ListKurikulum(c *fiber.Ctx) error {
	ks, err := h.svc.ListKurikulum(c.UserContext(), c.Query("prodi_id"))
	if err != nil {
		return err
	}
	return httpx.OK(c, ks)
}

func (h *Handler) ListMataKuliah(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	ms, err := h.svc.ListMataKuliah(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, ms)
}

func (h *Handler) SaveMataKuliah(c *fiber.Ctx) error {
	var in MataKuliahInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	id := ""
	if c.Params("id") != "" {
		var err error
		if id, err = httpx.ParamID(c, "id"); err != nil {
			return err
		}
	}
	m, err := h.svc.SaveMataKuliah(c.UserContext(), id, in)
	if err != nil {
		return err
	}
	if id == "" {
		return httpx.Created(c, m)
	}
	return httpx.OK(c, m)
}

func (h *Handler) ListEkuivalensi(c *fiber.Ctx) error {
	es, err := h.svc.ListEkuivalensi(c.UserContext(), c.Query("kurikulum_id"))
	if err != nil {
		return err
	}
	return httpx.OK(c, es)
}

func (h *Handler) CreateEkuivalensi(c *fiber.Ctx) error {
	var in struct {
		MkLamaID string `json:"mk_lama_id"`
		MkBaruID string `json:"mk_baru_id"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	e, err := h.svc.CreateEkuivalensi(c.UserContext(), in.MkLamaID, in.MkBaruID)
	if err != nil {
		return err
	}
	return httpx.Created(c, e)
}

// ===== Kelas =====

func (h *Handler) ListKelas(c *fiber.Ctx) error {
	ctx := c.UserContext()
	periode := c.Query("periode_id")
	if periode == "" {
		if ap, err := ActivePeriode(ctx, h.pool); err == nil {
			periode = ap.ID
		}
	}
	q := httpx.ParsePage(c)
	ks, err := QueryKelas(ctx, h.pool, `k.periode_id::text=$2 AND ($3='' OR mk.kurikulum_id IN (SELECT id FROM kurikulum WHERE prodi_id::text=$3))
		AND (mk.nama ILIKE $4 OR mk.kode ILIKE $4 OR coalesce(d.nama,'') ILIKE $4) AND ($5='' OR k.status=$5)
		ORDER BY mk.kode, k.nama`, periode, c.Query("prodi_id"), q.Like(), c.Query("status"))
	if err != nil {
		return err
	}
	return httpx.OK(c, ks)
}

func (h *Handler) GetKelas(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	k, err := GetKelas(c.UserContext(), h.pool, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, k)
}

func (h *Handler) SaveKelas(c *fiber.Ctx) error {
	var in KelasInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	id := ""
	if c.Params("id") != "" {
		var err error
		if id, err = httpx.ParamID(c, "id"); err != nil {
			return err
		}
	}
	k, err := h.svc.SaveKelas(c.UserContext(), id, in)
	if err != nil {
		return err
	}
	if id == "" {
		return httpx.Created(c, k)
	}
	return httpx.OK(c, k)
}

func (h *Handler) DeleteKelas(c *fiber.Ctx) error { return h.deleteRow(c, "kelas", "kelas") }

func (h *Handler) CancelKelas(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	var in struct {
		Alasan string `json:"alasan"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	k, err := h.svc.CancelKelas(c.UserContext(), id, in.Alasan)
	if err != nil {
		return err
	}
	return httpx.OK(c, k)
}

func (h *Handler) GantiDosen(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	var in struct {
		DosenID string `json:"dosen_id"`
		Efektif string `json:"efektif"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	k, err := h.svc.GantiDosen(c.UserContext(), id, in.DosenID, in.Efektif)
	if err != nil {
		return err
	}
	return httpx.OK(c, k)
}

// ===== Akademik — mahasiswa =====

func (h *Handler) MyJadwal(c *fiber.Ctx) error {
	j, err := h.svc.Jadwal(c.UserContext())
	if err != nil {
		return err
	}
	return httpx.OK(c, j)
}

func (h *Handler) MyKHS(c *fiber.Ctx) error {
	pid := c.Query("periode_id")
	if pid != "" {
		if _, err := httpx.OptUUID("periode_id", pid); err != nil {
			return err
		}
	}
	k, err := h.svc.KHS(c.UserContext(), pid)
	if err != nil {
		return err
	}
	return httpx.OK(c, k)
}

func (h *Handler) MyPeriodes(c *fiber.Ctx) error {
	p, err := h.svc.Periodes(c.UserContext())
	if err != nil {
		return err
	}
	return httpx.OK(c, p)
}

func (h *Handler) MyTranskrip(c *fiber.Ctx) error {
	t, err := h.svc.Transkrip(c.UserContext())
	if err != nil {
		return err
	}
	return httpx.OK(c, t)
}

func (h *Handler) MyProfil(c *fiber.Ctx) error {
	p, err := h.svc.Profil(c.UserContext())
	if err != nil {
		return err
	}
	return httpx.OK(c, p)
}

func (h *Handler) MasaStudiWarning(c *fiber.Ctx) error {
	ctx := c.UserContext()
	p, _ := appctx.PrincipalFrom(ctx)
	var scope *string
	if p.ActiveRole == "kaprodi" {
		scope = p.ScopeProdi("kaprodi")
	}
	if s := c.Query("prodi_id"); s != "" && scope == nil {
		scope = &s
	}
	rows, err := h.svc.MasaStudiWarnings(ctx, scope)
	if err != nil {
		return err
	}
	return httpx.OK(c, rows)
}
