# KampusOps — API Contract (v1)

Single source of truth between `backend/` (Go Fiber) and `frontend/` (Vue 3).
Both sides MUST follow this file exactly. Base URL: `/api/v1`. All JSON is `snake_case`.

## Conventions

- **Success**: `{ "data": <payload>, "meta"?: { "page": 1, "per_page": 20, "total": 123 } }`
- **Error**: HTTP 4xx/5xx with `{ "error": { "code": "KRS_QUOTA_FULL", "message": "Kelas sudah penuh", "details"?: any } }`
  - Validation: `422` code `VALIDATION_ERROR`, `details: { field: "message" }`
  - Business-rule violation: `422` with a specific code (see list below), `details` optional
  - Conflict (schedule clash, duplicate): `409`
  - Not found / cross-tenant access: `404` (never 403 for cross-tenant — EC-16)
- **Auth**: `Authorization: Bearer <access_token>`; access token 15 min, refresh rotation.
- **Active role**: header `X-Active-Role: <role>`; server checks the user actually holds it.
- **Idempotency**: mutating KRS + payment endpoints accept `Idempotency-Key` header (EC-02).
- **Lists** accept `?q=&page=&per_page=` (default per_page 20, max 100).
- **Money**: integer rupiah (`int64`). **Grades**: `bobot_x100` integer (e.g. 375 = 3.75), IPS/IPK returned as string with 2 decimals (`"3.42"`) (EC-10).
- **Timestamps**: RFC3339 UTC. Dates: `YYYY-MM-DD`. Tenant timezone in `tenant.timezone`.

### Roles (enum `Role`)
`super_admin | admin_kampus | operator | kaprodi | dosen_wali | dosen | keuangan | mahasiswa | calon_mahasiswa`

### Error codes (business)
`AUTH_INVALID_CREDENTIALS, AUTH_LOCKED, AUTH_TOKEN_EXPIRED, FORBIDDEN_ROLE,
KRS_WINDOW_CLOSED, KRS_STUDENT_NOT_ACTIVE, KRS_UNPAID, KRS_QUOTA_FULL, KRS_SKS_LIMIT,
KRS_PREREQ_NOT_MET, KRS_SCHEDULE_CLASH, KRS_INVALID_STATE, KRS_DUPLICATE_COURSE,
SCHEDULE_ROOM_CLASH, SCHEDULE_LECTURER_CLASH, GRADE_LOCKED, GRADE_NOT_ALLOWED,
PERIOD_ALREADY_ACTIVE, CUTI_LIMIT_REACHED, PMB_INVALID_TRANSITION, PDDIKTI_BLOCKED`

---

## Shared types (TypeScript notation)

