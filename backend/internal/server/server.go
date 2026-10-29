// Package server assembles the Fiber application: middleware, routes and RBAC.
package server

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kampusops/backend/internal/academic"
	"github.com/kampusops/backend/internal/admission"
	"github.com/kampusops/backend/internal/audit"
	"github.com/kampusops/backend/internal/auth"
	"github.com/kampusops/backend/internal/dashboard"
	"github.com/kampusops/backend/internal/finance"
	"github.com/kampusops/backend/internal/krs"
	"github.com/kampusops/backend/internal/lecture"
	"github.com/kampusops/backend/internal/notification"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/config"
	"github.com/kampusops/backend/internal/platform/httpx"
	mw "github.com/kampusops/backend/internal/platform/middleware"
	"github.com/kampusops/backend/internal/platform/redis"
	"github.com/kampusops/backend/internal/platform/token"
	"github.com/kampusops/backend/internal/reporting"
	"github.com/kampusops/backend/internal/tenant"
)

type Deps struct {
	Cfg  config.Config
	Pool *pgxpool.Pool
	KV   redis.KV
	Log  *slog.Logger
	// Now overrides the clock of time-dependent services (tests).
	Now func() time.Time
}

// Services exposes the constructed services (used by tests).
type Services struct {
	Academic  *academic.Service
	KRS       *krs.Service
	Lecture   *lecture.Service
	Finance   *finance.Service
	Admission *admission.Service
	Reporting *reporting.Service
	Tenants   *tenant.Store
}

const (
	superAdmin = "super_admin"
	admin      = "admin_kampus"
	operator   = "operator"
	kaprodi    = "kaprodi"
	wali       = "dosen_wali"
	dosen      = "dosen"
	keuangan   = "keuangan"
	mhs        = "mahasiswa"
	calon      = "calon_mahasiswa"
)

