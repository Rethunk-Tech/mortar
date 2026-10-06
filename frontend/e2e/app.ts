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
  await expect(welcome.or(farm)).toBeVisible({ timeout: 30_000 })
  if (await welcome.isVisible()) {
    await welcome.click()
  }
  await farm.click()
  await expect(page.getByRole('tab', { name: 'Mods' })).toBeVisible()
  // The tour shows once per sandbox, so only the first spec sees it.
  const skip = page.getByRole('button', { name: 'Skip tour' })
  if (await skip.isVisible({ timeout: 1500 }).catch(() => false)) {
    await skip.click()
  }
}
