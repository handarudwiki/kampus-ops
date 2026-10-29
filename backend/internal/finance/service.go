package finance

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kampusops/backend/internal/academic"
	"github.com/kampusops/backend/internal/audit"
	"github.com/kampusops/backend/internal/domain"
	"github.com/kampusops/backend/internal/notification"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/httpx"
	"github.com/kampusops/backend/internal/tenant"
)

type Service struct {
	pool          *pgxpool.Pool
	tenants       *tenant.Store
	webhookSecret string
	Now           func() time.Time
}

func NewService(pool *pgxpool.Pool, tenants *tenant.Store, webhookSecret string) *Service {
	return &Service{pool: pool, tenants: tenants, webhookSecret: webhookSecret, Now: time.Now}
}

type Tagihan struct {
	ID          string        `json:"id"`
	MahasiswaID *string       `json:"mahasiswa_id"`
	NIM         string        `json:"nim"`
	Nama        string        `json:"nama"`
	ProdiNama   string        `json:"prodi_nama"`
	PeriodeKode string        `json:"periode_kode"`
	Jenis       string        `json:"jenis"`
	Jumlah      int64         `json:"jumlah"`
	Terbayar    int64         `json:"terbayar"`
	Status      string        `json:"status"`
	JatuhTempo  string        `json:"jatuh_tempo"`
	VANumber    *string       `json:"va_number"`
	Cicilan     []Installment `json:"cicilan"`
}

type Pembayaran struct {
	ID            string `json:"id"`
	TagihanID     string `json:"tagihan_id"`
	TransactionID string `json:"transaction_id"`
	Jumlah        int64  `json:"jumlah"`
	PaidAt        string `json:"paid_at"`
}

type Dispensasi struct {
	ID            string  `json:"id"`
	MahasiswaID   string  `json:"mahasiswa_id"`
	NIM           string  `json:"nim"`
	Nama          string  `json:"nama"`
	PeriodeKode   string  `json:"periode_kode"`
	Jenis         string  `json:"jenis"`
	Alasan        string  `json:"alasan"`
	Status        string  `json:"status"`
	DisetujuiOleh *string `json:"disetujui_oleh"`
	CreatedAt     string  `json:"created_at"`
}

const tagihanSel = `SELECT t.id, t.mahasiswa_id, coalesce(m.nim, pd.no_daftar, ''), coalesce(m.nama, pd.nama, ''), coalesce(pm.nama, pp.nama, ''),
	coalesce(pe.kode, ''), t.jenis, t.jumlah, t.terbayar, t.status, t.jatuh_tempo::text, t.va_number
	FROM tagihan t LEFT JOIN mahasiswa m ON m.id=t.mahasiswa_id LEFT JOIN prodi pm ON pm.id=m.prodi_id
	LEFT JOIN pendaftar pd ON pd.id=t.pendaftar_id LEFT JOIN prodi pp ON pp.id=pd.prodi_pilihan_id
	LEFT JOIN periode pe ON pe.id=t.periode_id `

