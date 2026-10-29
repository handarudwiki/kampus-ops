/** All number/date formatting goes through here so every screen reads the same. */

const rupiahFmt = new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 })
const numFmt = new Intl.NumberFormat('id-ID')

export const rupiah = (n: number | null | undefined) => (n == null ? '—' : rupiahFmt.format(n))

/** Compact rupiah for stat cards: Rp 1,2 M / Rp 450 jt */
export function rupiahShort(n: number) {
  const abs = Math.abs(n)
  if (abs >= 1e12) return `Rp ${(n / 1e12).toLocaleString('id-ID', { maximumFractionDigits: 1 })} T`
  if (abs >= 1e9) return `Rp ${(n / 1e9).toLocaleString('id-ID', { maximumFractionDigits: 1 })} M`
  if (abs >= 1e6) return `Rp ${(n / 1e6).toLocaleString('id-ID', { maximumFractionDigits: 1 })} jt`
  return rupiah(n)
}

export const num = (n: number | null | undefined) => (n == null ? '—' : numFmt.format(n))

export function date(iso: string | null | undefined, opts: Intl.DateTimeFormatOptions = { day: 'numeric', month: 'short', year: 'numeric' }) {
  if (!iso) return '—'
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleDateString('id-ID', opts)
}

export function dateTime(iso: string | null | undefined, timeZone?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString('id-ID', {
    day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', timeZone,
  })
}

export function relative(iso: string) {
  const diff = (Date.now() - new Date(iso).getTime()) / 1000
  const rtf = new Intl.RelativeTimeFormat('id-ID', { numeric: 'auto' })
  if (Math.abs(diff) < 60) return 'baru saja'
  if (Math.abs(diff) < 3600) return rtf.format(-Math.round(diff / 60), 'minute')
  if (Math.abs(diff) < 86400) return rtf.format(-Math.round(diff / 3600), 'hour')
  return rtf.format(-Math.round(diff / 86400), 'day')
}

export const HARI = ['', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'] as const

export function slot(s: { hari: number; jam_mulai: string; jam_selesai: string }) {
  return `${HARI[s.hari]}, ${s.jam_mulai}–${s.jam_selesai}`
}

/** bobot_x100 (EC-10) → "3,75". Integer math only; no float drift. */
export function bobot(x100: number | null | undefined) {
  if (x100 == null) return '—'
  const whole = Math.trunc(x100 / 100)
  const frac = Math.abs(x100 % 100).toString().padStart(2, '0')
  return `${whole},${frac}`
}

/** API sends IPS/IPK as "3.42"; display with Indonesian decimal comma. */
export const ip = (s: string | null | undefined) => (s ? s.replace('.', ',') : '—')

/**
 * Split a display number into a strong head and a muted tail, the signature
 * "253,056" treatment on stat cards. "12.375" → ["12", ".375"].
 */
export function splitFigure(v: string): [string, string] {
  const m = v.match(/^(.*?\d)([.,]\d+.*)$/)
  return m ? [m[1], m[2]] : [v, '']
}

export const initials = (name: string) =>
  name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]!.toUpperCase()).join('')
