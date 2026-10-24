import { expect, test } from '@playwright/test'
import { dwell, openGameSelect, openSeedFarm, openSettings } from './app.ts'

// Browse reaches the real Nexus and GitHub search endpoints from the sandbox; the suite has no offline network
// fixture, so that spec skips itself when they cannot be reached.
const SEARCH_WAIT_MS = 20_000
const REACH_MS = 3000

/** Whether the sites Browse searches answer from this machine. */
async function online(): Promise<boolean> {
  const answers = await Promise.all(
    ['https://api.github.com', 'https://api.nexusmods.com'].map((url) =>
      fetch(url, { method: 'HEAD', signal: AbortSignal.timeout(REACH_MS) }).then(
        () => true,
        () => false,
      ),
    ),
  )
  return answers.every(Boolean)
}

test('Game Select lists Stardew Valley before Lethal Company', async ({ page }) => {
  await openSeedFarm(page)
  await openGameSelect(page)
  const stardew = page.getByRole('button', { name: 'Open Stardew Valley' })
  const lethal = page.getByText('Lethal Company', { exact: true })
  await expect(lethal).toBeVisible()
  await expect(stardew).toBeVisible()
  const [s, l] = await Promise.all([stardew.boundingBox(), lethal.boundingBox()])
  expect(s && l && (s.y < l.y || (s.y === l.y && s.x < l.x))).toBe(true)
  // The centre of the tile is its profile cards, which open their own profile.
  await stardew.click({ position: { x: 8, y: 8 } })
  await expect(page.getByRole('tab', { name: 'Mods' })).toBeVisible()
})

test('Browse defaults to All sources and a best-match search shows results from more than one source', async ({
  page,
}, testInfo) => {
  testInfo.skip(!(await online()), 'Nexus and GitHub are not reachable from here')
  await openSeedFarm(page)
  await page.getByRole('tab', { name: 'Browse' }).click()
  await expect(page.getByRole('button', { name: 'All sources' })).toHaveAttribute(
    'aria-pressed',
    'true',
  )
  // The default sort, most downloaded, leaves out GitHub, which cannot rank by downloads; best match merges every site.
  // The sort is kept in the game's settings, so it is put back for the specs after this one.
  const sort = page.getByRole('combobox', { name: 'Sort' })
  const chosen = ((await sort.textContent()) ?? '').trim()
  await sort.click()
  await page.getByRole('option', { name: 'Sort: best match' }).click()
  await page.getByLabel('Search all mod sites').fill('farm')
  // A grid card has no open-page button; each card is keyed by its source.
  await expect(page.locator('[data-hit^="nexus:"]').first()).toBeVisible({
    timeout: SEARCH_WAIT_MS,
  })
  await expect(page.locator('[data-hit^="github:"]').first()).toBeVisible({
    timeout: SEARCH_WAIT_MS,
  })
  await sort.click()
  await page.getByRole('option', { name: chosen, exact: true }).click()
})

test('the game title opens a switcher listing both playable games', async ({ page }) => {
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'Stardew Valley' }).first().click()
  const switcher = page.getByRole('menu', { name: 'Games' })
  await expect(switcher.getByRole('menuitemradio', { name: 'Lethal Company' })).toBeVisible()
  await expect(switcher.getByRole('menuitemradio', { name: 'Stardew Valley' })).toBeChecked()
  await page.keyboard.press('Escape')
  await expect(switcher).toBeHidden()
})

test('switching games asks nothing about the previous game’s profiles', async ({ page }) => {
  await openSeedFarm(page)
  const refused: string[] = []
  page.on('response', (r) => {
    if (r.url().endsWith('/wails/runtime') && r.status() >= 400) {
      refused.push(r.request().postData() ?? '')
    }
  })
  await page.getByRole('button', { name: 'Stardew Valley' }).first().click()
  await page.getByRole('menuitemradio', { name: 'Lethal Company' }).click()
  await expect(page.getByRole('button', { name: 'Seed Lobby' }).first()).toBeVisible()
  await expect(page.getByRole('tab', { name: 'Mods' })).toBeVisible()
  await dwell(page, 1000)
  expect(refused).toEqual([])
})

test('Settings › Mods and profiles has the adult-mods switch, off by default', async ({ page }) => {
  await openSeedFarm(page)
  await openSettings(page)
  await page.getByRole('navigation').getByRole('button', { name: 'Mods and profiles' }).click()
  const adult = page.getByLabel('Show adult mods in browse')
  await expect(adult).toBeVisible()
  await expect(adult).not.toBeChecked()
})
