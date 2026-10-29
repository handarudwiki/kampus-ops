-- KampusOps core schema.
-- Rules: every business table carries tenant_id, created_at, updated_at and every
-- unique index starts with tenant_id (EC-16). Timestamps are stored in UTC (timestamptz).

CREATE OR REPLACE FUNCTION iso(ts timestamptz) RETURNS text
LANGUAGE sql IMMUTABLE AS $$
  SELECT to_char(ts AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
$$;

CREATE TABLE tenant (
  id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  kode               text NOT NULL,
  nama               text NOT NULL,
  timezone           text NOT NULL DEFAULT 'Asia/Jakarta',
  plan               text NOT NULL DEFAULT 'basic',
  konfigurasi_aturan jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX tenant_kode_uq ON tenant (upper(kode));

CREATE TABLE users (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  nama            text NOT NULL,
  email           text NOT NULL,
  identifier      text,              -- NIM / NIDN / no pendaftaran
  password_hash   text NOT NULL,
  failed_attempts int  NOT NULL DEFAULT 0,
  locked_until    timestamptz,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_tenant_email_uq ON users (tenant_id, lower(email));
CREATE UNIQUE INDEX users_tenant_identifier_uq ON users (tenant_id, identifier) WHERE identifier IS NOT NULL;

CREATE TABLE fakultas (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  kode       text NOT NULL,
  nama       text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX fakultas_tenant_kode_uq ON fakultas (tenant_id, kode);

CREATE TABLE prodi (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  fakultas_id     uuid NOT NULL REFERENCES fakultas(id),
  kode            text NOT NULL,
  nama            text NOT NULL,
  jenjang         text NOT NULL CHECK (jenjang IN ('D3','S1','S2')),
  masa_studi_maks int  NOT NULL,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX prodi_tenant_kode_uq ON prodi (tenant_id, kode);

CREATE TABLE user_role (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role           text NOT NULL CHECK (role IN ('super_admin','admin_kampus','operator','kaprodi','dosen_wali','dosen','keuangan','mahasiswa','calon_mahasiswa')),
  scope_prodi_id uuid REFERENCES prodi(id),
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX user_role_uq ON user_role (tenant_id, user_id, role);

CREATE TABLE refresh_token (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash  text NOT NULL,
  expires_at  timestamptz NOT NULL,
  revoked_at  timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX refresh_token_hash_uq ON refresh_token (tenant_id, token_hash);
CREATE INDEX refresh_token_hash_idx ON refresh_token (token_hash);

CREATE TABLE ruang (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  kode       text NOT NULL,
  nama       text NOT NULL,
  kapasitas  int  NOT NULL DEFAULT 40,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ruang_tenant_kode_uq ON ruang (tenant_id, kode);

CREATE TABLE dosen (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  user_id    uuid REFERENCES users(id) ON DELETE SET NULL,
  nidn       text NOT NULL,
  nama       text NOT NULL,
  email      text NOT NULL,
  prodi_id   uuid NOT NULL REFERENCES prodi(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX dosen_tenant_nidn_uq ON dosen (tenant_id, nidn);
CREATE INDEX dosen_user_idx ON dosen (tenant_id, user_id);

CREATE TABLE periode (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  kode        text NOT NULL,
  tahun       int  NOT NULL,
  tipe        text NOT NULL CHECK (tipe IN ('Ganjil','Genap','Pendek')),
  status      text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','aktif','selesai')),
  tahapan     jsonb NOT NULL DEFAULT '{}'::jsonb,
  exported_at timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX periode_tenant_kode_uq ON periode (tenant_id, kode);
-- FR-06: only one active period per tenant, enforced by the database.
CREATE UNIQUE INDEX periode_one_active_uq ON periode (tenant_id) WHERE status = 'aktif';

CREATE TABLE kurikulum (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  prodi_id       uuid NOT NULL REFERENCES prodi(id),
  nama           text NOT NULL,
  angkatan_mulai int  NOT NULL,
  aktif          boolean NOT NULL DEFAULT true,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX kurikulum_tenant_uq ON kurikulum (tenant_id, prodi_id, nama);

CREATE TABLE mata_kuliah (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  kurikulum_id uuid NOT NULL REFERENCES kurikulum(id),
  kode         text NOT NULL,
  nama         text NOT NULL,
  sks          int  NOT NULL CHECK (sks BETWEEN 1 AND 8),
  semester     int  NOT NULL CHECK (semester BETWEEN 1 AND 14),
  sifat        text NOT NULL DEFAULT 'wajib' CHECK (sifat IN ('wajib','pilihan')),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX mata_kuliah_tenant_uq ON mata_kuliah (tenant_id, kurikulum_id, kode);

CREATE TABLE prasyarat (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  mk_id        uuid NOT NULL REFERENCES mata_kuliah(id) ON DELETE CASCADE,
  mk_syarat_id uuid NOT NULL REFERENCES mata_kuliah(id) ON DELETE CASCADE,
  nilai_min    text,   -- minimum letter grade, NULL = "sudah lulus"
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX prasyarat_tenant_uq ON prasyarat (tenant_id, mk_id, mk_syarat_id);

CREATE TABLE ekuivalensi (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  mk_lama_id uuid NOT NULL REFERENCES mata_kuliah(id) ON DELETE CASCADE,
  mk_baru_id uuid NOT NULL REFERENCES mata_kuliah(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ekuivalensi_tenant_uq ON ekuivalensi (tenant_id, mk_lama_id, mk_baru_id);

CREATE TABLE mahasiswa (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  user_id       uuid REFERENCES users(id) ON DELETE SET NULL,
  nim           text NOT NULL,
  nik           text,
  nama          text NOT NULL,
  email         text NOT NULL,
  prodi_id      uuid NOT NULL REFERENCES prodi(id),
  angkatan      int  NOT NULL,
  kurikulum_id  uuid NOT NULL REFERENCES kurikulum(id),
  dosen_wali_id uuid REFERENCES dosen(id) ON DELETE SET NULL,
  -- denormalised cache of the latest riwayat_status row; rules read riwayat_status.
  status        text NOT NULL DEFAULT 'aktif' CHECK (status IN ('aktif','cuti','non_aktif','lulus','keluar','do')),
  -- cached cumulative grade numbers (sum sks*bobot_x100 / sum sks) recomputed in-tx (EC-10)
  ipk_num       bigint NOT NULL DEFAULT 0,
  ipk_den       bigint NOT NULL DEFAULT 0,
  sks_lulus     int    NOT NULL DEFAULT 0,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX mahasiswa_tenant_nim_uq ON mahasiswa (tenant_id, nim);
CREATE INDEX mahasiswa_user_idx ON mahasiswa (tenant_id, user_id);
CREATE INDEX mahasiswa_wali_idx ON mahasiswa (tenant_id, dosen_wali_id);
CREATE INDEX mahasiswa_prodi_idx ON mahasiswa (tenant_id, prodi_id, angkatan);

CREATE TABLE riwayat_status (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  mahasiswa_id uuid NOT NULL REFERENCES mahasiswa(id) ON DELETE CASCADE,
  periode_id   uuid NOT NULL REFERENCES periode(id),
  status       text NOT NULL CHECK (status IN ('aktif','cuti','non_aktif','lulus','keluar','do')),
  alasan       text,
  created_by   uuid,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX riwayat_status_mhs_idx ON riwayat_status (tenant_id, mahasiswa_id, created_at);

-- EC-04: NIM sequence per (tenant, prodi, angkatan), locked with SELECT ... FOR UPDATE.
CREATE TABLE nim_sequence (
  tenant_id  uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  prodi_id   uuid NOT NULL REFERENCES prodi(id),
  angkatan   int  NOT NULL,
  last_no    int  NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, prodi_id, angkatan)
);

CREATE TABLE kelas (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  periode_id  uuid NOT NULL REFERENCES periode(id),
  mk_id       uuid NOT NULL REFERENCES mata_kuliah(id),
  nama        text NOT NULL,
  sks         int  NOT NULL,
  dosen_id    uuid REFERENCES dosen(id),
  ruang_id    uuid REFERENCES ruang(id),
  hari        int  NOT NULL CHECK (hari BETWEEN 1 AND 6),
  jam_mulai   time NOT NULL,
  jam_selesai time NOT NULL,
  kuota       int  NOT NULL CHECK (kuota >= 0),
  terisi      int  NOT NULL DEFAULT 0,
  status      text NOT NULL DEFAULT 'dibuka' CHECK (status IN ('dibuka','dibatalkan')),
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT kelas_terisi_range CHECK (terisi >= 0 AND terisi <= kuota),
  CONSTRAINT kelas_jam_valid CHECK (jam_selesai > jam_mulai)
);
CREATE UNIQUE INDEX kelas_tenant_uq ON kelas (tenant_id, periode_id, mk_id, nama);
CREATE INDEX kelas_periode_idx ON kelas (tenant_id, periode_id);

-- EC-12: lecturer assignment history with effective dates.
CREATE TABLE kelas_dosen (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  kelas_id        uuid NOT NULL REFERENCES kelas(id) ON DELETE CASCADE,
  dosen_id        uuid NOT NULL REFERENCES dosen(id),
  efektif_mulai   date NOT NULL,
  efektif_selesai date,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX kelas_dosen_idx ON kelas_dosen (tenant_id, kelas_id);

CREATE TABLE komponen_nilai (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  kelas_id     uuid NOT NULL REFERENCES kelas(id) ON DELETE CASCADE,
  key          text NOT NULL,
  label        text NOT NULL,
  bobot_persen int  NOT NULL,
  urutan       int  NOT NULL DEFAULT 0,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX komponen_nilai_uq ON komponen_nilai (tenant_id, kelas_id, key);

CREATE TABLE krs (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  mahasiswa_id uuid NOT NULL REFERENCES mahasiswa(id) ON DELETE CASCADE,
  periode_id   uuid NOT NULL REFERENCES periode(id),
  status       text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','diajukan','disetujui','ditolak')),
  total_sks    int  NOT NULL DEFAULT 0,
  max_sks      int  NOT NULL DEFAULT 0,
  ips_lalu     text,
  catatan      text,
  submitted_at timestamptz,
  decided_at   timestamptz,
  decided_by   uuid,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX krs_tenant_mhs_periode_uq ON krs (tenant_id, mahasiswa_id, periode_id);
CREATE INDEX krs_periode_status_idx ON krs (tenant_id, periode_id, status);

CREATE TABLE krs_detail (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  krs_id     uuid NOT NULL REFERENCES krs(id) ON DELETE CASCADE,
  kelas_id   uuid NOT NULL REFERENCES kelas(id),
  mk_id      uuid NOT NULL REFERENCES mata_kuliah(id),
  sks        int  NOT NULL,
  is_ulang   boolean NOT NULL DEFAULT false,
  status     text NOT NULL DEFAULT 'aktif' CHECK (status IN ('aktif','dibatalkan')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX krs_detail_kelas_uq ON krs_detail (tenant_id, krs_id, kelas_id) WHERE status = 'aktif';
CREATE UNIQUE INDEX krs_detail_mk_uq ON krs_detail (tenant_id, krs_id, mk_id) WHERE status = 'aktif';
CREATE INDEX krs_detail_kelas_idx ON krs_detail (tenant_id, kelas_id);

CREATE TABLE pertemuan (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  kelas_id   uuid NOT NULL REFERENCES kelas(id) ON DELETE CASCADE,
  ke         int  NOT NULL,
  tanggal    date NOT NULL,
  topik      text NOT NULL DEFAULT '',
  dosen_id   uuid REFERENCES dosen(id),   -- EC-12: recorded per meeting
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX pertemuan_uq ON pertemuan (tenant_id, kelas_id, ke);

CREATE TABLE presensi (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  pertemuan_id uuid NOT NULL REFERENCES pertemuan(id) ON DELETE CASCADE,
  mahasiswa_id uuid NOT NULL REFERENCES mahasiswa(id) ON DELETE CASCADE,
  status       text NOT NULL CHECK (status IN ('H','I','S','A')),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX presensi_uq ON presensi (tenant_id, pertemuan_id, mahasiswa_id);

CREATE TABLE nilai (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id         uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  krs_detail_id     uuid REFERENCES krs_detail(id) ON DELETE SET NULL,
  mahasiswa_id      uuid NOT NULL REFERENCES mahasiswa(id) ON DELETE CASCADE,
  kelas_id          uuid NOT NULL REFERENCES kelas(id),
  komponen          jsonb NOT NULL DEFAULT '{}'::jsonb,   -- key -> score x100 (integer)
  angka_x100        int,
  huruf             text,
  bobot_x100        int,
  final_at          timestamptz,
  perlu_lapor_ulang boolean NOT NULL DEFAULT false,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX nilai_detail_uq ON nilai (tenant_id, krs_detail_id) WHERE krs_detail_id IS NOT NULL;
CREATE INDEX nilai_mhs_idx ON nilai (tenant_id, mahasiswa_id);
CREATE INDEX nilai_kelas_idx ON nilai (tenant_id, kelas_id);

CREATE TABLE koreksi_nilai (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  nilai_id      uuid NOT NULL REFERENCES nilai(id) ON DELETE CASCADE,
  komponen_lama jsonb NOT NULL,
  komponen_baru jsonb NOT NULL,
  angka_lama    int  NOT NULL,
  angka_baru    int  NOT NULL,
  huruf_lama    text NOT NULL,
  huruf_baru    text NOT NULL,
  bobot_lama    int  NOT NULL,
  bobot_baru    int  NOT NULL,
  alasan        text NOT NULL,
  status        text NOT NULL DEFAULT 'diajukan' CHECK (status IN ('diajukan','disetujui','ditolak')),
  diajukan_oleh uuid NOT NULL REFERENCES users(id),
  approved_by   uuid REFERENCES users(id),
  catatan       text,
  decided_at    timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX koreksi_one_pending_uq ON koreksi_nilai (tenant_id, nilai_id) WHERE status = 'diajukan';

CREATE TABLE tarif (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  prodi_id   uuid NOT NULL REFERENCES prodi(id),
  angkatan   int  NOT NULL,
  jenis      text NOT NULL CHECK (jenis IN ('ukt','praktikum','cuti')),
  jumlah     bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX tarif_uq ON tarif (tenant_id, prodi_id, angkatan, jenis);

CREATE TABLE gelombang (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  nama           text NOT NULL,
  mulai          date NOT NULL,
  selesai        date NOT NULL,
  kuota          int  NOT NULL,
  terisi         int  NOT NULL DEFAULT 0,
  biaya_formulir bigint NOT NULL,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT gelombang_terisi_range CHECK (terisi >= 0 AND terisi <= kuota)
);
CREATE UNIQUE INDEX gelombang_uq ON gelombang (tenant_id, nama);

CREATE TABLE pendaftar (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id        uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  user_id          uuid REFERENCES users(id) ON DELETE SET NULL,
  no_daftar        text NOT NULL,
  nama             text NOT NULL,
  email            text NOT NULL,
  nik              text NOT NULL,
  gelombang_id     uuid NOT NULL REFERENCES gelombang(id),
  prodi_pilihan_id uuid NOT NULL REFERENCES prodi(id),
  status           text NOT NULL DEFAULT 'daftar' CHECK (status IN ('daftar','bayar_formulir','lengkapi_berkas','verifikasi','lulus','tidak_lulus','daftar_ulang')),
  berkas           jsonb NOT NULL DEFAULT '[]'::jsonb,
  nim              text,
  mahasiswa_id     uuid REFERENCES mahasiswa(id),
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX pendaftar_no_uq ON pendaftar (tenant_id, no_daftar);
CREATE UNIQUE INDEX pendaftar_nim_uq ON pendaftar (tenant_id, nim) WHERE nim IS NOT NULL;

CREATE TABLE tagihan (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  mahasiswa_id uuid REFERENCES mahasiswa(id) ON DELETE CASCADE,
  pendaftar_id uuid REFERENCES pendaftar(id) ON DELETE CASCADE,
  periode_id   uuid REFERENCES periode(id),
  jenis        text NOT NULL CHECK (jenis IN ('ukt','praktikum','cuti','formulir_pmb')),
  jumlah       bigint NOT NULL CHECK (jumlah >= 0),
  terbayar     bigint NOT NULL DEFAULT 0,   -- cache of SUM(pembayaran.jumlah) (EC-03)
  status       text NOT NULL DEFAULT 'belum_bayar' CHECK (status IN ('belum_bayar','sebagian','lunas')),
  jatuh_tempo  date NOT NULL,
  va_number    text,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT tagihan_owner CHECK (mahasiswa_id IS NOT NULL OR pendaftar_id IS NOT NULL)
);
CREATE UNIQUE INDEX tagihan_mhs_uq ON tagihan (tenant_id, mahasiswa_id, periode_id, jenis) WHERE mahasiswa_id IS NOT NULL;
CREATE UNIQUE INDEX tagihan_va_uq ON tagihan (tenant_id, va_number) WHERE va_number IS NOT NULL;
-- The payment webhook carries no tenant: VA numbers embed a per-tenant prefix and
-- are resolved through this (non-unique) lookup index.
CREATE INDEX tagihan_va_lookup_idx ON tagihan (va_number) WHERE va_number IS NOT NULL;
CREATE INDEX tagihan_periode_idx ON tagihan (tenant_id, periode_id, status);

CREATE TABLE cicilan (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  tagihan_id  uuid NOT NULL REFERENCES tagihan(id) ON DELETE CASCADE,
  ke          int  NOT NULL,
  jumlah      bigint NOT NULL,
  jatuh_tempo date NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX cicilan_uq ON cicilan (tenant_id, tagihan_id, ke);

CREATE TABLE pembayaran (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  tagihan_id     uuid NOT NULL REFERENCES tagihan(id) ON DELETE CASCADE,
  transaction_id text NOT NULL,
  jumlah         bigint NOT NULL CHECK (jumlah > 0),
  paid_at        timestamptz NOT NULL,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);
-- EC-03: duplicate webhook deliveries are dropped by this index.
CREATE UNIQUE INDEX pembayaran_trx_uq ON pembayaran (tenant_id, transaction_id);

CREATE TABLE dispensasi (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  mahasiswa_id   uuid NOT NULL REFERENCES mahasiswa(id) ON DELETE CASCADE,
  periode_id     uuid NOT NULL REFERENCES periode(id),
  jenis          text NOT NULL CHECK (jenis IN ('cicilan','penundaan','krs_susulan')),
  alasan         text NOT NULL,
  status         text NOT NULL DEFAULT 'diajukan' CHECK (status IN ('diajukan','disetujui','ditolak')),
  disetujui_oleh uuid REFERENCES users(id),
  created_by     uuid,
  decided_at     timestamptz,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX dispensasi_mhs_idx ON dispensasi (tenant_id, mahasiswa_id, periode_id);

CREATE TABLE audit_log (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  actor_id    uuid,
  actor_nama  text NOT NULL,
  actor_role  text NOT NULL,
  entitas     text NOT NULL,
  entitas_id  uuid NOT NULL,
  aksi        text NOT NULL,
  before      jsonb,
  after       jsonb,
  request_id  text NOT NULL DEFAULT '',
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_tenant_idx ON audit_log (tenant_id, created_at DESC);
CREATE INDEX audit_log_entitas_idx ON audit_log (tenant_id, entitas, entitas_id);

-- Append-only: block UPDATE / DELETE on audit_log (TRUNCATE by the seeder is allowed).
CREATE OR REPLACE FUNCTION audit_log_append_only() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'audit_log is append-only';
END $$;
CREATE TRIGGER audit_log_no_update BEFORE UPDATE OR DELETE ON audit_log
  FOR EACH ROW EXECUTE FUNCTION audit_log_append_only();

CREATE TABLE notifikasi (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  judul      text NOT NULL,
  isi        text NOT NULL,
  link       text,
  read_at    timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notifikasi_user_idx ON notifikasi (tenant_id, user_id, created_at DESC);

CREATE TABLE validasi_run (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  periode_id uuid NOT NULL REFERENCES periode(id),
  score      int  NOT NULL,
  errors     int  NOT NULL,
  warnings   int  NOT NULL,
  result     jsonb NOT NULL,
  ran_by     uuid,
  ran_at     timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX validasi_run_idx ON validasi_run (tenant_id, periode_id, ran_at DESC);

CREATE TABLE validasi_pddikti (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
  run_id      uuid NOT NULL REFERENCES validasi_run(id) ON DELETE CASCADE,
  periode_id  uuid NOT NULL REFERENCES periode(id),
  aturan      text NOT NULL,
  entitas     text NOT NULL,
  entitas_id  uuid NOT NULL,
  tingkat     text NOT NULL CHECK (tingkat IN ('error','warning')),
  pesan       text NOT NULL,
  resolved_at timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX validasi_pddikti_idx ON validasi_pddikti (tenant_id, periode_id, aturan);