```ts
type ID = string // uuid

interface Tenant { id: ID; code: string; name: string; timezone: string }
interface UserRole { role: Role; scope_prodi_id: ID | null; scope_prodi_name: string | null }
interface User { id: ID; name: string; email: string; identifier: string; roles: UserRole[] }

interface Fakultas { id: ID; kode: string; nama: string }
interface Prodi { id: ID; fakultas_id: ID; fakultas_nama: string; kode: string; nama: string;
  jenjang: 'D3'|'S1'|'S2'; masa_studi_maks: number }
interface Dosen { id: ID; nidn: string; nama: string; email: string; prodi_id: ID; prodi_nama: string }
interface Ruang { id: ID; kode: string; nama: string; kapasitas: number }

type PeriodeTipe = 'Ganjil'|'Genap'|'Pendek'
interface Tahapan { krs_mulai: string; krs_selesai: string; revisi_mulai: string|null; revisi_selesai: string|null;
  kuliah_mulai: string; kuliah_selesai: string; uts: string; uas: string; nilai_selesai: string } // dates
interface Periode { id: ID; kode: string /* 20261 */; tahun: number; tipe: PeriodeTipe;
  status: 'draft'|'aktif'|'selesai'; tahapan: Tahapan }

type StatusMhs = 'aktif'|'cuti'|'non_aktif'|'lulus'|'keluar'|'do'
interface Mahasiswa { id: ID; nim: string; nik: string|null; nama: string; email: string;
  prodi_id: ID; prodi_nama: string; angkatan: number; kurikulum_id: ID;
  dosen_wali_id: ID|null; dosen_wali_nama: string|null; status: StatusMhs;
  ipk: string; sks_lulus: number; semester_ke: number }
interface RiwayatStatus { periode_kode: string; status: StatusMhs; alasan: string|null; created_at: string }

interface Kurikulum { id: ID; prodi_id: ID; nama: string; angkatan_mulai: number; aktif: boolean }
interface MataKuliah { id: ID; kurikulum_id: ID; kode: string; nama: string; sks: number; semester: number;
  sifat: 'wajib'|'pilihan'; prasyarat: { mk_id: ID; kode: string; nama: string; nilai_min: string|null }[] }

interface Slot { hari: 1|2|3|4|5|6; jam_mulai: string /* "07:30" */; jam_selesai: string }
interface Kelas { id: ID; periode_id: ID; mk_id: ID; mk_kode: string; mk_nama: string; sks: number;
  nama: string /* "A" */; dosen_id: ID|null; dosen_nama: string|null; ruang_id: ID|null; ruang_nama: string|null;
  slot: Slot; kuota: number; terisi: number; status: 'dibuka'|'dibatalkan' }

type KrsStatus = 'draft'|'diajukan'|'disetujui'|'ditolak'
interface KrsItem { kelas: Kelas; is_ulang: boolean }
interface Krs { id: ID; mahasiswa: Pick<Mahasiswa,'id'|'nim'|'nama'|'prodi_nama'|'angkatan'|'ipk'>;
  periode_id: ID; status: KrsStatus; total_sks: number; max_sks: number; ips_lalu: string|null;
  catatan: string|null; submitted_at: string|null; decided_at: string|null; items: KrsItem[] }
interface Offering extends Kelas { eligible: boolean; reasons: string[] /* error codes */; in_krs: boolean }

interface Pertemuan { id: ID; kelas_id: ID; ke: number; tanggal: string; topik: string; dosen_nama: string;
  rekap: { hadir: number; izin: number; sakit: number; alpa: number } }
type PresensiStatus = 'H'|'I'|'S'|'A'
interface PresensiRow { mahasiswa_id: ID; nim: string; nama: string; status: PresensiStatus|null }

interface KomponenNilai { key: string; label: string; bobot_persen: number }
interface NilaiRow { krs_detail_id: ID; nilai_id: ID|null; mahasiswa_id: ID; nim: string; nama: string;
  komponen: Record<string, number|null>; angka: string|null; huruf: string|null; bobot_x100: number|null;
  final: boolean; koreksi_pending: boolean }
interface KoreksiNilai { id: ID; nilai_id: ID; kelas_nama: string; mk_nama: string; mahasiswa_nama: string; nim: string;
  huruf_lama: string; huruf_baru: string; angka_lama: string; angka_baru: string; alasan: string;
  status: 'diajukan'|'disetujui'|'ditolak'; diajukan_oleh: string; created_at: string }

interface KhsRow { mk_kode: string; mk_nama: string; sks: number; huruf: string|null; bobot_x100: number|null; is_ulang: boolean }
interface Khs { periode: Periode; rows: KhsRow[]; total_sks: number; ips: string; ipk: string }
interface Transkrip { mahasiswa: Mahasiswa; rows: (KhsRow & { semester: number; periode_kode: string })[];
  total_sks: number; ipk: string }
interface JadwalItem { kelas: Kelas }

type TagihanStatus = 'belum_bayar'|'sebagian'|'lunas'
interface Tagihan { id: ID; mahasiswa_id: ID; nim: string; nama: string; prodi_nama: string; periode_kode: string;
  jenis: 'ukt'|'praktikum'|'cuti'|'formulir_pmb'; jumlah: number; terbayar: number; status: TagihanStatus;
  jatuh_tempo: string; va_number: string|null; cicilan: { ke: number; jumlah: number; jatuh_tempo: string; lunas: boolean }[] }
interface Pembayaran { id: ID; tagihan_id: ID; transaction_id: string; jumlah: number; paid_at: string }
interface Dispensasi { id: ID; mahasiswa_id: ID; nim: string; nama: string; periode_kode: string;
  jenis: 'cicilan'|'penundaan'|'krs_susulan'; alasan: string; status: 'diajukan'|'disetujui'|'ditolak';
  disetujui_oleh: string|null; created_at: string }

type PmbStatus = 'daftar'|'bayar_formulir'|'lengkapi_berkas'|'verifikasi'|'lulus'|'tidak_lulus'|'daftar_ulang'
interface Gelombang { id: ID; nama: string; mulai: string; selesai: string; kuota: number; terisi: number; biaya_formulir: number }
interface Pendaftar { id: ID; no_daftar: string; nama: string; email: string; nik: string; gelombang_id: ID; gelombang_nama: string;
  prodi_pilihan_id: ID; prodi_pilihan_nama: string; status: PmbStatus; berkas: { nama: string; ok: boolean }[];
  nim: string|null; created_at: string }

interface ValidationIssue { id: ID; rule: string /* PV-01 */; level: 'error'|'warning'; entity: string;
  entity_id: ID; entity_label: string; message: string; fix_url: string /* SPA route, e.g. /mahasiswa/<id> */ }
interface ValidationResult { periode_id: ID; score: number /* 0-100 */; errors: number; warnings: number;
  ran_at: string; by_rule: { rule: string; label: string; level: 'error'|'warning'; count: number }[];
  items: ValidationIssue[] }

interface AuditEntry { id: ID; actor_nama: string; actor_role: Role; entitas: string; entitas_id: ID; aksi: string;
  before: unknown; after: unknown; created_at: string; request_id: string }

interface Notifikasi { id: ID; judul: string; isi: string; link: string|null; read: boolean; created_at: string }

// Dashboard is generic so every role renders with the same components
interface StatCard { key: string; label: string; value: string; hint: string|null;
  tone: 'blue'|'green'|'amber'|'red'|'violet'|'pink'; trend: 'up'|'down'|null }
interface Alert { level: 'info'|'warning'|'error'; message: string; link: string|null }
interface Dashboard { stats: StatCard[]; alerts: Alert[];
  chart: { title: string; labels: string[]; series: { name: string; values: number[] }[] } | null;
  table: { title: string; columns: { key: string; label: string }[]; rows: Record<string,string>[]; link: string|null } | null }
```

