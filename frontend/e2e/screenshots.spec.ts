import { test, type Page } from '@playwright/test'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { apiGet, loginAs, settle } from './helpers'

const OUT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../screenshots')

async function shot(page: Page, name: string, url?: string, opts: { full?: boolean } = {}) {
  if (url) await page.goto(url)
  await settle(page)
  await page.screenshot({ path: `${OUT}/${name}.png`, fullPage: opts.full ?? true })
}

test('00 · public pages', async ({ page }) => {
  await shot(page, '01-login', '/login', { full: false })
  await shot(page, '02-pmb-daftar', '/daftar', { full: false })
  await shot(page, '03-design-system', '/design-system')
})

test('10 · mahasiswa', async ({ page }) => {
  const token = await loginAs(page, '2024110001', 'mahasiswa')
  // Build a realistic draft KRS: take eligible classes until ~15 SKS
  const offers = await apiGet<{ id: string; sks: number; eligible: boolean; in_krs: boolean; mk_kode: string }[]>(page, token, 'mahasiswa', '/krs/me/offerings')
  let sks = 0
  const taken = new Set<string>()
  for (const o of offers) {
    if (!o.eligible || o.in_krs || taken.has(o.mk_kode) || sks >= 15) continue
    const r = await page.request.post('/api/v1/krs/me/items', {
      data: { kelas_id: o.id },
      headers: { Authorization: `Bearer ${token}`, 'X-Active-Role': 'mahasiswa', 'Idempotency-Key': crypto.randomUUID() },
    })
    if (r.ok()) { sks += o.sks; taken.add(o.mk_kode) }
  }
  await shot(page, '10-mahasiswa-dashboard', '/')
  await shot(page, '11-mahasiswa-krs', '/krs')
  await shot(page, '12-mahasiswa-jadwal', '/jadwal')
  await shot(page, '13-mahasiswa-khs', '/khs')
  await shot(page, '14-mahasiswa-transkrip', '/transkrip')
  await shot(page, '15-mahasiswa-tagihan', '/tagihan')
  await shot(page, '16-mahasiswa-profil', '/profil')
})

test('17 · mahasiswa unpaid is blocked from KRS', async ({ page }) => {
  await loginAs(page, '2024110002', 'mahasiswa')
  await shot(page, '17-mahasiswa-krs-diblokir', '/krs', { full: false })
})

test('20 · dosen wali', async ({ page }) => {
  const token = await loginAs(page, '0011223344', 'dosen_wali')
  await shot(page, '20-dosenwali-dashboard', '/')
  await shot(page, '21-dosenwali-perwalian', '/perwalian')
  const list = await apiGet<{ id: string }[]>(page, token, 'dosen_wali', '/krs/advisees?status=diajukan')
  const any = list[0] ?? (await apiGet<{ id: string }[]>(page, token, 'dosen_wali', '/krs/advisees'))[0]
  if (any) await shot(page, '22-dosenwali-krs-detail', `/krs/${any.id}`)
})

test('30 · dosen pengampu', async ({ page }) => {
  const token = await loginAs(page, '0011223344', 'dosen')
  await shot(page, '30-dosen-mengajar', '/mengajar')
  const kelas = await apiGet<{ id: string; terisi: number }[]>(page, token, 'dosen', '/teaching/kelas')
  const busiest = [...kelas].sort((a, b) => b.terisi - a.terisi)[0]
  if (busiest) {
    await shot(page, '31-dosen-presensi', `/kelas/${busiest.id}`)
    await page.goto(`/kelas/${busiest.id}/nilai`)
    await settle(page)
    // Fill a few grades so the weighted preview is visible
    const inputs = page.locator('input[type=number]')
    const n = Math.min(await inputs.count(), 18)
    for (let i = 0; i < n; i++) await inputs.nth(i).fill(String(70 + ((i * 7) % 28)))
    await shot(page, '32-dosen-nilai')
  }
  await shot(page, '33-dosen-koreksi', '/koreksi-nilai')
})