func (s *Service) queryTagihan(ctx context.Context, q db.Querier, where string, args ...any) ([]Tagihan, error) {
	rows, err := q.Query(ctx, tagihanSel+`WHERE t.tenant_id=$1 AND `+where, append([]any{appctx.TenantID(ctx)}, args...)...)
	if err != nil {
		return nil, err
	}
	out := []Tagihan{}
	for rows.Next() {
		var t Tagihan
		if err := rows.Scan(&t.ID, &t.MahasiswaID, &t.NIM, &t.Nama, &t.ProdiNama, &t.PeriodeKode, &t.Jenis, &t.Jumlah, &t.Terbayar, &t.Status, &t.JatuhTempo, &t.VANumber); err != nil {
			rows.Close()
			return nil, err
		}
		t.Cicilan = []Installment{}
		out = append(out, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(out) == 0 {
		return out, err
	}
	ids := make([]string, len(out))
	idx := map[string]int{}
	for i, t := range out {
		ids[i] = t.ID
		idx[t.ID] = i
	}
	crow, err := q.Query(ctx, `SELECT tagihan_id, ke, jumlah, jatuh_tempo::text FROM cicilan WHERE tenant_id=$1 AND tagihan_id = ANY($2::uuid[]) ORDER BY ke`, appctx.TenantID(ctx), ids)
	if err != nil {
		return nil, err
	}
	defer crow.Close()
	for crow.Next() {
		var tid string
		var c Installment
		if err := crow.Scan(&tid, &c.Ke, &c.Jumlah, &c.JatuhTempo); err != nil {
			return nil, err
		}
		out[idx[tid]].Cicilan = append(out[idx[tid]].Cicilan, c)
	}
	for i := range out {
		out[i].Cicilan = MarkLunas(out[i].Cicilan, out[i].Terbayar)
	}
	return out, crow.Err()
}

func (s *Service) getTagihan(ctx context.Context, q db.Querier, id string) (Tagihan, error) {
	ts, err := s.queryTagihan(ctx, q, `t.id=$2`, id)
	if err != nil {
		return Tagihan{}, err
	}
	if len(ts) == 0 {
		return Tagihan{}, httpx.NotFound("Tagihan tidak ditemukan")
	}
	return ts[0], nil
}

type ListFilter struct {
	Status, PeriodeID, ProdiID, Q, Jenis string
}

func (s *Service) List(ctx context.Context, f ListFilter, limit, offset int) ([]Tagihan, int, error) {
	where := `($2='' OR t.status=$2) AND ($3='' OR t.periode_id::text=$3) AND ($4='' OR pm.id::text=$4 OR pp.id::text=$4)
		AND (coalesce(m.nama, pd.nama) ILIKE $5 OR coalesce(m.nim, pd.no_daftar) ILIKE $5 OR coalesce(t.va_number,'') ILIKE $5) AND ($6='' OR t.jenis=$6)`
	args := []any{f.Status, f.PeriodeID, f.ProdiID, "%" + strings.TrimSpace(f.Q) + "%", f.Jenis}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM tagihan t LEFT JOIN mahasiswa m ON m.id=t.mahasiswa_id LEFT JOIN prodi pm ON pm.id=m.prodi_id
		LEFT JOIN pendaftar pd ON pd.id=t.pendaftar_id LEFT JOIN prodi pp ON pp.id=pd.prodi_pilihan_id WHERE t.tenant_id=$1 AND `+where,
		append([]any{appctx.TenantID(ctx)}, args...)...).Scan(&total); err != nil {
		return nil, 0, err
	}
	ts, err := s.queryTagihan(ctx, s.pool, where+` ORDER BY t.status <> 'belum_bayar', t.jatuh_tempo, 3 LIMIT $7 OFFSET $8`, append(args, limit, offset)...)
	return ts, total, err
}

// Mine returns the caller's bills (mahasiswa, or calon_mahasiswa for the PMB form fee).
func (s *Service) Mine(ctx context.Context) ([]Tagihan, error) {
	uid := appctx.UserID(ctx)
	return s.queryTagihan(ctx, s.pool, `(t.mahasiswa_id IN (SELECT id FROM mahasiswa WHERE tenant_id=$1 AND user_id=$2)
		OR t.pendaftar_id IN (SELECT id FROM pendaftar WHERE tenant_id=$1 AND user_id=$2)) ORDER BY t.jatuh_tempo DESC`, uid)
}

// Generate creates bills for a period from prodi/angkatan tariffs (FR-25). Students on
// cuti get the cuti fee instead (EC-08). Existing bills are kept (idempotent).
func (s *Service) Generate(ctx context.Context, periodeID string) (int, error) {
	if _, err := httpx.OptUUID("periode_id", periodeID); err != nil || periodeID == "" {
		return 0, httpx.ValidationField("periode_id", "wajib diisi")
	}
	p, err := academic.GetPeriode(ctx, s.pool, periodeID)
	if err != nil {
		return 0, err
	}
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return 0, err
	}
	due := p.Tahapan.KrsSelesai
	if due == "" {
		due = domain.LocalDate(s.Now().AddDate(0, 0, 30), t.Loc)
	}
	var created int
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		p, _ := appctx.PrincipalFrom(ctx)
		actor, nama, role := (*string)(nil), "system", "system"
		if p.UserID != "" {
			actor, nama, role = &p.UserID, p.Name, p.ActiveRole
		}
		// latest status per student from riwayat_status
		err := tx.QueryRow(ctx, `WITH st AS (
				SELECT DISTINCT ON (r.mahasiswa_id) r.mahasiswa_id, r.status FROM riwayat_status r WHERE r.tenant_id=$1
				ORDER BY r.mahasiswa_id, r.created_at DESC, r.id DESC),
			src AS (
				SELECT m.id AS mahasiswa_id, tr.jenis, tr.jumlah FROM mahasiswa m JOIN st ON st.mahasiswa_id=m.id AND st.status='aktif'
				JOIN LATERAL (SELECT DISTINCT ON (jenis) jenis, jumlah FROM tarif WHERE tenant_id=$1 AND prodi_id=m.prodi_id AND angkatan <= m.angkatan
					AND jenis IN ('ukt','praktikum') ORDER BY jenis, angkatan DESC) tr ON true
				WHERE m.tenant_id=$1
				UNION ALL
				SELECT m.id, 'cuti', $5::bigint FROM mahasiswa m JOIN st ON st.mahasiswa_id=m.id AND st.status='cuti' WHERE m.tenant_id=$1),
			ins AS (
				INSERT INTO tagihan (tenant_id, mahasiswa_id, periode_id, jenis, jumlah, jatuh_tempo)
				SELECT $1, mahasiswa_id, $2, jenis, jumlah, $3::date FROM src
				ON CONFLICT (tenant_id, mahasiswa_id, periode_id, jenis) WHERE mahasiswa_id IS NOT NULL DO NOTHING
				RETURNING id, mahasiswa_id, jenis, jumlah),
			aud AS (
				INSERT INTO audit_log (tenant_id, actor_id, actor_nama, actor_role, entitas, entitas_id, aksi, after, request_id)
				SELECT $1, $4, $6, $7, 'tagihan', id, 'generate', jsonb_build_object('mahasiswa_id', mahasiswa_id, 'jenis', jenis, 'jumlah', jumlah), $8 FROM ins
				RETURNING 1)
			SELECT count(*) FROM aud`, tid, periodeID, due, actor, t.Rules.BiayaCuti, nama, role, appctx.RequestID(ctx)).Scan(&created)
		return err
	})
	return created, err
}

func randomDigits(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		d, _ := rand.Int(rand.Reader, big.NewInt(10))
		b.WriteByte(byte('0' + d.Int64()))
	}
	return b.String()
}

// IssueVA assigns a fake virtual account number (tenant prefix + 12 random digits).
func (s *Service) IssueVA(ctx context.Context, id string) (Tagihan, error) {
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return Tagihan{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		var va *string
		var status string
		if err := tx.QueryRow(ctx, `SELECT va_number, status FROM tagihan WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, appctx.TenantID(ctx), id).Scan(&va, &status); err != nil {
			return httpx.NotFound("Tagihan tidak ditemukan")
		}
		if va != nil {
			return nil
		}
		if status == "lunas" {
			return httpx.Rule("VALIDATION_ERROR", "Tagihan sudah lunas")
		}
		for i := 0; i < 5; i++ {
			n := t.Rules.VAPrefix + randomDigits(12)
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tagihan WHERE va_number=$1)`, n).Scan(&exists); err != nil {
				return err
			}
			if exists {
				continue
			}
			if _, err := tx.Exec(ctx, `UPDATE tagihan SET va_number=$3, updated_at=now() WHERE tenant_id=$1 AND id=$2`, appctx.TenantID(ctx), id, n); err != nil {
				return err
			}
			return audit.Write(ctx, tx, "tagihan", id, "issue_va", nil, map[string]string{"va_number": n})
		}
		return fmt.Errorf("could not allocate VA number")
	})
	if err != nil {
		return Tagihan{}, err
	}
	return s.getTagihan(ctx, s.pool, id)
}

// Cicilan creates an installment plan (FR-27); approved by keuangan by creating it.
func (s *Service) Cicilan(ctx context.Context, id string, n int) (Tagihan, error) {
	if n < 2 || n > 4 {
		return Tagihan{}, httpx.ValidationField("jumlah_cicilan", "harus 2, 3, atau 4")
	}
	tg, err := s.getTagihan(ctx, s.pool, id)
	if err != nil {
		return tg, err
	}
	if tg.Status == "lunas" {
		return tg, httpx.Rule("VALIDATION_ERROR", "Tagihan sudah lunas")
	}
	if len(tg.Cicilan) > 0 {
		return tg, httpx.Conflict("DUPLICATE", "Tagihan sudah memiliki cicilan")
	}
	plan := SplitCicilan(tg.Jumlah, n, tg.JatuhTempo)
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		for _, c := range plan {
			if _, err := tx.Exec(ctx, `INSERT INTO cicilan (tenant_id, tagihan_id, ke, jumlah, jatuh_tempo) VALUES ($1,$2,$3,$4,$5)`,
				appctx.TenantID(ctx), id, c.Ke, c.Jumlah, c.JatuhTempo); err != nil {
				return err
			}
		}
		return audit.Write(ctx, tx, "tagihan", id, "cicilan", nil, plan)
	})
	if err != nil {
		return Tagihan{}, err
	}
	return s.getTagihan(ctx, s.pool, id)
}

func (s *Service) Pembayaran(ctx context.Context, id string) ([]Pembayaran, error) {
	if _, err := s.getTagihan(ctx, s.pool, id); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, tagihan_id, transaction_id, jumlah, iso(paid_at) FROM pembayaran WHERE tenant_id=$1 AND tagihan_id=$2 ORDER BY paid_at`,
		appctx.TenantID(ctx), id)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Pembayaran])
	if out == nil {
		out = []Pembayaran{}
	}
	return out, err
}