---

## Endpoints

### Platform
| Method | Path | Notes |
|---|---|---|
| GET | `/healthz`, `/readyz` | outside `/api/v1` |

### Auth (FR-01..04)
| Method | Path | Body → Response |
|---|---|---|
| POST | `/auth/login` | `{ tenant_code, identifier, password }` → `{ access_token, refresh_token, expires_in, user: User, tenant: Tenant }` (identifier = email / NIM / NIDN). 5 fails → `AUTH_LOCKED` |
| POST | `/auth/refresh` | `{ refresh_token }` → same as login (rotated) |
| POST | `/auth/logout` | `{ refresh_token }` → `204` |
| GET | `/me` | → `{ user: User, tenant: Tenant }` |
| GET | `/tenants/public` | no auth → `Tenant[]` (for login tenant picker) |

### Dashboard
| GET | `/dashboard` | uses `X-Active-Role` → `Dashboard` |

### Tenants (super_admin)
| GET/POST | `/tenants` | `Tenant & { mahasiswa_count, plan }` |

### Master data (admin_kampus, operator; read for others where sensible) (FR-05, FR-06)
CRUD = `GET list`, `GET /:id`, `POST`, `PUT /:id`, `DELETE /:id`
- `/fakultas`, `/prodi`, `/dosen`, `/ruang`, `/periode`, `/mahasiswa`
- `/mahasiswa?prodi_id=&angkatan=&status=`
- `GET /mahasiswa/:id/riwayat-status` → `RiwayatStatus[]`
- `POST /mahasiswa/:id/status` `{ status, alasan }` (cuti limit EC-08 → `CUTI_LIMIT_REACHED`)
- `POST /periode/:id/activate` (only one active, FR-06)
- `GET /periode/active` → `Periode`
- `GET /users`, `POST /users`, `PUT /users/:id/roles` `{ roles: UserRole[] }` (admin_kampus)