test('40 · kaprodi', async ({ page }) => {
  await loginAs(page, '0011223344', 'kaprodi')
  await shot(page, '40-kaprodi-dashboard', '/')
  await shot(page, '41-kaprodi-kurikulum', '/kurikulum')
  await shot(page, '42-kaprodi-koreksi-nilai', '/koreksi-nilai')
  await shot(page, '43-kaprodi-masa-studi', '/masa-studi')
})

test('50 · operator akademik', async ({ page }) => {
  const token = await loginAs(page, 'baak@unika.ac.id', 'operator')
  await shot(page, '50-operator-dashboard', '/')
  await shot(page, '51-operator-mahasiswa', '/mahasiswa')
  const m = await apiGet<{ id: string }[]>(page, token, 'operator', '/mahasiswa?per_page=1')
  if (m[0]) await shot(page, '52-operator-mahasiswa-detail', `/mahasiswa/${m[0].id}`)
  await shot(page, '53-operator-kelas', '/kelas')
  await page.goto('/kelas')
  await settle(page)
  await page.getByRole('button', { name: 'Tambah kelas' }).click()
  await shot(page, '54-operator-kelas-form', undefined, { full: false })
  await page.goto('/pddikti')
  await settle(page)
  await page.getByRole('button', { name: 'Jalankan validator' }).click()
  await shot(page, '55-operator-pddikti')
  await shot(page, '56-operator-pmb', '/pmb')
  await shot(page, '57-operator-audit-log', '/audit-log')
  await shot(page, '58-operator-dosen', '/dosen')
  await shot(page, '59-operator-periode', '/periode')
})

test('60 · keuangan', async ({ page }) => {
  await loginAs(page, 'keuangan@unika.ac.id', 'keuangan')
  await shot(page, '60-keuangan-dashboard', '/')
  await shot(page, '61-keuangan-tagihan', '/keuangan/tagihan')
  await shot(page, '62-keuangan-dispensasi', '/keuangan/dispensasi')
  await shot(page, '63-keuangan-piutang', '/keuangan/piutang')
})

test('70 · admin & vendor', async ({ page, browser }) => {
  await loginAs(page, 'admin@unika.ac.id', 'admin_kampus')
  await shot(page, '70-admin-dashboard', '/')
  await shot(page, '71-admin-pengguna', '/pengguna')
  const p2 = await browser.newPage({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 2 })
  await loginAs(p2, 'super@kampusops.id', 'super_admin')
  await shot(p2, '72-superadmin-tenants', '/tenants')
  const p3 = await browser.newPage({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 2 })
  await loginAs(p3, 'calon@mail.com', 'calon_mahasiswa')
  await shot(p3, '73-calon-status-pmb', '/')
})

test('80 · UX details', async ({ browser }) => {
  // Command palette
  const p = await browser.newPage({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 2 })
  await loginAs(p, 'baak@unika.ac.id', 'operator')
  await p.goto('/')
  await settle(p)
  await p.keyboard.press('Control+k')
  await p.keyboard.type('pdd')
  await shot(p, '80-command-palette', undefined, { full: false })

  // Dark mode
  const d = await browser.newPage({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 2, colorScheme: 'dark' })
  await loginAs(d, 'baak@unika.ac.id', 'operator')
  await shot(d, '81-dark-operator-dashboard', '/')
  const d2 = await browser.newPage({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 2, colorScheme: 'dark' })
  await loginAs(d2, '2024110001', 'mahasiswa')
  await shot(d2, '82-dark-mahasiswa-krs', '/krs')

  // Mobile 390px
  const m = await browser.newPage({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 3, isMobile: true, hasTouch: true })
  await loginAs(m, '2024110001', 'mahasiswa')
  await shot(m, '83-mobile-dashboard', '/', { full: false })
  await shot(m, '84-mobile-krs', '/krs', { full: false })
  await shot(m, '85-mobile-tagihan', '/tagihan', { full: false })
})
