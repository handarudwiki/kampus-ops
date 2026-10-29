import type { Page } from '@playwright/test'

export const PASSWORD = 'password123'

/** Log in through the API and seed the SPA session so each shot starts on the right role. */
export async function loginAs(page: Page, identifier: string, role: string, tenant = 'UNIKA') {
  const res = await page.request.post('/api/v1/auth/login', { data: { tenant_code: tenant, identifier, password: PASSWORD } })
  if (!res.ok()) throw new Error(`login ${identifier} failed: ${res.status()} ${await res.text()}`)
  const { data } = await res.json()
  await page.addInitScript(([a, r, role]) => {
    localStorage.setItem('ko-access', a)
    localStorage.setItem('ko-refresh', r)
    localStorage.setItem('ko-role', role)
  }, [data.access_token, data.refresh_token, role] as const)
  return data.access_token as string
}

/** Authenticated API GET for looking up IDs to deep-link into detail pages. */
export async function apiGet<T>(page: Page, token: string, role: string, path: string): Promise<T> {
  const res = await page.request.get('/api/v1' + path, { headers: { Authorization: `Bearer ${token}`, 'X-Active-Role': role } })
  if (!res.ok()) throw new Error(`GET ${path}: ${res.status()}`)
  return (await res.json()).data as T
}

/** Wait for skeletons/spinners to settle so screenshots show real data. */
export async function settle(page: Page) {
  await page.waitForLoadState('networkidle')
  await page.waitForFunction(() => !document.querySelector('[aria-busy="true"], .animate-pulse:not(.ko-live), .animate-spin'), null, { timeout: 15_000 }).catch(() => {})
  await page.waitForTimeout(400)
}