func New(d Deps) (*fiber.App, *Services) {
	if d.Log == nil {
		d.Log = slog.Default()
	}
	app := fiber.New(fiber.Config{
		ErrorHandler:          httpx.ErrorHandler,
		DisableStartupMessage: true,
		AppName:               "KampusOps API",
		BodyLimit:             2 * 1024 * 1024,
		ReadTimeout:           15 * time.Second,
		WriteTimeout:          30 * time.Second,
	})
	app.Use(recover.New())
	app.Use(mw.RequestID())
	app.Use(mw.Logger(d.Log))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.ReplaceAll(d.Cfg.CORSOrigins, " ", ""),
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Active-Role, Idempotency-Key, X-Request-ID",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		ExposeHeaders:    "X-Request-ID, Idempotent-Replayed, Content-Disposition",
		AllowCredentials: false,
		MaxAge:           600,
	}))

	tenants := tenant.NewStore(d.Pool)
	iss := token.NewIssuer(d.Cfg.JWTSecret)
	authSvc := auth.NewService(d.Pool, tenants, iss)
	authH := auth.NewHandler(authSvc, tenants)
	acadH := academic.NewHandler(d.Pool, tenants)
	krsSvc := krs.NewService(d.Pool, tenants)
	lecSvc := lecture.NewService(d.Pool, tenants)
	finSvc := finance.NewService(d.Pool, tenants, d.Cfg.WebhookSecret)
	admSvc := admission.NewService(d.Pool, tenants)
	repSvc := reporting.NewService(d.Pool)
	dashSvc := dashboard.NewService(d.Pool, tenants, krsSvc, acadH.Service())
	if d.Now != nil {
		acadH.Service().Now, krsSvc.Now, lecSvc.Now, finSvc.Now, admSvc.Now, repSvc.Now, dashSvc.Now = d.Now, d.Now, d.Now, d.Now, d.Now, d.Now, d.Now
	}
	krsH := krs.NewHandler(krsSvc)
	lecH := lecture.NewHandler(lecSvc)
	finH := finance.NewHandler(finSvc)
	admH := admission.NewHandler(admSvc)
	repH := reporting.NewHandler(repSvc)
	auditH := audit.NewHandler(d.Pool)
	notifH := notification.NewHandler(d.Pool)

	// ---- platform ----
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Get("/readyz", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		checks := fiber.Map{"db": "ok", "redis": "ok"}
		status := 200
		if err := d.Pool.Ping(ctx); err != nil {
			checks["db"], status = err.Error(), 503
		}
		if d.KV != nil {
			if err := d.KV.Ping(ctx); err != nil {
				checks["redis"], status = err.Error(), 503
			}
		}
		return c.Status(status).JSON(fiber.Map{"status": map[bool]string{true: "ready", false: "not_ready"}[status == 200], "checks": checks})
	})
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	api := app.Group("/api/v1")
	R := mw.Roles

	// ---- public ----
	loginLimiter := limiter.New(limiter.Config{Max: 30, Expiration: time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return httpx.E(429, "RATE_LIMITED", "Terlalu banyak percobaan login, coba lagi sebentar lagi")
		}})
	api.Post("/auth/login", loginLimiter, authH.Login)
	api.Post("/auth/refresh", authH.Refresh)
	api.Post("/auth/logout", authH.Logout)
	api.Get("/tenants/public", authH.PublicTenants)
	api.Get("/pmb/public-options", admH.PublicOptions)
	api.Post("/pmb/register", loginLimiter, admH.Register)
	api.Post("/webhooks/payment", finH.Webhook)
	api.Get("/pmb/gelombang", mw.OptionalAuth(iss, tenants), admH.ListGelombang)

	// ---- authenticated ----
	a := api.Group("", mw.Auth(iss, tenants))
	readers := []string{superAdmin, admin, operator, kaprodi, wali, dosen, keuangan, mhs}
	staff := []string{admin, operator}

	a.Get("/me", authH.Me)
	a.Get("/dashboard", dashSvc.Handle)
	a.Get("/notifikasi", notifH.List)
	a.Post("/notifikasi/:id/read", notifH.Read)

	a.Get("/tenants", R(superAdmin), authH.ListTenants)
	a.Post("/tenants", R(superAdmin), authH.CreateTenant)
	a.Get("/users", R(admin, superAdmin), authH.ListUsers)
	a.Post("/users", R(admin, superAdmin), authH.CreateUser)
	a.Put("/users/:id/roles", R(admin, superAdmin), authH.SetRoles)

	// master data
	crud := func(path string, list, get, save, del fiber.Handler) {
		a.Get(path, R(readers...), list)
		a.Get(path+"/:id", R(readers...), get)
		a.Post(path, R(staff...), save)
		a.Put(path+"/:id", R(staff...), save)
		a.Delete(path+"/:id", R(staff...), del)
	}
	crud("/fakultas", acadH.ListFakultas, acadH.GetFakultas, acadH.SaveFakultas, acadH.DeleteFakultas)
	crud("/prodi", acadH.ListProdi, acadH.GetProdi, acadH.SaveProdi, acadH.DeleteProdi)
	crud("/dosen", acadH.ListDosen, acadH.GetDosen, acadH.SaveDosen, acadH.DeleteDosen)
	crud("/ruang", acadH.ListRuang, acadH.GetRuang, acadH.SaveRuang, acadH.DeleteRuang)
	a.Get("/periode/active", acadH.ActivePeriode)
	crud("/periode", acadH.ListPeriode, acadH.GetPeriode, acadH.SavePeriode, acadH.DeletePeriode)
	a.Post("/periode/:id/activate", R(staff...), acadH.ActivatePeriode)

	mhsReaders := []string{admin, operator, kaprodi, wali, keuangan}
	a.Get("/mahasiswa", R(mhsReaders...), acadH.ListMahasiswa)
	a.Get("/mahasiswa/:id", R(mhsReaders...), acadH.GetMahasiswa)
	a.Post("/mahasiswa", R(staff...), acadH.SaveMahasiswa)
	a.Put("/mahasiswa/:id", R(staff...), acadH.SaveMahasiswa)
	a.Delete("/mahasiswa/:id", R(staff...), acadH.DeleteMahasiswa)
	a.Get("/mahasiswa/:id/riwayat-status", R(mhsReaders...), acadH.RiwayatStatus)
	a.Post("/mahasiswa/:id/status", R(staff...), acadH.ChangeStatus)

	// kurikulum
	a.Get("/kurikulum", R(readers...), acadH.ListKurikulum)
	a.Get("/kurikulum/:id/mata-kuliah", R(readers...), acadH.ListMataKuliah)
	a.Post("/mata-kuliah", R(kaprodi, operator, admin), acadH.SaveMataKuliah)
	a.Put("/mata-kuliah/:id", R(kaprodi, operator, admin), acadH.SaveMataKuliah)
	a.Get("/ekuivalensi", R(readers...), acadH.ListEkuivalensi)
	a.Post("/ekuivalensi", R(kaprodi, operator, admin), acadH.CreateEkuivalensi)

	// kelas & jadwal
	kelasViewers := []string{operator, admin, kaprodi, dosen, wali}
	a.Get("/kelas", R(readers...), acadH.ListKelas)
	a.Get("/kelas/:id", R(kelasViewers...), acadH.GetKelas)
	a.Post("/kelas", R(operator, kaprodi, admin), acadH.SaveKelas)
	a.Put("/kelas/:id", R(operator, kaprodi, admin), acadH.SaveKelas)
	a.Delete("/kelas/:id", R(operator, admin), acadH.DeleteKelas)
	a.Post("/kelas/:id/cancel", R(operator, kaprodi, admin), acadH.CancelKelas)
	a.Post("/kelas/:id/ganti-dosen", R(operator, kaprodi, admin), acadH.GantiDosen)

	// KRS — mahasiswa (EC-01, EC-02) with a per-user rate limit (NFR security)
	krsLimiter := limiter.New(limiter.Config{Max: 60, Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string { return "krs:" + appctx.UserID(c.UserContext()) },
		LimitReached: func(c *fiber.Ctx) error { return httpx.E(429, "RATE_LIMITED", "Terlalu banyak permintaan KRS") }})
	idem := mw.Idempotency(d.KV)
	a.Get("/krs/me", R(mhs), krsH.Mine)
	a.Get("/krs/me/offerings", R(mhs), krsH.Offerings)
	a.Post("/krs/me/items", R(mhs), krsLimiter, idem, krsH.AddItem)
	a.Delete("/krs/me/items/:kelas_id", R(mhs), krsLimiter, idem, krsH.RemoveItem)
	a.Post("/krs/me/submit", R(mhs), krsLimiter, idem, krsH.Submit)
	// KRS — dosen wali
	a.Get("/krs/advisees", R(wali, operator, admin, kaprodi), krsH.Advisees)
	a.Get("/krs/:id", R(wali, operator, admin, kaprodi, mhs), krsH.Get)
	a.Post("/krs/:id/approve", R(wali, operator), idem, krsH.Approve)
	a.Post("/krs/:id/reject", R(wali, operator), idem, krsH.Reject)

	// perkuliahan
	lec := []string{dosen, wali, kaprodi, operator, admin}
	a.Get("/teaching/kelas", R(dosen, wali, kaprodi), lecH.Teaching)
	a.Get("/kelas/:id/peserta", R(lec...), lecH.Peserta)
	a.Get("/kelas/:id/pertemuan", R(lec...), lecH.ListPertemuan)
	a.Post("/kelas/:id/pertemuan", R(dosen, wali, operator), lecH.CreatePertemuan)
	a.Get("/pertemuan/:id/presensi", R(lec...), lecH.Presensi)
	a.Put("/pertemuan/:id/presensi", R(dosen, wali, operator), lecH.SavePresensi)
	a.Get("/kelas/:id/nilai", R(lec...), lecH.GetNilai)
	a.Put("/kelas/:id/nilai", R(dosen, wali, operator), lecH.SaveNilai)
	a.Post("/kelas/:id/nilai/finalize", R(dosen, wali, operator), lecH.Finalize)
	a.Post("/nilai/:id/koreksi", R(dosen, wali, operator), lecH.RequestKoreksi)
	a.Get("/koreksi-nilai", R(kaprodi, dosen, wali, operator, admin), lecH.ListKoreksi)
	a.Post("/koreksi-nilai/:id/approve", R(kaprodi), lecH.ApproveKoreksi)
	a.Post("/koreksi-nilai/:id/reject", R(kaprodi), lecH.RejectKoreksi)

	// akademik — mahasiswa
	a.Get("/akademik/me/jadwal", R(mhs), acadH.MyJadwal)
	a.Get("/akademik/me/khs", R(mhs), acadH.MyKHS)
	a.Get("/akademik/me/periodes", R(mhs), acadH.MyPeriodes)
	a.Get("/akademik/me/transkrip", R(mhs), acadH.MyTranskrip)
	a.Get("/akademik/me/profil", R(mhs), acadH.MyProfil)
	a.Get("/akademik/masa-studi-warning", R(kaprodi, operator, admin), acadH.MasaStudiWarning)

	// keuangan
	fin := []string{keuangan, admin, operator}
	a.Get("/tagihan", R(fin...), finH.List)
	a.Get("/tagihan/me", R(mhs, calon), finH.Mine)
	a.Post("/tagihan/generate", R(keuangan, admin), finH.Generate)
	a.Post("/tagihan/:id/va", R(keuangan, admin, mhs, calon), finH.IssueVA)
	a.Post("/tagihan/:id/cicilan", R(keuangan), finH.Cicilan)
	a.Get("/tagihan/:id/pembayaran", R(keuangan, admin, operator, mhs, calon), finH.Pembayaran)
	a.Post("/simulator/pay", idem, finH.SimulatePay)
	a.Get("/dispensasi", R(keuangan, operator, admin, mhs), finH.ListDispensasi)
	a.Post("/dispensasi", R(keuangan, operator, admin, mhs), finH.CreateDispensasi)
	a.Post("/dispensasi/:id/approve", R(keuangan, operator, admin), finH.ApproveDispensasi)
	a.Post("/dispensasi/:id/reject", R(keuangan, operator, admin), finH.RejectDispensasi)
	a.Get("/keuangan/piutang", R(keuangan, admin), finH.Piutang)

	// PMB
	a.Post("/pmb/gelombang", R(staff...), admH.CreateGelombang)
	a.Get("/pmb/pendaftar", R(staff...), admH.ListPendaftar)
	a.Get("/pmb/me", R(calon), admH.Me)
	a.Post("/pmb/pendaftar/:id/transition", R(admin, operator, calon), admH.Transition)
	a.Post("/pmb/pendaftar/:id/daftar-ulang", R(staff...), admH.DaftarUlang)

	// PDDikti
	a.Post("/pddikti/validate", R(operator, admin), repH.Validate)
	a.Get("/pddikti/validation", R(operator, admin), repH.Latest)
	a.Get("/pddikti/export/:entity", R(operator, admin), repH.Export)

	// audit
	a.Get("/audit-log", R(admin, operator, keuangan), auditH.List)

	app.Use(func(c *fiber.Ctx) error { return httpx.NotFound("Endpoint tidak ditemukan") })

	return app, &Services{Academic: acadH.Service(), KRS: krsSvc, Lecture: lecSvc, Finance: finSvc, Admission: admSvc, Reporting: repSvc, Tenants: tenants}
}
