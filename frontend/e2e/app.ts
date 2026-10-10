import { readFileSync } from 'node:fs'
import { expect, type Page } from '@playwright/test'
import { SANDBOX_GAMES } from './sandbox.ts'

const CATALOG = new URL('../../internal/components/components.json', import.meta.url)

/** Catalog games the sandbox enables for itself (scripts/selftest.sh MORTAR_SELFTEST_ENABLE, see sandbox.ts). */
const SANDBOX_ENABLED = SANDBOX_GAMES.split(',')

/** Game select shows a tile for each game the embedded catalog enables, and for each the sandbox enables beside them. */
export const ENABLED_GAMES = (() => {
  const { games } = JSON.parse(readFileSync(CATALOG, 'utf8')) as {
    games: { id: string; enabled: boolean }[]
  }
  return games.filter((g) => g.enabled || SANDBOX_ENABLED.includes(g.id)).length
})()

/** Opens the seeded Stardew Valley on the Seed Farm profile with the first-run tour dismissed. */
export async function openSeedFarm(page: Page) {
  await page.goto('/')
  const welcome = page.getByRole('button', { name: 'Continue' })
  const farm = page.getByRole('button', { name: /^(Open )?Seed Farm/ }).first()
  // Mortar reopens the last game, which an earlier spec may have left on another one.
  const otherGame = page.getByRole('tablist', { name: 'Profile sections' })
  await expect(welcome.or(farm).or(otherGame).first()).toBeVisible({ timeout: 30_000 })
  if (await welcome.isVisible()) {
    await welcome.click()
  }
  if (!(await farm.isVisible())) {
    await openGameSelect(page)
  }
  await farm.click()
  await expect(page.getByRole('tab', { name: 'Mods' })).toBeVisible()
  // The tour shows once per sandbox, so only the first spec sees it.
  const skip = page.getByRole('button', { name: 'Skip tour' })
  if (await skip.isVisible({ timeout: 1500 }).catch(() => false)) {
    await skip.click()
  }
}

/** Switches the open game's profile from the title bar's profile menu. */
export async function switchProfile(page: Page, name: string) {
  await page.getByRole('button', { name: /^Switch profile/ }).click()
  await page.getByRole('menuitemradio', { name: new RegExp(`^${name}`) }).click()
}

/** Opens the open game's settings from the Home tab. */
export async function openGameSettings(page: Page, game = 'Stardew Valley') {
  await page.getByRole('tab', { name: 'Home' }).click()
  await page.getByRole('button', { name: `${game} settings` }).click()
}

/** Opens Game select from the title bar's game menu. */
export async function openGameSelect(page: Page) {
  await page.getByRole('button', { name: /^(Switch game|Choose a game)/ }).click()
  await page.getByRole('menuitem', { name: /^All games/ }).click()
}

/** Opens Mortar's settings from the Mortar menu. */
export async function openSettings(page: Page) {
  await page.getByRole('button', { name: 'Mortar menu' }).click()
  await page.getByRole('menuitem', { name: /^Mortar settings/ }).click()
}
