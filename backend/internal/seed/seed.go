// Package seed builds the synthetic demo dataset (BRD §9): 2 tenants with overlapping
// prodi codes / NIMs, ~1,400 students, a finished and an active period, schedules,
// grades, bills, PMB applicants and deliberately bad records for PV-01..PV-08.
// Everything is generated in memory and bulk-loaded with COPY in one transaction.
package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/kampusops/backend/internal/domain"
	"github.com/kampusops/backend/internal/krs"
)

const Password = "password123"

type Summary struct {
	Tenants, Mahasiswa, Dosen, Kelas, KRS, KRSDetail, Nilai, Tagihan, Pendaftar int
}

func (s Summary) String() string {
	return fmt.Sprintf("tenants=%d mahasiswa=%d dosen=%d kelas=%d krs=%d krs_detail=%d nilai=%d tagihan=%d pendaftar=%d",
		s.Tenants, s.Mahasiswa, s.Dosen, s.Kelas, s.KRS, s.KRSDetail, s.Nilai, s.Tagihan, s.Pendaftar)
}

// ---- specs ----

type prodiSpec struct {
	kode, nama, jenjang, fak, prefix string
	masa                             int
	bank                             [][]string
	cohorts                          map[int]int
	ukt, prak                        int64
}

type tenantSpec struct {
	kode, nama, tz, va, domain string
	cutiMax                    int
	fakultas                   [][2]string
	prodi                      []prodiSpec
	plan                       string
}

var s1 = map[int]int{2019: 2, 2020: 4, 2021: 20, 2022: 45, 2023: 60, 2024: 70, 2025: 70}
var d3 = map[int]int{2021: 2, 2022: 4, 2023: 15, 2024: 70, 2025: 75}

var specs = []tenantSpec{
	{kode: "UNIKA", nama: "Universitas Nusantara Karya", tz: "Asia/Jakarta", va: "88081", domain: "unika.ac.id", cutiMax: 2, plan: "pro",
		fakultas: [][2]string{{"FT", "Fakultas Teknik"}, {"FEB", "Fakultas Ekonomi dan Bisnis"}},
		prodi: []prodiSpec{
			{kode: "11", nama: "Teknik Informatika", jenjang: "S1", fak: "FT", prefix: "IF", masa: 14, bank: bankTI, cohorts: s1, ukt: 7500000, prak: 750000},
			{kode: "12", nama: "Sistem Informasi", jenjang: "S1", fak: "FT", prefix: "SI", masa: 14, bank: bankSI, cohorts: s1, ukt: 7000000, prak: 500000},
			{kode: "21", nama: "Manajemen", jenjang: "S1", fak: "FEB", prefix: "MN", masa: 14, bank: bankMN, cohorts: s1, ukt: 6000000},
			{kode: "22", nama: "Akuntansi", jenjang: "D3", fak: "FEB", prefix: "AK", masa: 10, bank: bankAK, cohorts: d3, ukt: 5000000},
		}},
	{kode: "POLTEK", nama: "Politeknik Timur Raya", tz: "Asia/Makassar", va: "88092", domain: "poltek.ac.id", cutiMax: 1, plan: "basic",
		fakultas: [][2]string{{"FT", "Jurusan Teknik"}},
		prodi: []prodiSpec{
			{kode: "11", nama: "Teknik Informatika", jenjang: "D3", fak: "FT", prefix: "IF", masa: 10, bank: bankTID3, cohorts: d3, ukt: 5500000, prak: 600000},
			{kode: "13", nama: "Teknik Mesin", jenjang: "D3", fak: "FT", prefix: "TM", masa: 10, bank: bankTM, cohorts: d3, ukt: 6000000, prak: 900000},
		}},
}

// ---- in-memory model ----

// acc groups rows by table and column list (a table may be loaded with several column sets).
type acc struct {
	sets map[string][]string // table → ordered column-set keys
	rows map[string][][]any  // "table|cols" → rows
}

func (a *acc) add(table, cols string, vals ...any) {
	key := table + "|" + strings.ReplaceAll(cols, " ", "")
	if _, ok := a.rows[key]; !ok {
		a.sets[table] = append(a.sets[table], key)
	}
	a.rows[key] = append(a.rows[key], vals)
}

var flushOrder = []string{"users", "fakultas", "prodi", "user_role", "ruang", "dosen", "periode", "kurikulum", "mata_kuliah", "prasyarat",
	"ekuivalensi", "mahasiswa", "riwayat_status", "nim_sequence", "kelas", "kelas_dosen", "komponen_nilai", "krs", "krs_detail", "nilai",
	"koreksi_nilai", "tarif", "gelombang", "pendaftar", "tagihan", "cicilan", "pembayaran", "dispensasi", "notifikasi", "audit_log"}

type dosenM struct {
	id, user uuid.UUID
	nidn     string
	nama     string
	wali     bool
}

type mkM struct {
	id       uuid.UUID
	kode     string
	nama     string
	sem, idx int
	sks      int
}

type kurM struct {
	id     uuid.UUID
	nama   string
	mulai  int
	aktif  bool
	mks    map[[2]int]*mkM
	prodi  *prodiM
	runsIn map[string]bool // periode kode → classes opened
}

type prodiM struct {
	id    uuid.UUID
	spec  prodiSpec
	kurs  []*kurM
	dosen []*dosenM
	walis []*dosenM
	maxSem int
}

type periodeM struct {
	id     uuid.UUID
	kode   string
	tahun  int
	tipe   string
	status string
	tah    domain.Tahapan
	parity int // 1 odd semesters, 0 even
}

type kelasM struct {
	id               uuid.UUID
	per              *periodeM
	mk               *mkM
	kur              *kurM
	prodi            *prodiM
	nama             string
	sks              int
	dosen            *dosenM
	ruang            uuid.UUID
	hari, mulai, end int
	kuota, terisi    int
	fixedKuota       int
}

