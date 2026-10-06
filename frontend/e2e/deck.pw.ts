import { expect, type Page, test } from '@playwright/test'
import { ENABLED_GAMES, openSeedFarm } from './app.ts'
import { DOWN, installPad, press } from './pad.ts'

// The Steam Deck's screen, which Mortar fills in Game Mode.
test.use({ viewport: { width: 1280, height: 800 } })

/** Every visible element that scrolls sideways, page included, as "tag.class scrollWidth>clientWidth". */
function sidewaysScrollers(page: Page): Promise<string[]> {
  return page.evaluate(async () => {
    // Measured in a painted frame: the list fits its columns when a ResizeObserver reports its new width, which
    // happens in the frame after a details panel opens, while a measurement in between sees the old column set.
    await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))
    return [document.documentElement, ...document.querySelectorAll('body *')]
      .filter((el) => {
        const scrolls =
          el === document.documentElement ||
          ['auto', 'scroll'].includes(getComputedStyle(el).overflowX)
        return scrolls && el.getClientRects().length > 0 && el.scrollWidth > el.clientWidth + 1
      })
      .map(
        (el) =>
          `${el.tagName.toLowerCase()}.${[...el.classList].slice(0, 2).join('.')} ${el.scrollWidth}>${el.clientWidth}`,
      )
  })
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
  await expect(tiles).toHaveCount(ENABLED_GAMES)
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

/** Every visible control smaller than 44x44, as "role name WxH"; a link in a line of text is exempt (WCAG 2.5.8). */
function smallTargets(page: Page): Promise<string[]> {
  return page.evaluate(() =>
    [
      ...document.querySelectorAll<HTMLElement>(
        'button, a[href], input:not([type="hidden"]), select, textarea, [role="button"], [role="tab"], [role="menuitem"], [role="option"], [role="checkbox"], [role="switch"], [role="radio"], [role="slider"], [tabindex="0"]',
      ),
    ]
      .filter((el) => el.getClientRects().length > 0 && el.closest('[aria-hidden="true"]') === null)
      .filter((el) => getComputedStyle(el).display !== 'inline' && !el.matches('.MuiLink-root'))
      // A resize handle's touch strip is its ::before.
      .filter((el) => Number.parseFloat(getComputedStyle(el, '::before').width) < 44)
      // A checkbox, radio, switch or text input sits inside its control, which is the target.
      .map(
        (el) =>
          el.closest<HTMLElement>('.MuiSwitch-root, .MuiButtonBase-root, .MuiInputBase-root') ?? el,
      )
      .filter((el, i, all) => all.indexOf(el) === i)
      .map((el) => ({ el, r: el.getBoundingClientRect() }))
      .filter(({ r }) => r.width < 44 || r.height < 44)
      .map(
        ({ el, r }) =>
          `${el.getAttribute('role') ?? el.tagName.toLowerCase()} "${(el.getAttribute('aria-label') ?? el.textContent ?? '').trim().slice(0, 30)}" ${Math.round(r.width)}x${Math.round(r.height)}`,
      ),
  )
}

test('with a gamepad leading, Mods, Browse and a dialog have 44px targets', async ({ page }) => {
  await installPad(page)
  await openSeedFarm(page)
  await press(page, DOWN)
  expect(await smallTargets(page), 'Mods targets under 44px').toEqual([])
  await expectNoSidewaysScroll(page, 'Mods')
  await page.getByRole('tab', { name: 'Browse' }).click()
  expect(await smallTargets(page), 'Browse targets under 44px').toEqual([])
  await expectNoSidewaysScroll(page, 'Browse')
  // Browse focuses its search field, where shortcuts stay silent.
  await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur())
  await page.keyboard.press('Control+j')
  await expect(page.getByRole('dialog', { name: 'Downloads' })).toBeVisible()
  expect(await smallTargets(page), 'Downloads targets under 44px').toEqual([])
})
