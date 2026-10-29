import { expect, test } from '@playwright/test'
import { loginAs, settle } from './helpers'

/** Smoke flows over the seeded demo. Run against a fresh seed. */
test('demo accounts land on their dashboard', async ({ page }) => {
  for (const [id, role] of [['2024110001', 'mahasiswa'], ['baak@unika.ac.id', 'operator'], ['keuangan@unika.ac.id', 'keuangan']] as const) {
    await page.context().clearCookies()
    await loginAs(page, id, role)
    await page.goto('/')
    await settle(page)
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Halo')
  }
})

test('unpaid student sees why KRS is blocked', async ({ page }) => {
  await loginAs(page, '2024110002', 'mahasiswa')
  await page.goto('/krs')
  await settle(page)
  await expect(page.getByText('Anda belum bisa mengisi KRS')).toBeVisible()
})

test('cross-tenant record returns not found (EC-16)', async ({ page }) => {
  await loginAs(page, 'baak@unika.ac.id', 'operator')
  await page.goto('/mahasiswa/00000000-0000-0000-0000-000000000000')
  await settle(page)
  await expect(page.getByText('Gagal memuat data')).toBeVisible()
})