### Kurikulum (kaprodi, operator) (FR-11..13)
- `GET /kurikulum?prodi_id=` → `Kurikulum[]`
- `GET /kurikulum/:id/mata-kuliah` → `MataKuliah[]`
- `POST /mata-kuliah`, `PUT /mata-kuliah/:id` (body includes `prasyarat: {mk_id, nilai_min}[]`)
- `GET /ekuivalensi?kurikulum_id=` → `{ id, mk_lama_id, mk_lama_kode, mk_lama_nama, mk_baru_id, mk_baru_kode, mk_baru_nama }[]` (rows whose `mk_baru` belongs to that kurikulum); `POST /ekuivalensi` `{ mk_lama_id, mk_baru_id }`
- `POST /kelas` body: `{ periode_id, mk_id, nama, dosen_id|null, ruang_id|null, slot: Slot, kuota }` (extra fields like `prodi_id` are ignored)

### Kelas & jadwal (operator, kaprodi) (FR-14, FR-15, EC-12, EC-13)
- `GET /kelas?periode_id=&prodi_id=&q=` → `Kelas[]`
- `GET /kelas/:id` → `Kelas` (any staff/dosen role; used by class detail page and PDDikti fix links)
- `POST /kelas`, `PUT /kelas/:id` → 409 `SCHEDULE_ROOM_CLASH|SCHEDULE_LECTURER_CLASH` with `details: { conflicts: Kelas[] }`
- `POST /kelas/:id/cancel` `{ alasan }` → cancels related KRS items, notifies (EC-13)
- `POST /kelas/:id/ganti-dosen` `{ dosen_id, efektif: date }` (EC-12)

### KRS — mahasiswa (FR-16..20, EC-01, EC-02, EC-07)
- `GET /krs/me` → `{ krs: Krs|null, window: { open: boolean, phase: 'krs'|'revisi'|'closed', start: string, end: string }, eligibility: { ok: boolean, reasons: string[] } }`
- `GET /krs/me/offerings?q=` → `Offering[]`
- `POST /krs/me/items` `{ kelas_id }` + `Idempotency-Key` → `Krs`
- `DELETE /krs/me/items/:kelas_id` → `Krs`
- `POST /krs/me/submit` → `Krs`

### KRS — dosen wali (EC-14)
- `GET /krs/advisees?status=&periode_id=` → `Krs[]`
- `GET /krs/:id` → `Krs`
- `POST /krs/:id/approve` `{ catatan? }` / `POST /krs/:id/reject` `{ catatan }` → `Krs`

### Perkuliahan — dosen (FR-21..23, EC-11, EC-12)
- `GET /teaching/kelas?periode_id=` → `Kelas[]` (classes the user teaches)
- `GET /kelas/:id/peserta` → `{ mahasiswa_id, nim, nama }[]`
- `GET /kelas/:id/pertemuan` → `Pertemuan[]`; `POST /kelas/:id/pertemuan` `{ tanggal, topik }`
- `GET /pertemuan/:id/presensi` → `PresensiRow[]`; `PUT /pertemuan/:id/presensi` `{ items: {mahasiswa_id, status}[] }`
- `GET /kelas/:id/nilai` → `{ komponen: KomponenNilai[], rows: NilaiRow[], finalized: boolean, deadline: string }`
- `PUT /kelas/:id/nilai` `{ rows: { krs_detail_id, komponen: Record<string, number|null> }[] }` → same as GET
- `POST /kelas/:id/nilai/finalize`
- `POST /nilai/:id/koreksi` `{ komponen: Record<string,number>, alasan }`
- `GET /koreksi-nilai?status=` → `KoreksiNilai[]` (kaprodi sees own prodi, dosen sees own)
- `POST /koreksi-nilai/:id/approve`, `POST /koreksi-nilai/:id/reject` `{ catatan }`

