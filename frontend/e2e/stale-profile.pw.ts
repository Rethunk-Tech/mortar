import { execFileSync } from 'node:child_process'
import process from 'node:process'
import { expect, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'
import { serverEnv } from './sandbox.ts'

const dir = process.env.MORTAR_E2E_DIR ?? ''
const cli = (...args: string[]) =>
  execFileSync(`${dir}/mortar-server`, args, { env: serverEnv(dir), encoding: 'utf8' }).trim()

test('a remembered profile deleted behind the app falls back without an error', async ({
  page,
}) => {
  const id = cli('profile', 'create', 'lethal-company', 'Stale Lobby').split('\t')[0] ?? ''
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'Game select' }).click()
  await page
    .getByRole('button', { name: 'Open Lethal Company' })
    .click({ position: { x: 8, y: 8 } })
  await page
    .getByRole('button', { name: /^(Open )?Stale Lobby/ })
    .first()
    .click()
  await expect(page.getByRole('tab', { name: 'Mods' })).toBeVisible()

  cli('profile', 'delete', 'lethal-company', id)
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  page.on('response', (r) => {
    if (r.status() >= 400) {
      errors.push(`${r.status()} ${r.url()}`)
    }
  })
  await page.reload()
  // Mortar reopens the last game, now on its remaining profile.
  await expect(page.getByRole('tab', { name: 'Mods' })).toBeVisible()
  await expect(page.getByRole('button', { name: /^(Open )?Seed Lobby/ }).first()).toBeVisible()
  // Startup checks and the profile load both read the remembered profile.
  await page.waitForLoadState('networkidle')
  expect(errors).toEqual([])
})
