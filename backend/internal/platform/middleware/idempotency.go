package middleware

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/httpx"
	"github.com/kampusops/backend/internal/platform/redis"
)

const IdempotencyTTL = 24 * time.Hour

type storedResponse struct {
	Done        bool   `json:"done"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Status      int    `json:"status"`
	ContentType string `json:"content_type"`
	Body        []byte `json:"body"`
}

// Idempotency implements EC-02: a request carrying Idempotency-Key is executed once per
// (tenant, user, key) for 24h; repeats replay the stored status and body.
func Idempotency(kv redis.KV) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Get("Idempotency-Key")
		if key == "" {
			return c.Next()
		}
		if len(key) > 200 {
			return httpx.ValidationField("Idempotency-Key", "maksimal 200 karakter")
		}
		ctx := c.UserContext()
		p, _ := appctx.PrincipalFrom(ctx)
		rk := "idem:" + p.TenantID + ":" + p.UserID + ":" + key
		marker, _ := json.Marshal(storedResponse{Method: c.Method(), Path: c.Path()})
		ok, err := kv.SetNX(ctx, rk, marker, IdempotencyTTL)
		if err != nil {
			return err
		}
		if !ok {
			b, found, err := kv.Get(ctx, rk)
			if err != nil {
				return err
			}
			var s storedResponse
			if found {
				_ = json.Unmarshal(b, &s)
			}
			if !found || !s.Done {
				return httpx.Conflict("IDEMPOTENCY_IN_PROGRESS", "Permintaan yang sama sedang diproses")
			}
			if s.Method != c.Method() || s.Path != c.Path() {
				return httpx.ValidationField("Idempotency-Key", "kunci sudah dipakai untuk permintaan lain")
			}
			c.Set("Idempotent-Replayed", "true")
			if s.ContentType != "" {
				c.Set(fiber.HeaderContentType, s.ContentType)
			}
			return c.Status(s.Status).Send(s.Body)
		}
		err = c.Next()
		if err != nil {
			// let the error handler render it now, so we can store the rendered response
			if herr := c.App().ErrorHandler(c, err); herr != nil {
				_ = kv.Del(ctx, rk)
				return herr
			}
		}
		status := c.Response().StatusCode()
		if status >= 500 {
			_ = kv.Del(ctx, rk) // allow a retry after server errors
			return nil
		}
		s := storedResponse{Done: true, Method: c.Method(), Path: c.Path(), Status: status,
			ContentType: string(c.Response().Header.ContentType()), Body: append([]byte(nil), c.Response().Body()...)}
		b, _ := json.Marshal(s)
		_ = kv.Set(ctx, rk, b, IdempotencyTTL)
		return nil
	}
}
