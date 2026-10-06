import { execFileSync } from 'node:child_process'
import process from 'node:process'
import { expect, type Page, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'
import { serverEnv } from './sandbox.ts'

// `mortar open` starts a second process, which a server build forwards to the running sandbox over the control
// channel, as a desktop build's single-instance lock does for the browser's "Open in Mortar" and a double-clicked file.

const dir = process.env.MORTAR_E2E_DIR ?? ''
let env: Record<string, string> = {}
const cli = (...args: string[]) =>
  execFileSync(`${dir}/mortar-server`, args, { env, encoding: 'utf8' }).trim()

const dialog = (page: Page) => page.getByRole('dialog', { name: 'Import a profile' })

test.beforeAll(() => {
  env = serverEnv(dir)
})

test('opening a share link lands on its import preview', async ({ page }) => {
  await openSeedFarm(page)
  cli('open', cli('share', 'stardew', 'Seed Farm'))
  await expect(dialog(page).getByText('Seed Farm').first()).toBeVisible()
  await dialog(page).getByLabel('Close').click()
})

test('opening a .mortar file lands on its import preview', async ({ page }) => {
  const file = `${dir}/open-farm.mortar`
  cli('export', 'stardew', 'Seed Farm', file)
  await openSeedFarm(page)
  cli('open', file)
  await expect(dialog(page).getByText('Seed Farm').first()).toBeVisible()
  await dialog(page).getByLabel('Close').click()
})

test('a .mortar file for another game is refused', async ({ page }) => {
  const file = `${dir}/open-other.mortar`
  const created = cli('profile', 'create', 'lethal-company', 'Other Game').split('\t')[0] ?? ''
  cli('export', 'lethal-company', created, file)
  await openSeedFarm(page)
  cli('open', file)
  await expect(
    page.getByText('That is for another game. Open that game and import it there.'),
  ).toBeVisible()
})
