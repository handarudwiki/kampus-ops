import type { AuthResponse, ListMeta } from './types'

const BASE = import.meta.env.VITE_API_BASE ?? '/api/v1'

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public details?: unknown,
  ) {
    super(message)
  }
}

/** Session state lives here (not in Pinia) so the client works without a store. */
export const session = {
  get access() { return read('ko-access') },
  get refresh() { return read('ko-refresh') },
  get role() { return read('ko-role') },
  set(auth: Pick<AuthResponse, 'access_token' | 'refresh_token'>) {
    write('ko-access', auth.access_token)
    write('ko-refresh', auth.refresh_token)
  },
  setRole(role: string) { write('ko-role', role) },
  clear() { ['ko-access', 'ko-refresh', 'ko-role'].forEach((k) => write(k, null)) },
}

function read(k: string) {
  try { return localStorage.getItem(k) } catch { return null }
}
function write(k: string, v: string | null) {
  try { v == null ? localStorage.removeItem(k) : localStorage.setItem(k, v) } catch { /* private mode */ }
}

let onUnauthorized: () => void = () => {}
export function setUnauthorizedHandler(fn: () => void) { onUnauthorized = fn }

let refreshing: Promise<boolean> | null = null
async function tryRefresh(): Promise<boolean> {
  const token = session.refresh
  if (!token) return false
  refreshing ??= fetch(`${BASE}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: token }),
  })
    .then(async (r) => {
      if (!r.ok) return false
      session.set((await r.json()).data as AuthResponse)
      return true
    })
    .catch(() => false)
    .finally(() => { refreshing = null })
  return refreshing
}

export interface RequestOpts {
  query?: Record<string, string | number | boolean | null | undefined>
  body?: unknown
  idempotent?: boolean
  raw?: boolean
}

async function request<T>(method: string, path: string, opts: RequestOpts = {}, retry = true): Promise<{ data: T; meta?: ListMeta }> {
  const url = new URL(BASE + path, window.location.origin)
  for (const [k, v] of Object.entries(opts.query ?? {})) {
    if (v !== undefined && v !== null && v !== '') url.searchParams.set(k, String(v))
  }
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (opts.body !== undefined) headers['Content-Type'] = 'application/json'
  if (session.access) headers.Authorization = `Bearer ${session.access}`
  if (session.role) headers['X-Active-Role'] = session.role
  if (opts.idempotent) headers['Idempotency-Key'] = crypto.randomUUID()

  let res: Response
  try {
    res = await fetch(url, { method, headers, body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined })
  } catch {
    throw new ApiError(0, 'NETWORK', 'Tidak dapat terhubung ke server')
  }

  if (res.status === 401 && retry && !path.startsWith('/auth/')) {
    if (await tryRefresh()) return request<T>(method, path, opts, false)
    session.clear()
    onUnauthorized()
  }
  if (res.status === 204) return { data: undefined as T }
  if (opts.raw) {
    if (!res.ok) throw await toError(res)
    return { data: (await res.blob()) as T }
  }
  if (!res.ok) throw await toError(res)
  return res.json()
}

async function toError(res: Response) {
  let body: { error?: { code?: string; message?: string; details?: unknown } } = {}
  try { body = await res.json() } catch { /* non-JSON */ }
  return new ApiError(
    res.status,
    body.error?.code ?? `HTTP_${res.status}`,
    body.error?.message ?? res.statusText,
    body.error?.details,
  )
}

export const api = {
  get: <T>(p: string, query?: RequestOpts['query']) => request<T>('GET', p, { query }).then((r) => r.data),
  list: <T>(p: string, query?: RequestOpts['query']) => request<T[]>('GET', p, { query }),
  post: <T>(p: string, body?: unknown, o: Omit<RequestOpts, 'body'> = {}) => request<T>('POST', p, { ...o, body: body ?? {} }).then((r) => r.data),
  put: <T>(p: string, body?: unknown) => request<T>('PUT', p, { body }).then((r) => r.data),
  del: <T>(p: string, o: Omit<RequestOpts, 'body'> = {}) => request<T>('DELETE', p, o).then((r) => r.data),
  blob: (p: string, query?: RequestOpts['query']) => request<Blob>('GET', p, { query, raw: true }).then((r) => r.data),
}