type studentM struct {
	id, user  uuid.UUID
	nim, nama string
	nik       *string
	prodi     *prodiM
	angkatan  int
	kur       *kurM
	wali      *dosenM
	status    string // current (20261)
	prevCuti  bool   // cuti in 20252
	lulus     bool
	ability   float64
	hist      map[[2]int]int // (sem, idx) → best bobot (from 20252)
	histNum   int64
	histDen   int64
	paid      string // lunas | sebagian | belum
	noKrs     bool
}

type seeder struct {
	ctx   context.Context
	now   time.Time
	hash  string
	rng   *rand.Rand
	a     *acc
	rules domain.Rules
	sum   Summary
	nikN  int
	vaN   int
}

func u() uuid.UUID { return uuid.New() }

func date(s string) time.Time {
	t, err := time.Parse(domain.DateLayout, s)
	if err != nil {
		panic(err)
	}
	return t
}

func ptr[T any](v T) *T { return &v }

func hm(min int) pgtype.Time { return pgtype.Time{Microseconds: int64(min) * 60 * 1_000_000, Valid: true} }

func jsonb(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// Run truncates all tenant data and seeds the demo dataset.
func Run(ctx context.Context, pool *pgxpool.Pool, now time.Time) (Summary, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(Password), bcrypt.DefaultCost)
	if err != nil {
		return Summary{}, err
	}
	sd := &seeder{ctx: ctx, now: now, hash: string(hash), rng: rand.New(rand.NewSource(20261)),
		a: &acc{sets: map[string][]string{}, rows: map[string][][]any{}}, rules: domain.DefaultRules()}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Summary{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `TRUNCATE tenant CASCADE`); err != nil {
		return Summary{}, err
	}
	for i, spec := range specs {
		tid := u()
		rules := domain.DefaultRules()
		rules.CutiMax = spec.cutiMax
		rules.VAPrefix = spec.va
		cfg := map[string]any{"sks_rules": rules.SKSRules, "sks_default_baru": rules.SKSDefaultBaru, "sks_pendek_max": rules.SKSPendekMax,
			"grade_scale": rules.GradeScale, "lulus_min_bobot_x100": rules.LulusMinBobotX100, "cuti_max": rules.CutiMax,
			"masa_studi_warning_sisa": rules.MasaStudiWarningSisa, "krs_grace_days": rules.KRSGraceDays, "nim_format": rules.NIMFormat,
			"min_peserta_kelas": rules.MinPesertaKelas, "biaya_cuti": rules.BiayaCuti, "va_prefix": rules.VAPrefix, "komponen_default": rules.KomponenDefault}
		if _, err := tx.Exec(ctx, `INSERT INTO tenant (id, kode, nama, timezone, plan, konfigurasi_aturan) VALUES ($1,$2,$3,$4,$5,$6)`,
			tid, spec.kode, spec.nama, spec.tz, spec.plan, jsonb(cfg)); err != nil {
			return Summary{}, err
		}
		sd.rules = rules
		sd.tenant(tid, i, spec)
		sd.sum.Tenants++
	}
	for _, t := range flushOrder {
		for _, key := range sd.a.sets[t] {
			cols := strings.Split(strings.SplitN(key, "|", 2)[1], ",")
			if _, err := tx.CopyFrom(ctx, pgx.Identifier{t}, cols, pgx.CopyFromRows(sd.a.rows[key])); err != nil {
				return Summary{}, fmt.Errorf("copy %s: %w", t, err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Summary{}, err
	}
	return sd.sum, nil
}

func (s *seeder) user(tid uuid.UUID, nama, email string, ident *string, roles ...any) uuid.UUID {
	id := u()
	s.a.add("users", "id,tenant_id,nama,email,identifier,password_hash", id, tid, nama, strings.ToLower(email), ident, s.hash)
	for i := 0; i < len(roles); i += 2 {
		var scope *uuid.UUID
		if roles[i+1] != nil {
			scope = roles[i+1].(*uuid.UUID)
		}
		s.a.add("user_role", "id,tenant_id,user_id,role,scope_prodi_id", u(), tid, id, roles[i].(string), scope)
	}
	return id
}

func (s *seeder) name() string {
	return firstNames[s.rng.Intn(len(firstNames))] + " " + lastNames[s.rng.Intn(len(lastNames))]
}

func (s *seeder) nik(ti int) *string {
	s.nikN++
	v := fmt.Sprintf("%d2%02d%012d", 3+ti, 71+ti, s.nikN)
	return &v
}

func (s *seeder) va(prefix string) string {
	s.vaN++
	return fmt.Sprintf("%s%011d", prefix, s.vaN)
}

func clipSem(semKe, maxSem int) int {
	if semKe <= maxSem {
		return semKe
	}
	// largest semester ≤ maxSem with the same parity
	if (maxSem-semKe)%2 == 0 {
		return maxSem
	}
	return maxSem - 1
}

var slotStarts = []int{7 * 60, 8*60 + 45, 10*60 + 30, 13 * 60, 14*60 + 45, 16*60 + 30}

const slotLen = 100

// ---- one tenant ----

func (s *seeder) tenant(tid uuid.UUID, ti int, spec tenantSpec) {
	a := s.a
	loc := domain.LoadLocation(spec.tz)
	today := s.now.In(loc)
	day := func(off int) string { return today.AddDate(0, 0, off).Format(domain.DateLayout) }

	// staff accounts
	super := uuid.Nil
	if ti == 0 {
		super = s.user(tid, "Super Admin KampusOps", "super@kampusops.id", nil, "super_admin", nil)
	}
	adminU := s.user(tid, "Admin "+spec.kode, "admin@"+spec.domain, nil, "admin_kampus", nil)
	baak := s.user(tid, "Operator BAAK "+spec.kode, "baak@"+spec.domain, nil, "operator", nil)
	keu := s.user(tid, "Bagian Keuangan "+spec.kode, "keuangan@"+spec.domain, nil, "keuangan", nil)
	_ = super

	// fakultas, prodi, dosen
	fak := map[string]uuid.UUID{}
	for _, f := range spec.fakultas {
		fak[f[0]] = u()
		a.add("fakultas", "id,tenant_id,kode,nama", fak[f[0]], tid, f[0], f[1])
	}
	var prodis []*prodiM
	for pi, ps := range spec.prodi {
		p := &prodiM{id: u(), spec: ps, maxSem: len(ps.bank)}
		prodis = append(prodis, p)
		a.add("prodi", "id,tenant_id,fakultas_id,kode,nama,jenjang,masa_studi_maks", p.id, tid, fak[ps.fak], ps.kode, ps.nama, ps.jenjang, ps.masa)
		for di := 0; di < 6; di++ {
			d := &dosenM{id: u(), nidn: fmt.Sprintf("0%d%s%04d", ti+1, ps.kode, di+1), nama: s.dosenName(), wali: true}
			roles := []any{"dosen", nil, "dosen_wali", nil}
			if ti == 0 && pi == 0 && di == 0 {
				d.nidn = "0011223344"
			}
			if ti == 0 && pi == 0 && di == 1 {
				d.nidn, d.wali, roles = "0055667788", false, []any{"dosen", nil}
			}
			if di == 0 {
				roles = append([]any{"kaprodi", &p.id}, roles...)
			}
			d.user = s.user(tid, d.nama, "dosen"+d.nidn+"@"+spec.domain, &d.nidn, roles...)
			a.add("dosen", "id,tenant_id,user_id,nidn,nama,email,prodi_id", d.id, tid, d.user, d.nidn, d.nama, "dosen"+d.nidn+"@"+spec.domain, p.id)
			p.dosen = append(p.dosen, d)
			if d.wali {
				p.walis = append(p.walis, d)
			}
			s.sum.Dosen++
		}
		// curricula: UNIKA TI has an old (2020) and a new (2024) curriculum (EC-05)
		mkKur := func(nama string, mulai int, aktif bool, prefix, suffix string) *kurM {
			k := &kurM{id: u(), nama: nama, mulai: mulai, aktif: aktif, mks: map[[2]int]*mkM{}, prodi: p, runsIn: map[string]bool{}}
			a.add("kurikulum", "id,tenant_id,prodi_id,nama,angkatan_mulai,aktif", k.id, tid, p.id, nama, mulai, aktif)
			sksPat := []int{3, 3, 3, 2, 2}
			for sem, names := range ps.bank {
				for idx, n := range names {
					m := &mkM{id: u(), kode: fmt.Sprintf("%s%d%d%s", prefix, sem+1, idx+1, suffix), nama: n, sem: sem + 1, idx: idx, sks: sksPat[idx]}
					k.mks[[2]int{sem + 1, idx}] = m
					a.add("mata_kuliah", "id,tenant_id,kurikulum_id,kode,nama,sks,semester,sifat", m.id, tid, k.id, m.kode, m.nama, m.sks, m.sem,
						map[bool]string{true: "pilihan", false: "wajib"}[sem >= 6 && idx == 4])
				}
			}
			for key, m := range k.mks {
				if key[0] < 2 || key[1] > 1 {
					continue
				}
				prev := k.mks[[2]int{key[0] - 1, key[1]}]
				var min *string
				if key[1] == 0 {
					min = ptr("C")
				}
				a.add("prasyarat", "id,tenant_id,mk_id,mk_syarat_id,nilai_min", u(), tid, m.id, prev.id, min)
			}
			p.kurs = append(p.kurs, k)
			return k
		}
		if ti == 0 && pi == 0 {
			old := mkKur("Kurikulum 2020", 2019, false, "T"+ps.prefix, "")
			nw := mkKur("Kurikulum 2024", 2024, true, ps.prefix, "")
			for key, m := range old.mks {
				a.add("ekuivalensi", "id,tenant_id,mk_lama_id,mk_baru_id", u(), tid, m.id, nw.mks[key].id)
			}
			old.runsIn["20252"] = true
			nw.runsIn["20252"], nw.runsIn["20261"] = true, true
		} else {
			k := mkKur("Kurikulum 2020", 2019, true, ps.prefix, "")
			k.runsIn["20252"], k.runsIn["20261"] = true, true
		}
		a.add("tarif", "id,tenant_id,prodi_id,angkatan,jenis,jumlah", u(), tid, p.id, 2019, "ukt", ps.ukt)
		if ps.prak > 0 {
			a.add("tarif", "id,tenant_id,prodi_id,angkatan,jenis,jumlah", u(), tid, p.id, 2019, "praktikum", ps.prak)
		}
		a.add("tarif", "id,tenant_id,prodi_id,angkatan,jenis,jumlah", u(), tid, p.id, 2019, "cuti", s.rules.BiayaCuti)
	}

	// rooms
	var rooms []uuid.UUID
	for r := 0; r < 10; r++ {
		id := u()
		rooms = append(rooms, id)
		gedung := "A"
		if r >= 5 {
			gedung = "B"
		}
		a.add("ruang", "id,tenant_id,kode,nama,kapasitas", id, tid, fmt.Sprintf("%s%d%02d", gedung, r%5/3+1, r%5+1), fmt.Sprintf("Ruang %s.%d", gedung, r%5+1), 40+(r%3)*5)
	}

	// periods: 20252 finished (exported), 20261 active with the KRS window open now (EC-17: tenant tz)
	past := &periodeM{id: u(), kode: "20252", tahun: 2025, tipe: "Genap", status: "selesai", parity: 0, tah: domain.Tahapan{
		KrsMulai: "2026-01-26", KrsSelesai: "2026-02-06", RevisiMulai: ptr("2026-02-16"), RevisiSelesai: ptr("2026-02-20"),
		KuliahMulai: "2026-02-09", KuliahSelesai: "2026-06-12", UTS: "2026-04-06", UAS: "2026-06-15", NilaiSelesai: "2026-07-03"}}
	act := &periodeM{id: u(), kode: "20261", tahun: 2026, tipe: "Ganjil", status: "aktif", parity: 1, tah: domain.Tahapan{
		KrsMulai: day(-3), KrsSelesai: day(7), RevisiMulai: ptr(day(10)), RevisiSelesai: ptr(day(14)),
		KuliahMulai: day(15), KuliahSelesai: day(120), UTS: day(60), UAS: day(125), NilaiSelesai: day(150)}}
	exported := date("2026-07-20")
	a.add("periode", "id,tenant_id,kode,tahun,tipe,status,tahapan,exported_at", past.id, tid, past.kode, past.tahun, past.tipe, past.status, jsonb(past.tah), &exported)
	a.add("periode", "id,tenant_id,kode,tahun,tipe,status,tahapan,exported_at", act.id, tid, act.kode, act.tahun, act.tipe, act.status, jsonb(act.tah), (*time.Time)(nil))

	// classes
	kelasPast := s.classes(prodis, past, rooms)
	kelasAct := s.classes(prodis, act, rooms)
	byKey := func(ks []*kelasM) map[string]*kelasM {
		m := map[string]*kelasM{}
		for _, k := range ks {
			m[fmt.Sprintf("%s|%d|%d|%s", k.kur.id, k.mk.sem, k.mk.idx, k.nama)] = k
		}
		return m
	}
	pastIdx, actIdx := byKey(kelasPast), byKey(kelasAct)
	var repeat *kelasM
	if ti == 0 { // EC-06: a repeat class of an even-semester course opened in the odd period
		p := prodis[0]
		k := p.kurs[1]
		repeat = &kelasM{id: u(), per: act, mk: k.mks[[2]int{4, 4}], kur: k, prodi: p, nama: "R", dosen: p.dosen[1], ruang: rooms[9], hari: 6, mulai: slotStarts[5], end: slotStarts[5] + slotLen}
		repeat.sks = repeat.mk.sks
		kelasAct = append(kelasAct, repeat)
	}

	// students
	var students []*studentM
	for _, p := range prodis {
		years := make([]int, 0)
		for y := range p.spec.cohorts {
			years = append(years, y)
		}
		sort.Ints(years)
		widx := 0
		for _, y := range years {
			n := p.spec.cohorts[y]
			for i := 1; i <= n; i++ {
				st := &studentM{id: u(), nim: fmt.Sprintf("%d%s%04d", y, p.spec.kode, i), nama: s.name(), prodi: p, angkatan: y,
					status: "aktif", ability: 74 + s.rng.NormFloat64()*8, hist: map[[2]int]int{}, paid: "lunas", nik: s.nik(ti)}
				st.kur = p.kurs[len(p.kurs)-1]
				if len(p.kurs) > 1 && y < 2024 {
					st.kur = p.kurs[0]
				}
				st.wali = p.walis[widx%len(p.walis)]
				widx++
				r := s.rng.Float64()
				switch {
				case y == 2021 && p.spec.jenjang == "S1" && i <= 5:
					st.lulus, st.status = true, "lulus"
				case r < 0.03:
					st.status = "cuti"
				case r < 0.05:
					st.status = "non_aktif"
				}
				if st.status == "aktif" {
					pr := s.rng.Float64()
					switch {
					case pr < 0.04:
						st.paid, st.noKrs = "belum", true
					case pr < 0.12:
						st.paid = "sebagian"
					}
					if s.rng.Float64() < 0.015 {
						st.noKrs = true // forgot to fill KRS → PV-01
					}
				}
				if ti == 1 && i == 3 && y == 2024 { // POLTEK: already took cuti once (cuti_max=1) → CUTI_LIMIT_REACHED demo
					st.prevCuti = true
				}
				students = append(students, st)
			}
			a.add("nim_sequence", "tenant_id,prodi_id,angkatan,last_no", tid, p.id, y, n)
		}
	}
	// demo students (UNIKA TI 2024): 2024110001 paid, 2024110002 unpaid; both advised by 0011223344
	var demo1, demo2 *studentM
	for _, st := range students {
		if st.nim == "2024110001" {
			demo1 = st
		}
		if st.nim == "2024110002" {
			demo2 = st
		}
	}
	for _, st := range []*studentM{demo1, demo2} {
		st.status, st.lulus, st.prevCuti = "aktif", false, false
		st.wali = st.prodi.walis[0]
		st.ability = 84
	}
	demo1.paid, demo1.noKrs = "lunas", true
	demo2.paid, demo2.noKrs = "belum", true
	if ti == 0 {
		demo1.nama, demo2.nama = "Andi Pratama", "Budi Santoso"
	}

	// ---- history: 20252 (grades) ----
	for _, st := range students {
		email := st.nim + "@student." + spec.domain
		st.user = s.user(tid, st.nama, email, &st.nim, "mahasiswa", nil)
		pastStatus := "aktif"
		if st.prevCuti {
			pastStatus = "cuti"
		}
		if st.lulus {
			pastStatus = "lulus"
		}
		if st.angkatan <= 2025 {
			a.add("riwayat_status", "id,tenant_id,mahasiswa_id,periode_id,status,alasan,created_at", u(), tid, st.id, past.id, pastStatus, (*string)(nil), date("2026-02-01"))
		}
		if !st.lulus {
			var alasan *string
			if st.status == "cuti" {
				alasan = ptr("Cuti akademik (alasan keluarga)")
			}
			if st.status == "non_aktif" {
				alasan = ptr("Tidak melakukan registrasi")
			}
			a.add("riwayat_status", "id,tenant_id,mahasiswa_id,periode_id,status,alasan,created_at", u(), tid, st.id, act.id, st.status, alasan, s.now.Add(-30*24*time.Hour))
		}
		if pastStatus != "aktif" && !st.lulus || st.angkatan > 2025 {
			continue
		}
		if st.lulus {
			continue
		}
		sem := clipSem(domain.SemesterKe(st.angkatan, 2025, "Genap"), st.prodi.maxSem)
		kur := st.kur
		if !kur.runsIn["20252"] {
			continue
		}
		krsID := u()
		group := "A"
		if s.rng.Intn(2) == 1 {
			group = "B"
		}
		var details int
		total := 0
		for idx := 0; idx < 5; idx++ {
			k := pastIdx[fmt.Sprintf("%s|%d|%d|%s", kur.id, sem, idx, group)]
			if k == nil {
				continue
			}
			did := u()
			k.terisi++
			total += k.sks
			a.add("krs_detail", "id,tenant_id,krs_id,kelas_id,mk_id,sks,is_ulang,status", did, tid, krsID, k.id, k.mk.id, k.sks, false, "aktif")
			score := st.ability + s.rng.NormFloat64()*7
			if st == demo1 {
				score = []float64{88, 82, 90, 77, 45}[idx]
			}
			score = math.Max(25, math.Min(98, score))
			ax := int(math.Round(score * 100))
			h, b := s.rules.Letter(ax)
			komp := map[string]int{"tugas": ax, "uts": ax, "uas": ax}
			nid := u()
			a.add("nilai", "id,tenant_id,krs_detail_id,mahasiswa_id,kelas_id,komponen,angka_x100,huruf,bobot_x100,final_at", nid, tid, did, st.id, k.id, jsonb(komp), ax, h, b, date("2026-07-01"))
			s.sum.Nilai++
			key := [2]int{sem, idx}
			if b > st.hist[key] || st.hist[key] == 0 {
				st.hist[key] = b
			}
			st.histNum += int64(k.sks) * int64(b)
			st.histDen += int64(k.sks)
			details++
			if k.dosen == st.prodi.dosen[1] && ti == 0 && st.prodi == prodis[0] && pendingKoreksi == uuid.Nil && h != "A" {
				pendingKoreksi = nid
				kh, kb := s.rules.Letter(ax + 800)
				a.add("koreksi_nilai", "id,tenant_id,nilai_id,komponen_lama,komponen_baru,angka_lama,angka_baru,huruf_lama,huruf_baru,bobot_lama,bobot_baru,alasan,status,diajukan_oleh",
					u(), tid, nid, jsonb(komp), jsonb(map[string]int{"tugas": ax + 800, "uts": ax + 800, "uas": ax + 800}), ax, ax+800, h, kh, b, kb,
					"Nilai UAS salah input (lembar jawaban halaman 2 belum dihitung)", "diajukan", k.dosen.user)
			}
		}
		if details > 0 {
			a.add("krs", "id,tenant_id,mahasiswa_id,periode_id,status,total_sks,max_sks,submitted_at,decided_at,decided_by", krsID, tid, st.id, past.id, "disetujui", total, 24,
				date("2026-02-01"), date("2026-02-03"), st.wali.user)
			s.sum.KRS++
			s.sum.KRSDetail += details
			a.add("tagihan", "id,tenant_id,mahasiswa_id,periode_id,jenis,jumlah,terbayar,status,jatuh_tempo", u(), tid, st.id, past.id, "ukt", st.prodi.spec.ukt, st.prodi.spec.ukt, "lunas", date("2026-02-06"))
			s.sum.Tagihan++
		}
	}
	pendingKoreksi = uuid.Nil

	// ---- active period 20261: bills + KRS ----
	pv06Done, pv07Cuti, pv07Non := false, false, false
	for _, st := range students {
		if st.lulus {
			continue
		}
		semKe := domain.SemesterKe(st.angkatan, 2026, "Ganjil")
		var ipsLalu *string
		var prev *int
		if st.histDen > 0 {
			v := domain.IPx100(st.histNum, st.histDen)
			prev = &v
			ipsLalu = ptr(domain.FormatIP(st.histNum, st.histDen))
		}
		maxSks := krs.MaxSKS(s.rules, "Ganjil", prev)
		// bills
		switch st.status {
		case "aktif":
			s.bills(tid, st, act, spec.va)
		case "cuti":
			a.add("tagihan", "id,tenant_id,mahasiswa_id,periode_id,jenis,jumlah,terbayar,status,jatuh_tempo,va_number", u(), tid, st.id, act.id, "cuti",
				s.rules.BiayaCuti, int64(0), "belum_bayar", date(act.tah.KrsSelesai), ptr(s.va(spec.va)))
			s.sum.Tagihan++
		}
		wantKrs := st.status == "aktif" && !st.noKrs
		// PV-07: one cuti and one non_aktif student that still have a KRS
		if ti == 0 && ((st.status == "cuti" && !pv07Cuti) || (st.status == "non_aktif" && !pv07Non)) {
			wantKrs = true
		}
		if !wantKrs {
			continue
		}
		sem := clipSem(semKe, st.prodi.maxSem)
		kur := st.prodi.kurs[len(st.prodi.kurs)-1] // only the active curriculum runs in 20261
		group := "A"
		if s.rng.Intn(2) == 1 {
			group = "B"
		}
		type pick struct {
			k     *kelasM
			ulang bool
		}
		var picks []pick
		for idx := 0; idx < 5; idx++ {
			if sem >= 2 && idx <= 1 { // prerequisite (s-1, idx) — through ekuivalensi for old-curriculum students (same key)
				b := st.hist[[2]int{sem - 1, idx}]
				min := s.rules.LulusMinBobotX100
				if idx == 0 {
					min = 200
				}
				if b < min {
					continue
				}
			}
			if k := actIdx[fmt.Sprintf("%s|%d|%d|%s", kur.id, sem, idx, group)]; k != nil {
				picks = append(picks, pick{k: k})
			}
		}
		pv06 := false
		if ti == 0 && !pv06Done && st.prodi == prodis[0] && st.angkatan == 2023 && st.status == "aktif" {
			// PV-06: KRS above the SKS limit without dispensation
			for _, key := range [][2]int{{1, 2}, {1, 3}, {1, 4}, {3, 2}, {3, 3}, {3, 4}} {
				if k := actIdx[fmt.Sprintf("%s|%d|%d|%s", kur.id, key[0], key[1], group)]; k != nil {
					picks = append(picks, pick{k: k})
				}
			}
			pv06, pv06Done = true, true
		}
		if len(picks) == 0 {
			continue
		}
		if st.status == "cuti" {
			pv07Cuti = true
		}
		if st.status == "non_aktif" {
			pv07Non = true
		}
		krsID := u()
		total := 0
		for _, pk := range picks {
			pk.k.terisi++
			total += pk.k.sks
			a.add("krs_detail", "id,tenant_id,krs_id,kelas_id,mk_id,sks,is_ulang,status", u(), tid, krsID, pk.k.id, pk.k.mk.id, pk.k.sks, pk.ulang, "aktif")
			s.sum.KRSDetail++
		}
		if pv06 {
			maxSks = 24
		}
		status := "disetujui"
		r := s.rng.Float64()
		switch {
		case st.status != "aktif":
			status = "draft"
		case r < 0.25:
			status = "diajukan"
		case r < 0.42:
			status = "draft"
		}
		var sub, dec *time.Time
		var decBy *uuid.UUID
		if status != "draft" {
			sub = ptr(s.now.Add(-time.Duration(30+s.rng.Intn(40)) * time.Hour))
		}
		if status == "disetujui" {
			dec, decBy = ptr(sub.Add(6*time.Hour)), &st.wali.user
		}
		a.add("krs", "id,tenant_id,mahasiswa_id,periode_id,status,total_sks,max_sks,ips_lalu,submitted_at,decided_at,decided_by", krsID, tid, st.id, act.id, status, total, maxSks, ipsLalu, sub, dec, decBy)
		s.sum.KRS++
	}

	// ---- deliberately bad records for the validator (UNIKA, active period) ----
	if ti == 0 {
		var aktif []*studentM
		for _, st := range students {
			if st.status == "aktif" && st != demo1 && st != demo2 && !st.noKrs {
				aktif = append(aktif, st)
			}
		}
		// PV-05: empty NIK, malformed NIK, duplicated NIK
		aktif[3].nik = nil
		aktif[7].nik = ptr("12345")
		aktif[11].nik = aktif[12].nik
		// PV-02: grade without a matching KRS row
		var target *kelasM
		for _, k := range kelasAct {
			if k.prodi == prodis[0] && k.mk.sem == 7 && k.nama == "B" {
				target = k
				break
			}
		}
		victim := aktif[20]
		a.add("nilai", "id,tenant_id,krs_detail_id,mahasiswa_id,kelas_id,komponen,angka_x100,huruf,bobot_x100", u(), tid, (*uuid.UUID)(nil), victim.id, target.id,
			jsonb(map[string]int{"tugas": 8000}), (*int)(nil), (*string)(nil), (*int)(nil))
		// PV-03: class without lecturer; PV-04: class SKS differs from course SKS
		for _, k := range kelasAct {
			if k.prodi == prodis[0] && k.mk.sem == 7 && k.mk.idx == 4 && k.nama == "B" {
				k.dosen = nil
			}
			if k.prodi == prodis[1] && k.mk.sem == 3 && k.mk.idx == 2 && k.nama == "A" {
				k.sks = k.mk.sks - 1
			}
		}
		// PV-08: corrected grade flagged for re-reporting (record a detail id via a fresh enrolment)
		var flagged *kelasM
		for _, k := range kelasAct {
			if k.prodi == prodis[0] && k.mk.sem == 5 && k.mk.idx == 3 && k.nama == "A" {
				flagged = k
			}
		}
		var st *studentM
		for _, x := range students {
			if x.status == "aktif" && x.noKrs && x.paid == "lunas" && x != demo1 && x != demo2 {
				st = x
				break
			}
		}
		krsID, did := u(), u()
		flagged.terisi++
		a.add("krs", "id,tenant_id,mahasiswa_id,periode_id,status,total_sks,max_sks,submitted_at,decided_at,decided_by", krsID, tid, st.id, act.id, "disetujui", flagged.sks, 24,
			s.now.Add(-50*time.Hour), s.now.Add(-40*time.Hour), st.wali.user)
		a.add("krs_detail", "id,tenant_id,krs_id,kelas_id,mk_id,sks,is_ulang,status", did, tid, krsID, flagged.id, flagged.mk.id, flagged.sks, false, "aktif")
		a.add("nilai", "id,tenant_id,krs_detail_id,mahasiswa_id,kelas_id,komponen,angka_x100,huruf,bobot_x100,final_at,perlu_lapor_ulang", u(), tid, did, st.id, flagged.id,
			jsonb(map[string]int{"tugas": 8200, "uts": 8200, "uas": 8200}), 8200, "A-", 370, s.now.Add(-24*time.Hour), true)
		s.sum.KRS++
		s.sum.KRSDetail++
		// demo student 1 failed "Metode Numerik" (D) in 20252 → may retake class R (is_ulang)
		_ = repeat
		// EC-01 favourite class: last seat left
		for _, k := range kelasAct {
			if k.prodi == prodis[0] && k.mk.sem == 5 && k.mk.idx == 1 && k.nama == "A" {
				k.fixedKuota = k.terisi + 1
			}
		}
	}

	// ---- students rows ----
	for _, st := range students {
		num, den, lulus := s.rules.Cumulative(histEntries(st), nil)
		status := st.status
		if st.lulus {
			status = "lulus"
		}
		a.add("mahasiswa", "id,tenant_id,user_id,nim,nik,nama,email,prodi_id,angkatan,kurikulum_id,dosen_wali_id,status,ipk_num,ipk_den,sks_lulus",
			st.id, tid, st.user, st.nim, st.nik, st.nama, st.nim+"@student."+spec.domain, st.prodi.id, st.angkatan, st.kur.id, st.wali.id, status, num, den, lulus)
		s.sum.Mahasiswa++
	}

	// ---- class rows ----
	for _, ks := range [][]*kelasM{kelasPast, kelasAct} {
		for _, k := range ks {
			k.kuota = 40
			if k.terisi+3 > k.kuota {
				k.kuota = k.terisi + 3 + s.rng.Intn(5)
			}
			if k.fixedKuota > 0 {
				k.kuota = k.fixedKuota
			}
			var dosen *uuid.UUID
			if k.dosen != nil {
				dosen = &k.dosen.id
				a.add("kelas_dosen", "id,tenant_id,kelas_id,dosen_id,efektif_mulai", u(), tid, k.id, k.dosen.id, date("2000-01-01"))
			}
			a.add("kelas", "id,tenant_id,periode_id,mk_id,nama,sks,dosen_id,ruang_id,hari,jam_mulai,jam_selesai,kuota,terisi,status",
				k.id, tid, k.per.id, k.mk.id, k.nama, k.sks, dosen, k.ruang, k.hari, hm(k.mulai), hm(k.end), k.kuota, k.terisi, "dibuka")
			for i, c := range s.rules.KomponenDefault {
				a.add("komponen_nilai", "id,tenant_id,kelas_id,key,label,bobot_persen,urutan", u(), tid, k.id, c.Key, c.Label, c.BobotPersen, i)
			}
			s.sum.Kelas++
		}
	}

	// ---- PMB ----
	s.pmb(tid, ti, spec, prodis, day)

	// ---- dispensasi + notifications ----
	n := 0
	for _, st := range students {
		if st.paid == "belum" && st != demo2 && n < 4 {
			a.add("dispensasi", "id,tenant_id,mahasiswa_id,periode_id,jenis,alasan,status,created_by", u(), tid, st.id, act.id, "penundaan",
				"Orang tua terdampak PHK; mohon penundaan pembayaran UKT", "diajukan", st.user)
			n++
		}
	}
	note := func(user uuid.UUID, judul, isi, link string) {
		a.add("notifikasi", "id,tenant_id,user_id,judul,isi,link", u(), tid, user, judul, isi, &link)
	}
	note(demo1.user, "Masa KRS dibuka", "Pengisian KRS "+act.kode+" dibuka sampai "+act.tah.KrsSelesai+".", "/krs")
	note(demo2.user, "Tagihan UKT belum dibayar", "Lunasi UKT agar dapat mengisi KRS.", "/tagihan")
	note(baak, "Validasi PDDikti", "Jalankan validator sebelum batas pelaporan.", "/pddikti")
	note(keu, "Pengajuan dispensasi", "Ada pengajuan penundaan pembayaran baru.", "/dispensasi")
	a.add("audit_log", "id,tenant_id,actor_id,actor_nama,actor_role,entitas,entitas_id,aksi,after,request_id", u(), tid, &adminU, "Admin "+spec.kode, "admin_kampus",
		"tenant", tid, "seed", jsonb(map[string]any{"mahasiswa": len(students)}), "seed")
}

var pendingKoreksi uuid.UUID

func histEntries(st *studentM) []domain.GradeEntry {
	out := []domain.GradeEntry{}
	for key, b := range st.hist {
		sks := []int{3, 3, 3, 2, 2}[key[1]]
		out = append(out, domain.GradeEntry{MkID: fmt.Sprintf("%d-%d", key[0], key[1]), Sks: sks, BobotX100: b})
	}
	return out
}

func (s *seeder) dosenName() string {
	g := gelar[s.rng.Intn(len(gelar))]
	n := s.name()
	if g != "" {
		n = g + " " + n
	}
	return n + []string{", M.Kom.", ", M.T.", ", M.M.", ", M.Si.", ", M.Ak."}[s.rng.Intn(5)]
}

// bills creates UKT (+ praktikum) bills of the active period with payments per st.paid.
func (s *seeder) bills(tid uuid.UUID, st *studentM, act *periodeM, vaPrefix string) {
	type bill struct {
		jenis  string
		jumlah int64
	}
	bs := []bill{{"ukt", st.prodi.spec.ukt}}
	if st.prodi.spec.prak > 0 {
		bs = append(bs, bill{"praktikum", st.prodi.spec.prak})
	}
	due := date(act.tah.KrsSelesai)
	for _, b := range bs {
		id := u()
		var paid int64
		status := "lunas"
		var va *string
		switch {
		case st.paid == "belum":
			status, va = "belum_bayar", ptr(s.va(vaPrefix))
		case st.paid == "sebagian" && b.jenis == "ukt":
			paid, status, va = b.jumlah/2, "sebagian", ptr(s.va(vaPrefix))
		default:
			paid = b.jumlah
		}
		s.a.add("tagihan", "id,tenant_id,mahasiswa_id,periode_id,jenis,jumlah,terbayar,status,jatuh_tempo,va_number", id, tid, st.id, act.id, b.jenis, b.jumlah, paid, status, due, va)
		s.sum.Tagihan++
		if paid > 0 {
			s.a.add("pembayaran", "id,tenant_id,tagihan_id,transaction_id,jumlah,paid_at", u(), tid, id, "SEED-"+uuid.NewString()[:13], paid, s.now.Add(-time.Duration(48+s.rng.Intn(400))*time.Hour))
		}
		if status == "sebagian" { // FR-27: approved installment plan, first installment paid
			s.a.add("cicilan", "id,tenant_id,tagihan_id,ke,jumlah,jatuh_tempo", u(), tid, id, 1, b.jumlah/2, due)
			s.a.add("cicilan", "id,tenant_id,tagihan_id,ke,jumlah,jatuh_tempo", u(), tid, id, 2, b.jumlah-b.jumlah/2, due.AddDate(0, 2, 0))
		}
	}
}

// classes opens two classes (A, B) for each course of the period's semester parity and
// assigns slots/rooms/lecturers without room, lecturer or same-semester clashes.
func (s *seeder) classes(prodis []*prodiM, per *periodeM, rooms []uuid.UUID) []*kelasM {
	queues := make([][]*kelasM, len(prodis))
	for pi, p := range prodis {
		for _, letter := range []string{"A", "B"} {
			for idx := 0; idx < 5; idx++ {
				for _, k := range p.kurs {
					if !k.runsIn[per.kode] {
						continue
					}
					for sem := 1; sem <= p.maxSem; sem++ {
						if sem%2 != per.parity {
							continue
						}
						m := k.mks[[2]int{sem, idx}]
						queues[pi] = append(queues[pi], &kelasM{id: u(), per: per, mk: m, kur: k, prodi: p, nama: letter, sks: m.sks})
					}
				}
			}
		}
	}
	var out []*kelasM
	slot := 0
	for {
		left := 0
		for _, q := range queues {
			left += len(q)
		}
		if left == 0 {
			break
		}
		room := 0
		for pi := range queues {
			for k := 0; k < 3 && len(queues[pi]) > 0 && room < len(rooms); k++ {
				c := queues[pi][0]
				queues[pi] = queues[pi][1:]
				c.hari = slot/len(slotStarts) + 1
				c.mulai = slotStarts[slot%len(slotStarts)]
				c.end = c.mulai + slotLen
				c.ruang = rooms[room]
				c.dosen = prodis[pi].dosen[(k+slot*3)%len(prodis[pi].dosen)]
				room++
				out = append(out, c)
			}
		}
		slot++
		if slot >= 6*len(slotStarts)-1 { // keep the last slot free (repeat class)
			panic("seed: not enough slots for classes")
		}
	}
	return out
}

func (s *seeder) pmb(tid uuid.UUID, ti int, spec tenantSpec, prodis []*prodiM, day func(int) string) {
	a := s.a
	g1, g2 := u(), u()
	statuses := []string{"daftar", "bayar_formulir", "lengkapi_berkas", "verifikasi", "lulus", "lulus", "tidak_lulus", "verifikasi", "lengkapi_berkas", "lulus"}
	count := 40
	if ti == 1 {
		count = 15
	}
	terisi1, terisi2 := 0, 0
	seq := 0
	mk := func(gel uuid.UUID, nama, email, status string, prodi *prodiM, biaya int64) {
		seq++
		id := u()
		no := fmt.Sprintf("PMB2026%05d", seq)
		user := s.user(tid, nama, email, &no, "calon_mahasiswa", nil)
		berkas := []map[string]any{{"nama": "KTP", "ok": false}, {"nama": "Ijazah / SKL", "ok": false}, {"nama": "Pas foto", "ok": false}}
		if status == "verifikasi" || status == "lulus" || status == "tidak_lulus" {
			for _, b := range berkas {
				b["ok"] = true
			}
		}
		a.add("pendaftar", "id,tenant_id,user_id,no_daftar,nama,email,nik,gelombang_id,prodi_pilihan_id,status,berkas", id, tid, user, no, nama, strings.ToLower(email),
			*s.nik(ti), gel, prodi.id, status, jsonb(berkas))
		tstatus, paid := "lunas", biaya
		if status == "daftar" {
			tstatus, paid = "belum_bayar", 0
		}
		tagID := u()
		a.add("tagihan", "id,tenant_id,pendaftar_id,jenis,jumlah,terbayar,status,jatuh_tempo,va_number", tagID, tid, id, "formulir_pmb", biaya, paid, tstatus,
			date(day(7)), ptr(spec.va+"9"+no[3:]))
		if paid > 0 {
			a.add("pembayaran", "id,tenant_id,tagihan_id,transaction_id,jumlah,paid_at", u(), tid, tagID, "SEED-"+uuid.NewString()[:13], paid, s.now.Add(-72*time.Hour))
		}
		s.sum.Pendaftar++
		s.sum.Tagihan++
		if gel == g1 {
			terisi1++
		} else {
			terisi2++
		}
	}
	if ti == 0 {
		mk(g2, "Calon Mahasiswa Demo", "calon@mail.com", "daftar", prodis[0], 300000)
	}
	for i := 0; i < count; i++ {
		gel, biaya := g2, int64(300000)
		if i%3 == 0 {
			gel, biaya = g1, 250000
		}
		mk(gel, s.name(), fmt.Sprintf("pendaftar%03d@mail.com", i+1), statuses[i%len(statuses)], prodis[i%len(prodis)], biaya)
	}
	a.add("gelombang", "id,tenant_id,nama,mulai,selesai,kuota,terisi,biaya_formulir", g1, tid, "Gelombang 1 2026", date("2026-01-05"), date("2026-03-31"), 200, terisi1, int64(250000))
	a.add("gelombang", "id,tenant_id,nama,mulai,selesai,kuota,terisi,biaya_formulir", g2, tid, "Gelombang 2 2026", date(day(-30)), date(day(30)), 300, terisi2, int64(300000))
}
