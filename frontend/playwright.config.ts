import { defineConfig, devices } from '@playwright/test'

// Runs against the self-test server (scripts/selftest.sh), never the real data folder; Chromium only.
export default defineConfig({
  testDir: './e2e',
  // Not .spec/.test, so bun test leaves these to Playwright.
  testMatch: '**/*.pw.ts',
  timeout: 120_000,
  workers: 1,
  reporter: 'list',
  use: {
    ...devices['Desktop Chrome'],
    baseURL: process.env.MORTAR_URL ?? 'http://127.0.0.1:9455',
    viewport: { width: 1400, height: 840 },
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})
