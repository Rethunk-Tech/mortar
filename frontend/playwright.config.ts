import { defineConfig, devices } from '@playwright/test'
import { sandboxPort } from './e2e/sandbox.ts'

const port = sandboxPort()

// Starts and seeds its own self-test sandbox in a folder and on a port of its own (e2e/sandbox.ts), never the real data folder; Chromium only.
export default defineConfig({
  testDir: './e2e',
  // Not .spec/.test, so bun test leaves these to Playwright.
  testMatch: '**/*.pw.ts',
  timeout: 30_000,
  expect: { timeout: 5000 },
  workers: 1,
  globalSetup: './e2e/global-setup.ts',
  reporter: [['list'], ['./e2e/failure-reporter.ts']],
  use: {
    ...devices['Desktop Chrome'],
    baseURL: `http://127.0.0.1:${port}`,
    viewport: { width: 1400, height: 840 },
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})
