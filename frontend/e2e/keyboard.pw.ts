import { expect, type Page, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

/** The focused element, or the card around it, draws a visible ring: an outline with a real width. */
async function expectFocusRing(page: Page) {
  const ringed = await page.evaluate(() => {
    for (let el = document.activeElement; el; el = el.parentElement) {
      const style = getComputedStyle(el)
      if (style.outlineStyle !== 'none' && Number.parseFloat(style.outlineWidth) > 0) {
        return el === document.activeElement || el.matches('.MuiCard-root')
      }
    }
    return false
  })
  expect(ringed, 'the focused control shows a ring').toBe(true)
}

/** Tab order follows the DOM, so a region's controls must sit left to right (title bar) or top to bottom (sidebar). */
async function expectTabOrderMatchesLayout(page: Page, region: string, axis: 'x' | 'y') {
  const positions = await page.evaluate(
    ([selector, key]) =>
      [...document.querySelectorAll(`${selector} :is(button, a[href], input, [tabindex="0"])`)]
        // The resize handle spans the sidebar's whole height, so it has no position in the order.
        .filter((el) => el.getClientRects().length > 0 && el.getAttribute('role') !== 'separator')
        .map((el) => el.getBoundingClientRect()[key === 'x' ? 'left' : 'top']),
    [region, axis] as const,
  )
  expect(positions.length).toBeGreaterThan(1)
  expect(positions, `${region} tab order`).toEqual([...positions].sort((a, b) => a - b))
}

test('the main flows work with the keyboard alone and focus stays visible and returns', async ({
  page,
}) => {
  await openSeedFarm(page)
  await expectTabOrderMatchesLayout(page, 'header', 'x')
  await expectTabOrderMatchesLayout(page, 'main nav', 'y')
  // Start from the page body so no earlier click leaves focus on a control.
  await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur())

  // The palette finishes a profile switch.
  await page.keyboard.press('Control+k')
  const palette = page.getByRole('dialog').getByRole('combobox')
  await expect(palette).toBeFocused()
  await palette.fill('Seed From Template')
  await page.keyboard.press('Enter')
  await expect(page.getByRole('dialog')).toBeHidden()
  await expect(page.getByRole('button', { name: /^Seed From Template/ }).first()).toBeVisible()

  // Tab reaches the mod cards and Enter opens the details panel.
  const first = page.getByRole('button', { name: /^Details of Seed Alpha/ })
  await expect(first).toBeVisible()
  for (let i = 0; i < 60 && !(await first.evaluate((el) => el === document.activeElement)); i++) {
    await page.keyboard.press('Tab')
  }
  await expect(first).toBeFocused()
  await expectFocusRing(page)
  // The author link is its own button after the card's title, then More actions.
  await page.keyboard.press('Tab')
  await expect(page.getByRole('button', { name: 'Self-test' }).first()).toBeFocused()
  await expectFocusRing(page)
  const more = page.getByRole('button', { name: 'More actions for Seed Alpha' })
  await page.keyboard.press('Tab')
  await expect(more).toBeFocused()
  await expectFocusRing(page)

  // A menu opens and closes from the keyboard and hands focus back to its button.
  await page.keyboard.press('Enter')
  await expect(page.getByRole('menu')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('menu')).toBeHidden()
  await expect(more).toBeFocused()

  // Enter on the card opens the details panel.
  await first.focus()
  await page.keyboard.press('Enter')
  const details = page.getByRole('complementary', { name: 'Selected mod' })
  await expect(details).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(details).toBeHidden()
  await expect(first).toBeFocused()

  // Panels opened by shortcut close with Escape and focus returns to where it was.
  await first.focus()
  for (const [keys, name] of [
    ['Control+j', 'Downloads'],
    ['Control+Shift+n', 'Notification history'],
  ] as const) {
    await page.keyboard.press(keys)
    const dialog = page.getByRole('dialog', { name })
    await expect(dialog).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(dialog).toBeHidden()
    await expect(first).toBeFocused()
  }

  // The title button's overlay still makes the whole card, tile included, one click target.
  const card = page.locator('[data-mod-id]').first()
  const box = await card.boundingBox()
  expect(box).not.toBeNull()
  await page.mouse.click(box?.x ?? 0, (box?.y ?? 0) + (box?.height ?? 0) / 2)
})
