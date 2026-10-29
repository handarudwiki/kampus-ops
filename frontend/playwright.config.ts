import { defineConfig } from '@playwright/test'

/**
 * Two projects:
 *  - screenshots: regenerates ./screenshots (npm run screenshots)
 *  - e2e: smoke flows (npm run e2e)
 * Expects the app at BASE_URL (default: Vite dev server proxying to the API).
 */
export default defineConfig({
  testDir: './e2e',
  timeout: 180_000,
  workers: 1,
  use: {
    baseURL: process.env.BASE_URL ?? 'http://localhost:5173',
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 2,
    colorScheme: 'light',
  },
  projects: [
    { name: 'screenshots', testMatch: /screenshots\.spec\.ts/ },
    { name: 'e2e', testMatch: /flows\.spec\.ts/ },
  ],
})
