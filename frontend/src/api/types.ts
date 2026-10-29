// Mirrors docs/API_CONTRACT.md. Keep in sync with the backend.
export type ID = string

export type Role =
  | 'super_admin'
  | 'admin_kampus'
  | 'operator'
  | 'kaprodi'
  | 'dosen_wali'
  | 'dosen'
  | 'keuangan'
  | 'mahasiswa'
  | 'calon_mahasiswa'

export interface Tenant { id: ID; code: string; name: string; timezone: string }
export interface UserRole { role: Role; scope_prodi_id: ID | null; scope_prodi_name: string | null }
export interface User { id: ID; name: string; email: string; identifier: string; roles: UserRole[] }
export interface AuthResponse { access_token: string; refresh_token: string; expires_in: number; user: User; tenant: Tenant }

export interface Fakultas { id: ID; kode: string; nama: string }
export interface Prodi {
  id: ID; fakultas_id: ID; fakultas_nama: string; kode: string; nama: string
  jenjang: 'D3' | 'S1' | 'S2'; masa_studi_maks: number
}
export interface Dosen { id: ID; nidn: string; nama: string; email: string; prodi_id: ID; prodi_nama: string }
export interface Ruang { id: ID; kode: string; nama: string; kapasitas: number }

export type PeriodeTipe = 'Ganjil' | 'Genap' | 'Pendek'
export interface Tahapan {
  krs_mulai: string; krs_selesai: string; revisi_mulai: string | null; revisi_selesai: string | null
  kuliah_mulai: string; kuliah_selesai: string; uts: string; uas: string; nilai_selesai: string
}
export interface Periode {
  id: ID; kode: string; tahun: number; tipe: PeriodeTipe
  status: 'draft' | 'aktif' | 'selesai'; tahapan: Tahapan
}

export type StatusMhs = 'aktif' | 'cuti' | 'non_aktif' | 'lulus' | 'keluar' | 'do'
export interface Mahasiswa {
  id: ID; nim: string; nik: string | null; nama: string; email: string
  prodi_id: ID; prodi_nama: string; angkatan: number; kurikulum_id: ID
  dosen_wali_id: ID | null; dosen_wali_nama: string | null; status: StatusMhs
  ipk: string; sks_lulus: number; semester_ke: number
}
export interface RiwayatStatus { periode_kode: string; status: StatusMhs; alasan: string | null; created_at: string }

export interface Kurikulum { id: ID; prodi_id: ID; nama: string; angkatan_mulai: number; aktif: boolean }
export interface MataKuliah {
  id: ID; kurikulum_id: ID; kode: string; nama: string; sks: number; semester: number
  sifat: 'wajib' | 'pilihan'
  prasyarat: { mk_id: ID; kode: string; nama: string; nilai_min: string | null }[]
}

export interface Slot { hari: 1 | 2 | 3 | 4 | 5 | 6; jam_mulai: string; jam_selesai: string }
export interface Kelas {
  id: ID; periode_id: ID; mk_id: ID; mk_kode: string; mk_nama: string; sks: number
  nama: string; dosen_id: ID | null; dosen_nama: string | null; ruang_id: ID | null; ruang_nama: string | null
  slot: Slot; kuota: number; terisi: number; status: 'dibuka' | 'dibatalkan'
}

export type KrsStatus = 'draft' | 'diajukan' | 'disetujui' | 'ditolak'
export interface KrsItem { kelas: Kelas; is_ulang: boolean }
export interface Krs {
  id: ID
  mahasiswa: Pick<Mahasiswa, 'id' | 'nim' | 'nama' | 'prodi_nama' | 'angkatan' | 'ipk'>
  periode_id: ID; status: KrsStatus; total_sks: number; max_sks: number; ips_lalu: string | null
  catatan: string | null; submitted_at: string | null; decided_at: string | null; items: KrsItem[]
}
export interface Offering extends Kelas { eligible: boolean; reasons: string[]; in_krs: boolean }
export interface KrsMe {
  krs: Krs | null
  window: { open: boolean; phase: 'krs' | 'revisi' | 'closed'; start: string; end: string }
  eligibility: { ok: boolean; reasons: string[] }
}

export interface Pertemuan {
  id: ID; kelas_id: ID; ke: number; tanggal: string; topik: string; dosen_nama: string
  rekap: { hadir: number; izin: number; sakit: number; alpa: number }
}
export type PresensiStatus = 'H' | 'I' | 'S' | 'A'
export interface PresensiRow { mahasiswa_id: ID; nim: string; nama: string; status: PresensiStatus | null }

