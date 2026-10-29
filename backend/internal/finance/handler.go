package finance

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/kampusops/backend/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(c *fiber.Ctx) error {
	f := ListFilter{Status: c.Query("status"), PeriodeID: c.Query("periode_id"), ProdiID: c.Query("prodi_id"), Q: c.Query("q"), Jenis: c.Query("jenis")}
	pg := httpx.ParsePage(c)
	limit, offset := 2000, 0 // unpaged by default (contract), paged when ?page= is given
	if c.Query("page") != "" || c.Query("per_page") != "" {
		limit, offset = pg.PerPage, pg.Offset()
	} else {
		pg.PerPage = limit
	}
	ts, total, err := h.svc.List(c.UserContext(), f, limit, offset)
	if err != nil {
		return err
	}
	return httpx.List(c, ts, pg, total)
}

func (h *Handler) Mine(c *fiber.Ctx) error {
	ts, err := h.svc.Mine(c.UserContext())
	if err != nil {
		return err
	}
	return httpx.OK(c, ts)
}

func (h *Handler) Generate(c *fiber.Ctx) error {
	var in struct {
		PeriodeID string `json:"periode_id"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	n, err := h.svc.Generate(c.UserContext(), in.PeriodeID)
	if err != nil {
		return err
	}
	return httpx.OK(c, fiber.Map{"created": n})
}

func (h *Handler) IssueVA(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	t, err := h.svc.IssueVA(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, t)
}

func (h *Handler) Cicilan(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	var in struct {
		JumlahCicilan int `json:"jumlah_cicilan"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	t, err := h.svc.Cicilan(c.UserContext(), id, in.JumlahCicilan)
	if err != nil {
		return err
	}
	return httpx.OK(c, t)
}

func (h *Handler) Pembayaran(c *fiber.Ctx) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	ps, err := h.svc.Pembayaran(c.UserContext(), id)
	if err != nil {
		return err
	}
	return httpx.OK(c, ps)
}

func (h *Handler) Webhook(c *fiber.Ctx) error {
	r, err := h.svc.HandleWebhook(c.UserContext(), c.Body(), c.Get("X-Signature"))
	if err != nil {
		return err
	}
	return httpx.OK(c, r)
}

func (h *Handler) SimulatePay(c *fiber.Ctx) error {
	var in struct {
		VANumber string `json:"va_number"`
		Amount   int64  `json:"amount"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	t, err := h.svc.SimulatePay(c.UserContext(), in.VANumber, in.Amount)
	if err != nil {
		return err
	}
	return httpx.OK(c, t)
}

func (h *Handler) ListDispensasi(c *fiber.Ctx) error {
	ds, err := h.svc.ListDispensasi(c.UserContext(), c.Query("status"))
	if err != nil {
		return err
	}
	return httpx.OK(c, ds)
}

func (h *Handler) CreateDispensasi(c *fiber.Ctx) error {
	var in DispensasiInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	d, err := h.svc.CreateDispensasi(c.UserContext(), in)
	if err != nil {
		return err
	}
	return httpx.Created(c, d)
}

func (h *Handler) decide(c *fiber.Ctx, approve bool) error {
	id, err := httpx.ParamID(c, "id")
	if err != nil {
		return err
	}
	d, err := h.svc.DecideDispensasi(c.UserContext(), id, approve)
	if err != nil {
		return err
	}
	return httpx.OK(c, d)
}

func (h *Handler) ApproveDispensasi(c *fiber.Ctx) error { return h.decide(c, true) }
func (h *Handler) RejectDispensasi(c *fiber.Ctx) error  { return h.decide(c, false) }

func (h *Handler) Piutang(c *fiber.Ctx) error {
	g := c.Query("group", "prodi")
	if g != "prodi" && g != "angkatan" {
		return httpx.ValidationField("group", "prodi atau angkatan")
	}
	rows, err := h.svc.Piutang(c.UserContext(), g, c.Query("periode_id"))
	if err != nil {
		return err
	}
	return httpx.OK(c, rows)
}

var _ = strconv.Itoa
