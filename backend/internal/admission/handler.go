package admission

import (
	"github.com/gofiber/fiber/v2"

	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ListGelombang(c *fiber.Ctx) error {
	_, authed := appctx.PrincipalFrom(c.UserContext())
	gs, err := h.svc.ListGelombang(c.UserContext(), !authed || c.Query("open") == "true")
	if err != nil {
		return err
	}
	return httpx.OK(c, gs)
}

func (h *Handler) CreateGelombang(c *fiber.Ctx) error {
	var in Gelombang
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	g, err := h.svc.CreateGelombang(c.UserContext(), in)
	if err != nil {
		return err
	}
	return httpx.Created(c, g)
}

func (h *Handler) PublicOptions(c *fiber.Ctx) error {
	code := c.Query("tenant_code")
	if code == "" {
		return httpx.ValidationField("tenant_code", "wajib diisi")
	}
	r, err := h.svc.PublicOptions(c.UserContext(), code)
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) ListPendaftar(c *fiber.Ctx) error {
	pg := httpx.ParsePage(c)
	ps, total, err := h.svc.ListPendaftar(c.UserContext(), c.Query("status"), c.Query("gelombang_id"), pg)
	if err != nil {
		return err
	}
	return httpx.List(c, ps, pg, total)
}

func (h *Handler) Register(c *fiber.Ctx) error {
	var in RegisterInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	p, err := h.svc.Register(c.UserContext(), in)
	if err != nil {
		return err
	}
	return httpx.Created(c, p)
}

func (h *Handler) Me(c *fiber.Ctx) error {
	p, err := h.svc.Me(c.UserContext())
	if err != nil {
		return err
	}
	return httpx.OK(c, p)
}

func (h *Handler) Transition(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	var in struct {
		To string `json:"to"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := httpx.NewV().OneOf("to", in.To, Daftar, BayarFormulir, LengkapiBerkas, Verifikasi, Lulus, TidakLulus, DaftarUlang).Err(); err != nil {
		return err
	}
	p, err := h.svc.Transition(c.UserContext(), id, in.To)
	if err != nil {
		return err
	}
	return httpx.OK(c, p)
}

func (h *Handler) DaftarUlang(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	p, err := h.svc.DaftarUlang(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, p)
}
