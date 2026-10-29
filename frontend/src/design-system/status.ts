import type { Tone } from '@/api/types'

/**
 * Every domain status in the app maps to exactly one label + tone here.
 * <KStatus domain="krs" :value="krs.status" /> is the only way statuses are rendered,
 * so "Disetujui" is always green and "Diajukan" is always amber on every screen.
 */
export interface StatusDef { label: string; tone: Tone }

const S = (label: string, tone: Tone): StatusDef => ({ label, tone })

export const STATUS = {
  krs: {
    draft: S('Draft', 'gray'),
    diajukan: S('Diajukan', 'amber'),
    disetujui: S('Disetujui', 'green'),
    ditolak: S('Ditolak', 'red'),
  },
  mahasiswa: {
    aktif: S('Aktif', 'green'),
    cuti: S('Cuti', 'amber'),
    non_aktif: S('Non-aktif', 'gray'),
    lulus: S('Lulus', 'blue'),
    keluar: S('Keluar', 'red'),
    do: S('Drop out', 'red'),
  },
  tagihan: {
    belum_bayar: S('Belum bayar', 'red'),
    sebagian: S('Sebagian', 'amber'),
    lunas: S('Lunas', 'green'),
  },
  periode: {
    draft: S('Draft', 'gray'),
    aktif: S('Aktif', 'green'),
    selesai: S('Selesai', 'blue'),
  },
  kelas: {
    dibuka: S('Dibuka', 'green'),
    dibatalkan: S('Dibatalkan', 'red'),
  },
  approval: {
    diajukan: S('Menunggu', 'amber'),
    disetujui: S('Disetujui', 'green'),
    ditolak: S('Ditolak', 'red'),
  },
  pmb: {
    daftar: S('Daftar', 'gray'),
    bayar_formulir: S('Bayar formulir', 'violet'),
    lengkapi_berkas: S('Lengkapi berkas', 'pink'),
    verifikasi: S('Verifikasi', 'amber'),
    lulus: S('Lulus seleksi', 'blue'),
    tidak_lulus: S('Tidak lulus', 'red'),
    daftar_ulang: S('Daftar ulang', 'green'),
  },
  presensi: {
    H: S('Hadir', 'green'),
    I: S('Izin', 'blue'),
    S: S('Sakit', 'amber'),
    A: S('Alpa', 'red'),
  },
  level: {
    error: S('Error', 'red'),
    warning: S('Peringatan', 'amber'),
    info: S('Info', 'blue'),
  },
  jenisTagihan: {
    ukt: S('UKT', 'blue'),
    praktikum: S('Praktikum', 'violet'),
    cuti: S('Biaya cuti', 'amber'),
    formulir_pmb: S('Formulir PMB', 'pink'),
  },
  jenisDispensasi: {
    cicilan: S('Cicilan', 'blue'),
    penundaan: S('Penundaan', 'violet'),
    krs_susulan: S('KRS susulan', 'pink'),
  },
} satisfies Record<string, Record<string, StatusDef>>

export type StatusDomain = keyof typeof STATUS

export function statusOf(domain: StatusDomain, value: string | null | undefined): StatusDef {
  if (!value) return S('—', 'gray')
  return (STATUS[domain] as Record<string, StatusDef>)[value] ?? S(value, 'gray')
}

/**
 * Find the status definition for a raw enum value coming from a generic payload
 * (e.g. dashboard tables). The column key narrows the search; falls back to any domain.
 */
const HINTS: Record<string, StatusDomain[]> = {
  status: ['krs', 'tagihan', 'mahasiswa', 'pmb', 'approval', 'kelas', 'periode'],
  jenis: ['jenisTagihan', 'jenisDispensasi'],
  tingkat: ['level'],
  level: ['level'],
}
export function lookupStatus(column: string, value: unknown): StatusDef | null {
  if (typeof value !== 'string') return null
  const order = [...(HINTS[column] ?? []), ...(Object.keys(STATUS) as StatusDomain[])]
  for (const d of order) {
    const hit = (STATUS[d] as Record<string, StatusDef>)[value]
    if (hit && (HINTS[column] || value.includes('_') || d === 'level')) return hit
  }
  return null
}

/** Options for <KSelect> filters built from the same map. */
export function statusOptions(domain: StatusDomain) {
  return Object.entries(STATUS[domain]).map(([value, d]) => ({ value, label: d.label }))
}

/** Human messages for API business error codes (docs/API_CONTRACT.md). */
export const ERROR_TEXT: Record<string, string> = {
  AUTH_INVALID_CREDENTIALS: 'Identitas atau kata sandi salah.',
  AUTH_LOCKED: 'Akun dikunci sementara setelah 5 kali gagal login.',
  FORBIDDEN_ROLE: 'Peran aktif Anda tidak punya akses ke aksi ini.',
  KRS_WINDOW_CLOSED: 'Jendela KRS sedang tidak dibuka.',
  KRS_STUDENT_NOT_ACTIVE: 'Status Anda bukan aktif pada periode ini.',
  KRS_UNPAID: 'Tagihan wajib belum lunas dan belum ada dispensasi.',
  KRS_QUOTA_FULL: 'Kelas sudah penuh.',
  KRS_SKS_LIMIT: 'Melebihi batas SKS Anda.',
  KRS_PREREQ_NOT_MET: 'Prasyarat mata kuliah belum terpenuhi.',
  KRS_SCHEDULE_CLASH: 'Bentrok dengan jadwal kelas lain di KRS Anda.',
  KRS_INVALID_STATE: 'KRS tidak bisa diubah pada status ini.',
  KRS_DUPLICATE_COURSE: 'Mata kuliah ini sudah ada di KRS.',
  SCHEDULE_ROOM_CLASH: 'Ruang sudah dipakai di slot yang sama.',
  SCHEDULE_LECTURER_CLASH: 'Dosen sudah mengajar di slot yang sama.',
  GRADE_LOCKED: 'Nilai sudah dikunci. Ajukan koreksi nilai.',
  GRADE_NOT_ALLOWED: 'Anda tidak berhak mengubah nilai kelas ini.',
  PERIOD_ALREADY_ACTIVE: 'Sudah ada periode aktif lain.',
  CUTI_LIMIT_REACHED: 'Batas cuti akademik sudah tercapai.',
  PMB_INVALID_TRANSITION: 'Perubahan status pendaftar tidak valid.',
  PDDIKTI_BLOCKED: 'Ekspor diblokir: masih ada error validasi.',
  NETWORK: 'Tidak dapat terhubung ke server.',
}

export function errorText(code: string, fallback?: string) {
  return ERROR_TEXT[code] ?? fallback ?? 'Terjadi kesalahan.'
}
