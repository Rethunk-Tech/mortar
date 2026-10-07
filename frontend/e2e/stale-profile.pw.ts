import { execFileSync } from 'node:child_process'
import process from 'node:process'
import { expect, test } from '@playwright/test'
import { openGameSelect, openSeedFarm } from './app.ts'
import { serverEnv } from './sandbox.ts'

const dir = process.env.MORTAR_E2E_DIR ?? ''
const cli = (...args: string[]) =>
  execFileSync(`${dir}/mortar-server`, args, { env: serverEnv(dir), encoding: 'utf8' }).trim()

test('a remembered profile deleted behind the app falls back without an error', async ({
  page,
}) => {
  const id = cli('profile', 'create', 'lethal-company', 'Stale Lobby').split('\t')[0] ?? ''
  await openSeedFarm(page)
  await openGameSelect(page)
  await page.getByRole('button', { name: /^Stale Lobby/ }).click()
  await expect(page.getByRole('tab', { name: 'Mods' })).toBeVisible()

  cli('profile', 'delete', 'lethal-company', id)
  // The open page may still poll the deleted profile for a moment; only what the next start does is judged.
  await page.goto('about:blank')
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  page.on('response', (r) => {
    if (r.status() >= 400) {
      errors.push(`${r.status()} ${r.url()} ${r.request().postData() ?? ''}`)
    }
  })
  await page.goto('/')
  // Mortar reopens the last game, now on its remaining profile.
  await expect(page.getByRole('tab', { name: 'Mods' })).toBeVisible()
  await expect(page.getByRole('button', { name: /^Switch profile: Seed Lobby/ })).toBeVisible()
  // Startup checks and the profile load both read the remembered profile.
  await page.waitForLoadState('networkidle')
  expect(errors).toEqual([])
})
