import type { PmbStatus } from '@/api/types'
import { STATUS } from '@/design-system'

/** FR-09 happy path. tidak_lulus is a terminal branch shown as a failed "lulus" step. */
export const PMB_FLOW: PmbStatus[] = ['daftar', 'bayar_formulir', 'lengkapi_berkas', 'verifikasi', 'lulus', 'daftar_ulang']
export const PMB_STEPS = PMB_FLOW.map((k) => ({ key: k, label: STATUS.pmb[k].label }))

/** Next allowed statuses from the current one (mirrors backend transition rules). */
export const PMB_NEXT: Record<PmbStatus, PmbStatus[]> = {
  daftar: ['bayar_formulir'],
  bayar_formulir: ['lengkapi_berkas'],
  lengkapi_berkas: ['verifikasi'],
  verifikasi: ['lulus', 'tidak_lulus'],
  lulus: [],
  tidak_lulus: [],
  daftar_ulang: [],
}
