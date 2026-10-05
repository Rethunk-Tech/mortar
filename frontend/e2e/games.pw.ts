import { expect, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

// Browse reaches the real Nexus and GitHub search endpoints from the sandbox; the suite has no offline network
// fixture, so that spec needs a connection.
const SEARCH_WAIT_MS = 20_000

test('Game Select lists Stardew Valley before Lethal Company, which the sandbox enables', async ({
  page,
}) => {
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'Game select' }).click()
  const stardew = page.getByRole('button', { name: 'Open Stardew Valley' })
  const lethal = page.getByText('Lethal Company', { exact: true })
  await expect(lethal).toBeVisible()
  const [s, l] = await Promise.all([stardew.boundingBox(), lethal.boundingBox()])
  expect(s && l && (s.y < l.y || (s.y === l.y && s.x < l.x))).toBe(true)
  // The centre of the tile is its profile cards, which open their own profile.
  await stardew.click({ position: { x: 8, y: 8 } })
  await expect(page.getByRole('tab', { name: 'Mods' })).toBeVisible()
})

test('hovering a Game Select tile grows it and dims the others without changing layout', async ({
  page,
}) => {
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'Game select' }).click()
  const tiles = page.locator('[data-tile]')
  await expect(tiles).toHaveCount(2)
  const layout = () => tiles.evaluateAll((els) => els.map((e) => [e.clientWidth, e.clientHeight]))
  const before = await layout()
  await tiles.first().hover()
  await expect(tiles.last()).toHaveCSS('filter', 'brightness(0.6)')
  await expect(tiles.first()).toHaveCSS('transform', 'matrix(1.04, 0, 0, 1.04, 0, 0)')
  expect(await layout()).toEqual(before)
  const grid = tiles.first().locator('..')
  expect(await grid.evaluate((e) => e.scrollWidth <= e.clientWidth)).toBe(true)
})

test('Browse defaults to All sources and a search shows results from more than one source', async ({
  page,
}) => {
  await openSeedFarm(page)
  await page.getByRole('tab', { name: 'Browse' }).click()
  await expect(page.getByRole('button', { name: 'All sources' })).toHaveAttribute(
    'aria-pressed',
    'true',
  )
  await page.getByLabel('Search all mod sites').fill('farm')
  await expect(page.getByLabel('Open on Nexus').first()).toBeVisible({ timeout: SEARCH_WAIT_MS })
  await expect(page.getByLabel('Open on GitHub').first()).toBeVisible({ timeout: SEARCH_WAIT_MS })
})

test('the game title opens a switcher listing both playable games', async ({ page }) => {
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'Stardew Valley' }).first().click()
  const switcher = page.getByRole('dialog', { name: 'Switch game' })
  await expect(switcher.getByRole('button', { name: 'Open Lethal Company' })).toBeVisible()
  await expect(switcher.getByRole('button', { name: 'Open Stardew Valley' })).toHaveAttribute(
    'aria-current',
    'true',
  )
  await page.keyboard.press('Escape')
  await expect(switcher).toBeHidden()
})

test('Settings › Mods and profiles has the adult-mods switch, off by default', async ({ page }) => {
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'Mortar menu' }).click()
  await page.getByRole('button', { name: 'Settings', exact: true }).click()
  await page.getByRole('navigation').getByRole('button', { name: 'Mods and profiles' }).click()
  const adult = page.getByLabel('Show adult mods in browse')
  await expect(adult).toBeVisible()
  await expect(adult).not.toBeChecked()
})
