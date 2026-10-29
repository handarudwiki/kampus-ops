import { reactive } from 'vue'
import { ApiError } from '@/api/client'
import { errorText } from './status'

/** Toasts + confirm dialogs. Use these instead of alert()/confirm(). */
export interface Toast { id: number; level: 'success' | 'error' | 'info'; title: string; body?: string }
export const toasts = reactive<Toast[]>([])
let seq = 0

export function toast(level: Toast['level'], title: string, body?: string) {
  const id = ++seq
  toasts.push({ id, level, title, body })
  setTimeout(() => dismiss(id), level === 'error' ? 6000 : 3500)
}
export function dismiss(id: number) {
  const i = toasts.findIndex((t) => t.id === id)
  if (i >= 0) toasts.splice(i, 1)
}

/** Turn any thrown error into a consistent toast. */
export function toastError(e: unknown, title = 'Gagal') {
  if (e instanceof ApiError) toast('error', title, errorText(e.code, e.message))
  else toast('error', title, String(e))
}

export interface ConfirmReq {
  title: string
  body?: string
  confirmLabel?: string
  danger?: boolean
  /** ask for a free-text reason (reject KRS, koreksi nilai, cancel kelas) */
  withReason?: boolean
  reasonLabel?: string
  resolve: (v: { ok: boolean; reason: string }) => void
}
export const confirmState = reactive<{ current: ConfirmReq | null }>({ current: null })

export function confirm(opts: Omit<ConfirmReq, 'resolve'>) {
  return new Promise<{ ok: boolean; reason: string }>((resolve) => {
    confirmState.current = { ...opts, resolve }
  })
}
