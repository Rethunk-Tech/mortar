import { readFileSync } from 'node:fs'
import { expect, type Page } from '@playwright/test'

const CATALOG = new URL('../../internal/components/components.json', import.meta.url)

/** Game select shows a tile for each game the embedded catalog enables. */
export const ENABLED_GAMES = (
  JSON.parse(readFileSync(CATALOG, 'utf8')) as { games: { enabled: boolean }[] }
).games.filter((g) => g.enabled).length

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