### Akademik — mahasiswa (FR-23, FR-24, EC-06)
- `GET /akademik/me/jadwal` → `JadwalItem[]`
- `GET /akademik/me/khs?periode_id=` → `Khs`; `GET /akademik/me/periodes` → `Periode[]`
- `GET /akademik/me/transkrip` → `Transkrip`
- `GET /akademik/me/profil` → `{ mahasiswa: Mahasiswa, riwayat: RiwayatStatus[] }`
- Kaprodi: `GET /akademik/masa-studi-warning` → `(Mahasiswa & { sisa_semester: number })[]` (EC-09)

### Keuangan (FR-25..28, EC-03, EC-15)
- `GET /tagihan?status=&periode_id=&prodi_id=&q=` → `Tagihan[]` (keuangan)
- `GET /tagihan/me` → `Tagihan[]` (mahasiswa)
- `POST /tagihan/generate` `{ periode_id }` → `{ created: number }`
- `POST /tagihan/:id/va` → `Tagihan` (issues fake VA number)
- `POST /tagihan/:id/cicilan` `{ jumlah_cicilan: 2|3|4 }` → `Tagihan`
- `GET /tagihan/:id/pembayaran` → `Pembayaran[]`
- `POST /webhooks/payment` (no auth, HMAC header `X-Signature`) `{ transaction_id, va_number, amount, paid_at }` → `200`; dedup by `transaction_id`, status recomputed from sum
- `POST /simulator/pay` (auth, any role, demo only) `{ va_number, amount }` → fires the webhook internally → `Tagihan`
- `GET /dispensasi?status=` / `POST /dispensasi` `{ mahasiswa_id, periode_id, jenis, alasan }` / `POST /dispensasi/:id/approve` / `POST /dispensasi/:id/reject`
- `GET /keuangan/piutang?group=prodi|angkatan` → `{ group: string; total_tagihan: number; total_terbayar: number; piutang: number }[]`

### PMB (FR-08..10, EC-04)
- `GET /pmb/gelombang`, `POST /pmb/gelombang`
- `GET /pmb/pendaftar?status=&gelombang_id=&q=` → `Pendaftar[]`
- `GET /pmb/public-options?tenant_code=` (no auth) → `{ gelombang: Gelombang[] /* currently open */, prodi: Prodi[] }`
- `POST /pmb/register` (no auth) `{ tenant_code, nama, email, nik, gelombang_id, prodi_pilihan_id, password }` → `Pendaftar`
- `GET /pmb/me` (calon_mahasiswa) → `Pendaftar`
- `POST /pmb/pendaftar/:id/transition` `{ to: PmbStatus }` → `Pendaftar` (`PMB_INVALID_TRANSITION`)
- `POST /pmb/pendaftar/:id/daftar-ulang` → `Pendaftar` with `nim` (sequence per prodi+angkatan, row lock)

### PDDikti (FR-29..31, PV-01..08)
- `POST /pddikti/validate` `{ periode_id }` → `ValidationResult`
- `GET /pddikti/validation?periode_id=` → latest `ValidationResult` or `null`
- `GET /pddikti/export/:entity?periode_id=` (`entity` = `mahasiswa|kelas|krs|nilai|aktivitas`) → `text/csv`; `422 PDDIKTI_BLOCKED` if errors exist

### Audit & notifikasi (FR-32, FR-33)
- `GET /audit-log?entitas=&actor=&q=` → `AuditEntry[]`
- `GET /notifikasi` → `Notifikasi[]`; `POST /notifikasi/:id/read`

---

## Seed / demo accounts (tenant `UNIKA`, password `password123`)
| Role | Identifier |
|---|---|
| super_admin | `super@kampusops.id` (tenant `UNIKA`) |
| admin_kampus | `admin@unika.ac.id` |
| operator | `baak@unika.ac.id` |
| kaprodi + dosen + dosen_wali | `0011223344` (NIDN) |
| dosen | `0055667788` |
| keuangan | `keuangan@unika.ac.id` |
| mahasiswa | `2024110001` (NIM, active, paid) and `2024110002` (unpaid) |
| calon_mahasiswa | `calon@mail.com` |
Second tenant `POLTEK` (timezone `Asia/Makassar`) with overlapping prodi kode / NIM to prove EC-16.
