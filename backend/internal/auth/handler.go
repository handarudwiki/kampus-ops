package auth

import (
	"github.com/gofiber/fiber/v2"

	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/httpx"
	"github.com/kampusops/backend/internal/tenant"
)

type Handler struct {
	svc     *Service
	tenants *tenant.Store
}

func NewHandler(svc *Service, tenants *tenant.Store) *Handler { return &Handler{svc: svc, tenants: tenants} }

func (h *Handler) Login(c *fiber.Ctx) error {
	var in struct {
		TenantCode string `json:"tenant_code"`
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := httpx.NewV().Required("tenant_code", in.TenantCode).Required("identifier", in.Identifier).Required("password", in.Password).Err(); err != nil {
		return err
	}
	res, err := h.svc.Login(c.UserContext(), in.TenantCode, in.Identifier, in.Password)
	if err != nil {
		return err
	}
	return httpx.OK(c, res)
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(c *fiber.Ctx) error {
	var in refreshReq
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := httpx.NewV().Required("refresh_token", in.RefreshToken).Err(); err != nil {
		return err
	}
	res, err := h.svc.Refresh(c.UserContext(), in.RefreshToken)
	if err != nil {
		return err
	}
	return httpx.OK(c, res)
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	var in refreshReq
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if in.RefreshToken != "" {
		if err := h.svc.Logout(c.UserContext(), in.RefreshToken); err != nil {
			return err
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) Me(c *fiber.Ctx) error {
	u, t, err := h.svc.Me(c.UserContext())
	if err != nil {
		return err
	}
	return httpx.OK(c, fiber.Map{"user": u, "tenant": t})
}

func (h *Handler) PublicTenants(c *fiber.Ctx) error {
	ts, err := h.tenants.List(c.UserContext())
	if err != nil {
		return err
	}
	out := make([]tenant.Tenant, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.Tenant)
	}
	return httpx.OK(c, out)
}

func (h *Handler) ListTenants(c *fiber.Ctx) error {
	ts, err := h.svc.ListTenantsWithStats(c.UserContext())
	if err != nil {
		return err
	}
	return httpx.OK(c, ts)
}

func (h *Handler) CreateTenant(c *fiber.Ctx) error {
	var in CreateTenantInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	t, err := h.svc.CreateTenant(c.UserContext(), in)
	if err != nil {
		return err
	}
	return httpx.Created(c, t)
}

func (h *Handler) ListUsers(c *fiber.Ctx) error {
	pg := httpx.ParsePage(c)
	us, total, err := h.svc.ListUsers(c.UserContext(), pg)
	if err != nil {
		return err
	}
	return httpx.List(c, us, pg, total)
}

func (h *Handler) CreateUser(c *fiber.Ctx) error {
	var in CreateUserInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	u, err := h.svc.CreateUser(c.UserContext(), in)
	if err != nil {
		return err
	}
	return httpx.Created(c, u)
}

func (h *Handler) SetRoles(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	var in struct {
		Roles []appctx.UserRole `json:"roles"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	u, err := h.svc.SetRoles(c.UserContext(), id, in.Roles)
	if err != nil {
		return err
	}
	return httpx.OK(c, u)
}
