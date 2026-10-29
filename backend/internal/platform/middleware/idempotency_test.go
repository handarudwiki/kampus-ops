package middleware

import (
	"io"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/httpx"
	"github.com/kampusops/backend/internal/platform/redis"
)

func TestIdempotencyReplaysFirstResponse(t *testing.T) { // EC-02
	var calls int32
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	app.Use(func(c *fiber.Ctx) error {
		c.SetUserContext(appctx.WithPrincipal(c.UserContext(), appctx.Principal{UserID: c.Get("X-User"), TenantID: "t1"}))
		return c.Next()
	})
	app.Post("/krs/me/items", Idempotency(redis.NewMemoryKV()), func(c *fiber.Ctx) error {
		n := atomic.AddInt32(&calls, 1)
		if c.Query("fail") == "1" {
			return httpx.Rule("KRS_QUOTA_FULL", "Kelas sudah penuh")
		}
		return c.JSON(fiber.Map{"call": n})
	})
	do := func(user, key, q string) (int, string, string) {
		req := httptest.NewRequest("POST", "/krs/me/items"+q, strings.NewReader(`{}`))
		req.Header.Set("X-User", user)
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b), res.Header.Get("Idempotent-Replayed")
	}
	s1, b1, _ := do("u1", "k1", "")
	s2, b2, rep := do("u1", "k1", "")
	if s1 != 200 || s2 != 200 || b1 != b2 || rep != "true" || calls != 1 {
		t.Fatalf("replay: %d %s / %d %s replayed=%s calls=%d", s1, b1, s2, b2, rep, calls)
	}
	// same key, different user → executes separately
	if _, b3, _ := do("u2", "k1", ""); b3 == b1 || calls != 2 {
		t.Fatal("keys must be scoped per user")
	}
	// no key → always executes
	do("u1", "", "")
	do("u1", "", "")
	if calls != 4 {
		t.Fatalf("calls=%d", calls)
	}
	// business errors (4xx) are stored and replayed as well
	s5, b5, _ := do("u1", "k-err", "?fail=1")
	s6, b6, rep6 := do("u1", "k-err", "?fail=1")
	if s5 != 422 || s6 != 422 || b5 != b6 || rep6 != "true" || calls != 5 {
		t.Fatalf("error replay: %d %d %s calls=%d", s5, s6, rep6, calls)
	}
}
