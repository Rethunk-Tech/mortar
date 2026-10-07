import { expect, type Page, test } from '@playwright/test'
import { openGameSettings, openSeedFarm, openSettings } from './app.ts'

/** Every row label on every page of the open settings screen, read with the search empty. */
async function rowLabels(page: Page): Promise<string[]> {
  const nav = page.getByRole('navigation', { name: 'Settings sections' })
  await expect(nav).toBeVisible()
  const labels = new Set<string>()
  // The page buttons; Back is the only one named by aria-label.
  for (const button of await nav.locator('button:not([aria-label])').all()) {
    await button.click()
    await page.waitForTimeout(50)
    for (const text of await page.locator('[data-setting-label]').allInnerTexts()) {
      labels.add(text.trim())
    }
  }
  return [...labels]
}

/** Searching for each label, and for each synonym, shows a row with that label. */
async function expectSearchFinds(page: Page, queries: [string, string][]) {
  const search = page.getByRole('textbox', { name: 'Search settings' })
  const missed: string[] = []
  for (const [query, label] of queries) {
    await search.fill(query)
    const row = page.locator('[data-setting-label]').getByText(label, { exact: true })
    // The page filters after a short debounce, so a row that is not there yet gets a moment.
    const found = await expect(row.first())
      .toBeVisible({ timeout: 1500 })
      .then(
        () => true,
        () => false,
      )
    if (!found) {
      missed.push(`${query} -> ${label}`)
    }
  }
  await search.fill('')
  expect(missed, 'settings search misses').toEqual([])
}

const SYNONYMS: [string, string][] = [
  ['dark', 'Theme'],
  ['firewall', 'Share profiles on the local network'],
  ['startup', 'Keep Mortar in the tray'],
]

test('settings search finds every row by its label, on Mortar and game settings', async ({
  page,
}) => {
  await openSeedFarm(page)
  await openSettings(page)
  const app = await rowLabels(page)
  expect(app.length).toBeGreaterThan(40)
  await expectSearchFinds(page, [...app.map((l): [string, string] => [l, l]), ...SYNONYMS])
  await page.getByRole('button', { name: 'Back' }).click()

  await openGameSettings(page)
  const game = await rowLabels(page)
  expect(game.length).toBeGreaterThan(10)
  await expectSearchFinds(
    page,
    game.map((l) => [l, l]),
  )
})

test('a Mortar settings search that finds little offers the same search in the game settings', async ({
  page,
}) => {
  await openSeedFarm(page)
  await openSettings(page)
  await page.getByRole('textbox', { name: 'Search settings' }).fill('Run logs kept')
  await page.getByRole('button', { name: 'Search Stardew Valley settings' }).click()
  await expect(page.getByRole('textbox', { name: 'Search settings' })).toHaveValue('Run logs kept')
  await expect(
    page.locator('[data-setting-label]').getByText('Run logs kept', { exact: true }),
  ).toBeVisible()
})

test('a search for a per-profile setting such as zoom points at Edit profile', async ({ page }) => {
  await openSeedFarm(page)
  await openSettings(page)
  await page.getByRole('textbox', { name: 'Search settings' }).fill('zoom')
  await page.getByRole('button', { name: 'Search Stardew Valley settings' }).click()
  await expect(page.getByText('Zoom is set per profile')).toBeVisible()
  await expect(page.getByText('No settings match')).toHaveCount(0)
  await page.getByRole('button', { name: /^Edit / }).click()
  await expect(page.getByRole('tab', { name: 'Game settings' })).toHaveAttribute(
    'aria-selected',
    'true',
  )
})