// WebhookPayload is the payment gateway event.
type WebhookPayload struct {
	TransactionID string `json:"transaction_id"`
	VANumber      string `json:"va_number"`
	Amount        int64  `json:"amount"`
	PaidAt        string `json:"paid_at"`
}

type WebhookResult struct {
	TagihanID string `json:"tagihan_id"`
	Status    string `json:"status"`
	Terbayar  int64  `json:"terbayar"`
	Duplicate bool   `json:"duplicate"`
}

// HandleWebhook verifies the HMAC signature and applies the payment (EC-03).
func (s *Service) HandleWebhook(ctx context.Context, body []byte, sig string) (WebhookResult, error) {
	if !VerifySignature(s.webhookSecret, body, sig) {
		return WebhookResult{}, httpx.Unauthorized("INVALID_SIGNATURE", "Signature webhook tidak valid")
	}
	var p WebhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return WebhookResult{}, httpx.ValidationField("body", "JSON tidak valid")
	}
	paid, err := time.Parse(time.RFC3339, p.PaidAt)
	v := httpx.NewV().Required("transaction_id", p.TransactionID).Required("va_number", p.VANumber).Check(p.Amount > 0, "amount", "harus > 0").
		Check(err == nil, "paid_at", "format RFC3339")
	if err := v.Err(); err != nil {
		return WebhookResult{}, err
	}
	return s.applyPayment(ctx, p, paid)
}

