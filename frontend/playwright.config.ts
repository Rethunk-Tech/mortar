import { defineConfig, devices } from '@playwright/test'

// Starts and seeds its own self-test sandbox on port 9465 (e2e/sandbox.ts), never the real data folder; Chromium only.
export default defineConfig({
  testDir: './e2e',
  // Not .spec/.test, so bun test leaves these to Playwright.
  testMatch: '**/*.pw.ts',
  timeout: 30_000,
  expect: { timeout: 5000 },
  workers: 1,
  globalSetup: './e2e/global-setup.ts',
  globalTeardown: './e2e/global-teardown.ts',
  reporter: 'list',
  use: {
    ...devices['Desktop Chrome'],
    baseURL: 'http://127.0.0.1:9465',
    viewport: { width: 1400, height: 840 },
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})
