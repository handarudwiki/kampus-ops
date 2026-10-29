package reporting

import (
	"github.com/gofiber/fiber/v2"

	"github.com/kampusops/backend/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Validate(c *fiber.Ctx) error {
	var in struct {
		PeriodeID string `json:"periode_id"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	r, err := h.svc.Validate(c.UserContext(), in.PeriodeID)
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) Latest(c *fiber.Ctx) error {
	r, err := h.svc.Latest(c.UserContext(), c.Query("periode_id"))
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) Export(c *fiber.Ctx) error {
	b, name, err := h.svc.Export(c.UserContext(), c.Params("entity"), c.Query("periode_id"))
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+name+`"`)
	return c.Send(b)
}