export interface KomponenNilai { key: string; label: string; bobot_persen: number }
export interface NilaiRow {
  krs_detail_id: ID; nilai_id: ID | null; mahasiswa_id: ID; nim: string; nama: string
  komponen: Record<string, number | null>; angka: string | null; huruf: string | null
  bobot_x100: number | null; final: boolean; koreksi_pending: boolean
}
export interface NilaiKelas { komponen: KomponenNilai[]; rows: NilaiRow[]; finalized: boolean; deadline: string }
export interface KoreksiNilai {
  id: ID; nilai_id: ID; kelas_nama: string; mk_nama: string; mahasiswa_nama: string; nim: string
  huruf_lama: string; huruf_baru: string; angka_lama: string; angka_baru: string; alasan: string
  status: 'diajukan' | 'disetujui' | 'ditolak'; diajukan_oleh: string; created_at: string
}

export interface KhsRow { mk_kode: string; mk_nama: string; sks: number; huruf: string | null; bobot_x100: number | null; is_ulang: boolean }
export interface Khs { periode: Periode; rows: KhsRow[]; total_sks: number; ips: string; ipk: string }
export interface Transkrip {
  mahasiswa: Mahasiswa
  rows: (KhsRow & { semester: number; periode_kode: string })[]
  total_sks: number; ipk: string
}
export interface JadwalItem { kelas: Kelas }

export type TagihanStatus = 'belum_bayar' | 'sebagian' | 'lunas'
export interface Tagihan {
  id: ID; mahasiswa_id: ID; nim: string; nama: string; prodi_nama: string; periode_kode: string
  jenis: 'ukt' | 'praktikum' | 'cuti' | 'formulir_pmb'; jumlah: number; terbayar: number; status: TagihanStatus
  jatuh_tempo: string; va_number: string | null
  cicilan: { ke: number; jumlah: number; jatuh_tempo: string; lunas: boolean }[]
}
export interface Pembayaran { id: ID; tagihan_id: ID; transaction_id: string; jumlah: number; paid_at: string }
export interface Dispensasi {
  id: ID; mahasiswa_id: ID; nim: string; nama: string; periode_kode: string
  jenis: 'cicilan' | 'penundaan' | 'krs_susulan'; alasan: string
  status: 'diajukan' | 'disetujui' | 'ditolak'; disetujui_oleh: string | null; created_at: string
}
export interface Piutang { group: string; total_tagihan: number; total_terbayar: number; piutang: number }

export type PmbStatus = 'daftar' | 'bayar_formulir' | 'lengkapi_berkas' | 'verifikasi' | 'lulus' | 'tidak_lulus' | 'daftar_ulang'
export interface Gelombang { id: ID; nama: string; mulai: string; selesai: string; kuota: number; terisi: number; biaya_formulir: number }
export interface Pendaftar {
  id: ID; no_daftar: string; nama: string; email: string; nik: string; gelombang_id: ID; gelombang_nama: string
  prodi_pilihan_id: ID; prodi_pilihan_nama: string; status: PmbStatus; berkas: { nama: string; ok: boolean }[]
  nim: string | null; created_at: string
}

export interface ValidationIssue {
  id: ID; rule: string; level: 'error' | 'warning'; entity: string
  entity_id: ID; entity_label: string; message: string; fix_url: string
}
export interface ValidationResult {
  periode_id: ID; score: number; errors: number; warnings: number; ran_at: string
  by_rule: { rule: string; label: string; level: 'error' | 'warning'; count: number }[]
  items: ValidationIssue[]
}

export interface AuditEntry {
  id: ID; actor_nama: string; actor_role: Role; entitas: string; entitas_id: ID; aksi: string
  before: unknown; after: unknown; created_at: string; request_id: string
}
export interface Notifikasi { id: ID; judul: string; isi: string; link: string | null; read: boolean; created_at: string }

export type Tone = 'blue' | 'green' | 'amber' | 'red' | 'violet' | 'pink' | 'gray'
export interface StatCard { key: string; label: string; value: string; hint: string | null; tone: Exclude<Tone, 'gray'>; trend: 'up' | 'down' | null }
export interface DashAlert { level: 'info' | 'warning' | 'error'; message: string; link: string | null }
export interface Dashboard {
  stats: StatCard[]
  alerts: DashAlert[]
  chart: { title: string; labels: string[]; series: { name: string; values: number[] }[] } | null
  table: { title: string; columns: { key: string; label: string }[]; rows: Record<string, string>[]; link: string | null } | null
}

export interface ListMeta { page: number; per_page: number; total: number }
export interface Paged<T> { data: T[]; meta?: ListMeta }
