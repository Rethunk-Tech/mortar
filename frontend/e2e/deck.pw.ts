import { expect, type Page, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

// The Steam Deck's screen, which Mortar fills in Game Mode.
test.use({ viewport: { width: 1280, height: 800 } })

/** Every visible element that scrolls sideways, page included, as "tag.class scrollWidth>clientWidth". */
function sidewaysScrollers(page: Page): Promise<string[]> {
  return page.evaluate(() =>
    [document.documentElement, ...document.querySelectorAll('body *')]
      .filter((el) => {
        const scrolls =
          el === document.documentElement ||
          ['auto', 'scroll'].includes(getComputedStyle(el).overflowX)
        return scrolls && el.getClientRects().length > 0 && el.scrollWidth > el.clientWidth + 1
      })
      .map(
        (el) =>
          `${el.tagName.toLowerCase()}.${[...el.classList].slice(0, 2).join('.')} ${el.scrollWidth}>${el.clientWidth}`,
      ),
  )
}

async function expectNoSidewaysScroll(page: Page, screen: string) {
  expect(await sidewaysScrollers(page), `${screen} scrolls sideways`).toEqual([])
}

test('every main screen fits 1280x800 without scrolling sideways', async ({ page }) => {
  await openSeedFarm(page)
  await expectNoSidewaysScroll(page, 'Mods')
  await page.getByRole('button', { name: /^Details of Seed Alpha/ }).click()
  await expect(page.getByRole('complementary', { name: 'Selected mod' })).toBeVisible()
  await expectNoSidewaysScroll(page, 'Mods with the details panel')
  await page.keyboard.press('Escape')
  // The list's fixed columns are the widest layout: the details panel leaves it about 700px.
  await page.getByRole('button', { name: 'List view' }).click()
  await page.locator('[data-mod-row]').first().click()
  await expect(page.getByRole('complementary', { name: 'Selected mod' })).toBeVisible()
  await expectNoSidewaysScroll(page, 'Mods list with the details panel')
  await page.keyboard.press('Escape')
  await page.getByRole('button', { name: 'Grid view' }).click()
  for (const tab of ['Browse', 'Problems']) {
    await page.getByRole('tab', { name: tab }).click()
    await expectNoSidewaysScroll(page, tab)
  }
  await page.getByRole('button', { name: 'Mortar menu' }).click()
  await page.getByRole('button', { name: 'Settings', exact: true }).click()
  await expect(page.getByRole('navigation').first()).toBeVisible()
  await expectNoSidewaysScroll(page, 'Settings')
  await page.keyboard.press('Escape')
  await page.getByRole('button', { name: 'Game select' }).click()
  await expect(page.locator('[data-tile]').first()).toBeVisible()
  await expectNoSidewaysScroll(page, 'Game select')
})

test('the arrow keys walk the Game select tiles', async ({ page }) => {
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'Game select' }).click()
  const tiles = page.locator('[data-tile]')
  await expect(tiles).toHaveCount(2)
  await page.getByRole('button', { name: 'Open Stardew Valley' }).focus()
  await page.keyboard.press('ArrowRight')
  expect(await tiles.nth(1).evaluate((t) => t.contains(document.activeElement))).toBe(true)
  await page.keyboard.press('ArrowLeft')
  await expect(page.getByRole('button', { name: 'Open Stardew Valley' })).toBeFocused()
  // The tile clips what spills over its edge, so the cover's ring is drawn inside it.
  await expect(page.getByRole('button', { name: 'Open Stardew Valley' })).toHaveCSS(
    'outline-offset',
    '-2px',
  )
})