func (s *Service) applyPayment(ctx context.Context, p WebhookPayload, paidAt time.Time) (WebhookResult, error) {
	var res WebhookResult
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		// The webhook carries no tenant: resolve it from the VA (tenant-prefixed numbers).
		var tenantID, jenis, before string
		var jumlah int64
		var pendaftarID, mhsID *string
		err := tx.QueryRow(ctx, `SELECT id, tenant_id, jumlah, jenis, status, pendaftar_id, mahasiswa_id FROM tagihan WHERE va_number=$1 FOR UPDATE`, p.VANumber).
			Scan(&res.TagihanID, &tenantID, &jumlah, &jenis, &before, &pendaftarID, &mhsID)
		if err != nil {
			return httpx.NotFound("Virtual account tidak dikenal")
		}
		if cur := appctx.TenantID(ctx); appctx.Tenant(ctx).ID != "" && cur != tenantID {
			return httpx.NotFound("Virtual account tidak dikenal") // EC-16: never reveal other tenants
		}
		if appctx.Tenant(ctx).ID == "" {
			t, err := s.tenants.Get(ctx, tenantID)
			if err != nil {
				return err
			}
			ctx = appctx.WithTenant(ctx, t.Info())
		}
		tag, err := tx.Exec(ctx, `INSERT INTO pembayaran (tenant_id, tagihan_id, transaction_id, jumlah, paid_at) VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (tenant_id, transaction_id) DO NOTHING`, tenantID, res.TagihanID, p.TransactionID, p.Amount, paidAt)
		if err != nil {
			return err
		}
		res.Duplicate = tag.RowsAffected() == 0
		if err := tx.QueryRow(ctx, `SELECT coalesce(sum(jumlah),0) FROM pembayaran WHERE tenant_id=$1 AND tagihan_id=$2`, tenantID, res.TagihanID).Scan(&res.Terbayar); err != nil {
			return err
		}
		res.Status = StatusFor(jumlah, res.Terbayar)
		if _, err := tx.Exec(ctx, `UPDATE tagihan SET terbayar=$3, status=$4, updated_at=now() WHERE tenant_id=$1 AND id=$2`, tenantID, res.TagihanID, res.Terbayar, res.Status); err != nil {
			return err
		}
		if res.Duplicate {
			return nil
		}
		if err := audit.Write(ctx, tx, "tagihan", res.TagihanID, "payment", map[string]any{"status": before},
			map[string]any{"status": res.Status, "terbayar": res.Terbayar, "transaction_id": p.TransactionID, "amount": p.Amount}); err != nil {
			return err
		}
		if res.Status == "lunas" && before != "lunas" {
			var uid *string
			if mhsID != nil {
				_ = tx.QueryRow(ctx, `SELECT user_id FROM mahasiswa WHERE id=$1`, *mhsID).Scan(&uid)
			} else if pendaftarID != nil {
				_ = tx.QueryRow(ctx, `SELECT user_id FROM pendaftar WHERE id=$1`, *pendaftarID).Scan(&uid)
			}
			if uid != nil {
				if err := notification.Send(ctx, tx, *uid, "Pembayaran lunas", "Tagihan "+jenis+" Anda sudah lunas.", notification.Link("/tagihan")); err != nil {
					return err
				}
			}
			// PMB: paying the form fee moves the applicant to bayar_formulir (FR-09).
			if jenis == "formulir_pmb" && pendaftarID != nil {
				t, err := tx.Exec(ctx, `UPDATE pendaftar SET status='bayar_formulir', updated_at=now() WHERE id=$1 AND tenant_id=$2 AND status='daftar'`, *pendaftarID, tenantID)
				if err != nil {
					return err
				}
				if t.RowsAffected() > 0 {
					if err := audit.Write(ctx, tx, "pendaftar", *pendaftarID, "transition", map[string]string{"status": "daftar"}, map[string]string{"status": "bayar_formulir"}); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	return res, err
}

// SimulatePay is the demo payment button: it builds and signs a webhook event and
// processes it through the same path as the real webhook.
func (s *Service) SimulatePay(ctx context.Context, va string, amount int64) (Tagihan, error) {
	if err := httpx.NewV().Required("va_number", va).Check(amount > 0, "amount", "harus > 0").Err(); err != nil {
		return Tagihan{}, err
	}
	var id string
	if err := s.pool.QueryRow(ctx, `SELECT id FROM tagihan WHERE tenant_id=$1 AND va_number=$2`, appctx.TenantID(ctx), va).Scan(&id); err != nil {
		return Tagihan{}, httpx.NotFound("Virtual account tidak dikenal")
	}
	body, _ := json.Marshal(WebhookPayload{TransactionID: "SIM-" + uuid.NewString(), VANumber: va, Amount: amount, PaidAt: s.Now().UTC().Format(time.RFC3339)})
	if _, err := s.HandleWebhook(ctx, body, Sign(s.webhookSecret, body)); err != nil {
		return Tagihan{}, err
	}
	return s.getTagihan(ctx, s.pool, id)
}

// ---- dispensasi (FR-27, EC-15) ----

const dispSel = `SELECT d.id, d.mahasiswa_id, m.nim, m.nama, p.kode, d.jenis, d.alasan, d.status, u.nama, iso(d.created_at)
	FROM dispensasi d JOIN mahasiswa m ON m.id=d.mahasiswa_id JOIN periode p ON p.id=d.periode_id LEFT JOIN users u ON u.id=d.disetujui_oleh `

func (s *Service) ListDispensasi(ctx context.Context, status string) ([]Dispensasi, error) {
	p, _ := appctx.PrincipalFrom(ctx)
	own := ""
	if p.ActiveRole == "mahasiswa" {
		own = p.UserID
	}
	rows, err := s.pool.Query(ctx, dispSel+`WHERE d.tenant_id=$1 AND ($2='' OR d.status=$2) AND ($3='' OR m.user_id::text=$3) ORDER BY d.created_at DESC LIMIT 500`,
		appctx.TenantID(ctx), status, own)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Dispensasi])
	if out == nil {
		out = []Dispensasi{}
	}
	return out, err
}

func (s *Service) getDispensasi(ctx context.Context, id string) (Dispensasi, error) {
	rows, err := s.pool.Query(ctx, dispSel+`WHERE d.tenant_id=$1 AND d.id=$2`, appctx.TenantID(ctx), id)
	if err != nil {
		return Dispensasi{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByPos[Dispensasi])
}

type DispensasiInput struct {
	MahasiswaID string `json:"mahasiswa_id"`
	PeriodeID   string `json:"periode_id"`
	Jenis       string `json:"jenis"`
	Alasan      string `json:"alasan"`
}

func (s *Service) CreateDispensasi(ctx context.Context, in DispensasiInput) (Dispensasi, error) {
	p, _ := appctx.PrincipalFrom(ctx)
	if p.ActiveRole == "mahasiswa" {
		m, err := academic.StudentByUser(ctx, s.pool, p.UserID)
		if err != nil {
			return Dispensasi{}, err
		}
		in.MahasiswaID = m.ID
	}
	if in.PeriodeID == "" {
		if ap, err := academic.ActivePeriode(ctx, s.pool); err == nil {
			in.PeriodeID = ap.ID
		}
	}
	if err := httpx.NewV().UUID("mahasiswa_id", in.MahasiswaID).UUID("periode_id", in.PeriodeID).OneOf("jenis", in.Jenis, "cicilan", "penundaan", "krs_susulan").
		Required("alasan", in.Alasan).Err(); err != nil {
		return Dispensasi{}, err
	}
	if err := academic.MustExist(ctx, s.pool, "mahasiswa", "mahasiswa_id", in.MahasiswaID); err != nil {
		return Dispensasi{}, err
	}
	if err := academic.MustExist(ctx, s.pool, "periode", "periode_id", in.PeriodeID); err != nil {
		return Dispensasi{}, err
	}
	var id string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO dispensasi (tenant_id, mahasiswa_id, periode_id, jenis, alasan, created_by) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
			appctx.TenantID(ctx), in.MahasiswaID, in.PeriodeID, in.Jenis, in.Alasan, p.UserID).Scan(&id); err != nil {
			return err
		}
		return audit.Write(ctx, tx, "dispensasi", id, "create", nil, in)
	})
	if err != nil {
		return Dispensasi{}, err
	}
	return s.getDispensasi(ctx, id)
}

