package lecture

import (
	"github.com/gofiber/fiber/v2"

	"github.com/kampusops/backend/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Teaching(c *fiber.Ctx) error {
	if _, err := httpx.OptUUID("periode_id", c.Query("periode_id")); err != nil {
		return err
	}
	r, err := h.svc.TeachingKelas(c.UserContext(), c.Query("periode_id"))
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func withID(c *fiber.Ctx, name string, fn func(id string) (any, error)) error {
	id, err := httpx.ParamID(c, name)
	if err != nil {
		return err
	}
	r, err := fn(id)
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) Peserta(c *fiber.Ctx) error {
	return withID(c, "id", func(id string) (any, error) { return h.svc.Peserta(c.UserContext(), id) })
}

func (h *Handler) ListPertemuan(c *fiber.Ctx) error {
	return withID(c, "id", func(id string) (any, error) { return h.svc.ListPertemuan(c.UserContext(), id) })
}

func (h *Handler) CreatePertemuan(c *fiber.Ctx) error {
	var in struct {
		Tanggal string `json:"tanggal"`
		Topik   string `json:"topik"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	p, err := h.svc.CreatePertemuan(c.UserContext(), id, in.Tanggal, in.Topik)
	if err != nil {
		return err
	}
	return httpx.Created(c, p)
}

func (h *Handler) Presensi(c *fiber.Ctx) error {
	return withID(c, "id", func(id string) (any, error) { return h.svc.Presensi(c.UserContext(), id) })
}

func (h *Handler) SavePresensi(c *fiber.Ctx) error {
	var in struct {
		Items []PresensiItem `json:"items"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	return withID(c, "id", func(id string) (any, error) { return h.svc.SavePresensi(c.UserContext(), id, in.Items) })
}

func (h *Handler) GetNilai(c *fiber.Ctx) error {
	return withID(c, "id", func(id string) (any, error) { return h.svc.GetNilai(c.UserContext(), id) })
}

func (h *Handler) SaveNilai(c *fiber.Ctx) error {
	var in struct {
		Rows []NilaiInput `json:"rows"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	return withID(c, "id", func(id string) (any, error) { return h.svc.SaveNilai(c.UserContext(), id, in.Rows) })
}

func (h *Handler) Finalize(c *fiber.Ctx) error {
	return withID(c, "id", func(id string) (any, error) { return h.svc.Finalize(c.UserContext(), id) })
}

func (h *Handler) RequestKoreksi(c *fiber.Ctx) error {
	var in struct {
		Komponen map[string]*float64 `json:"komponen"`
		Alasan   string              `json:"alasan"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	k, err := h.svc.RequestKoreksi(c.UserContext(), id, in.Komponen, in.Alasan)
	if err != nil {
		return err
	}
	return httpx.Created(c, k)
}

func (h *Handler) ListKoreksi(c *fiber.Ctx) error {
	r, err := h.svc.ListKoreksi(c.UserContext(), c.Query("status"))
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) decide(c *fiber.Ctx, approve bool) error {
	var in struct {
		Catatan string `json:"catatan"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	return withID(c, "id", func(id string) (any, error) { return h.svc.DecideKoreksi(c.UserContext(), id, approve, in.Catatan) })
}

func (h *Handler) ApproveKoreksi(c *fiber.Ctx) error { return h.decide(c, true) }
func (h *Handler) RejectKoreksi(c *fiber.Ctx) error  { return h.decide(c, false) }
