package krs

import (
	"github.com/gofiber/fiber/v2"

	"github.com/kampusops/backend/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Mine(c *fiber.Ctx) error {
	r, err := h.svc.GetMine(c.UserContext())
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) Offerings(c *fiber.Ctx) error {
	r, err := h.svc.Offerings(c.UserContext(), c.Query("q"))
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) AddItem(c *fiber.Ctx) error {
	var in struct {
		KelasID string `json:"kelas_id"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := httpx.NewV().UUID("kelas_id", in.KelasID).Err(); err != nil {
		return err
	}
	r, err := h.svc.AddItem(c.UserContext(), in.KelasID)
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) RemoveItem(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "kelas_id")
	if err != nil {
		return err
	}
	r, err := h.svc.RemoveItem(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) Submit(c *fiber.Ctx) error {
	r, err := h.svc.Submit(c.UserContext())
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) Advisees(c *fiber.Ctx) error {
	if _, err := httpx.OptUUID("periode_id", c.Query("periode_id")); err != nil {
		return err
	}
	r, err := h.svc.Advisees(c.UserContext(), c.Query("status"), c.Query("periode_id"))
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) Get(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	r, err := h.svc.Get(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) decide(c *fiber.Ctx, approve bool) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	var in struct {
		Catatan string `json:"catatan"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	r, err := h.svc.Decide(c.UserContext(), id, approve, in.Catatan)
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) Approve(c *fiber.Ctx) error { return h.decide(c, true) }
func (h *Handler) Reject(c *fiber.Ctx) error  { return h.decide(c, false) }