// DecideDispensasi approves/rejects; an approved krs_susulan reopens KRS for that
// student only (EC-15) and is recorded in the audit log.
func (s *Service) DecideDispensasi(ctx context.Context, id string, approve bool) (Dispensasi, error) {
	status := "ditolak"
	if approve {
		status = "disetujui"
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		var cur, jenis, mhs string
		if err := tx.QueryRow(ctx, `SELECT status, jenis, mahasiswa_id FROM dispensasi WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tid, id).Scan(&cur, &jenis, &mhs); err != nil {
			return httpx.NotFound("Dispensasi tidak ditemukan")
		}
		if cur != "diajukan" {
			return httpx.Rule("VALIDATION_ERROR", "Dispensasi sudah diproses")
		}
		if _, err := tx.Exec(ctx, `UPDATE dispensasi SET status=$3, disetujui_oleh=$4, decided_at=now(), updated_at=now() WHERE tenant_id=$1 AND id=$2`,
			tid, id, status, appctx.UserID(ctx)); err != nil {
			return err
		}
		if err := audit.Write(ctx, tx, "dispensasi", id, map[bool]string{true: "approve", false: "reject"}[approve],
			map[string]string{"status": cur}, map[string]string{"status": status, "jenis": jenis}); err != nil {
			return err
		}
		var uid *string
		_ = tx.QueryRow(ctx, `SELECT user_id FROM mahasiswa WHERE id=$1 AND tenant_id=$2`, mhs, tid).Scan(&uid)
		if uid != nil {
			return notification.Send(ctx, tx, *uid, "Dispensasi "+status, "Pengajuan dispensasi "+jenis+" Anda "+status+".", notification.Link("/krs"))
		}
		return nil
	})
	if err != nil {
		return Dispensasi{}, err
	}
	return s.getDispensasi(ctx, id)
}

// ---- piutang (FR-28) ----

type PiutangRow struct {
	Group         string `json:"group"`
	TotalTagihan  int64  `json:"total_tagihan"`
	TotalTerbayar int64  `json:"total_terbayar"`
	Piutang       int64  `json:"piutang"`
}

func (s *Service) Piutang(ctx context.Context, group, periodeID string) ([]PiutangRow, error) {
	expr := `p.nama`
	if group == "angkatan" {
		expr = `m.angkatan::text`
	}
	rows, err := s.pool.Query(ctx, `SELECT `+expr+`, coalesce(sum(t.jumlah),0)::bigint, coalesce(sum(least(t.terbayar, t.jumlah)),0)::bigint,
		coalesce(sum(greatest(t.jumlah - t.terbayar, 0)),0)::bigint
		FROM tagihan t JOIN mahasiswa m ON m.id=t.mahasiswa_id JOIN prodi p ON p.id=m.prodi_id
		WHERE t.tenant_id=$1 AND ($2='' OR t.periode_id::text=$2) GROUP BY 1 ORDER BY 1`, appctx.TenantID(ctx), periodeID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[PiutangRow])
	if out == nil {
		out = []PiutangRow{}
	}
	return out, err
}
