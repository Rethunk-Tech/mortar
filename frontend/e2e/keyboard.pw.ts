import { expect, type Page, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

/** The focused element draws a visible ring: `:focus-visible` outlines it with a real width. */
async function expectFocusRing(page: Page) {
  const ring = await page.evaluate(() => {
    const el = document.activeElement
    const style = el ? getComputedStyle(el) : null
    return style
      ? { style: style.outlineStyle, width: Number.parseFloat(style.outlineWidth) }
      : null
  })
  expect(ring, 'something has focus').not.toBeNull()
  expect(ring?.style).not.toBe('none')
  expect(ring?.width).toBeGreaterThan(0)
}

test('the main flows work with the keyboard alone and focus stays visible and returns', async ({
  page,
}) => {
  await openSeedFarm(page)
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
  // The author link sits inside the card, so More actions is a Tab or two further on.
  const more = page.getByRole('button', { name: 'More actions for Seed Alpha' })
  for (let i = 0; i < 4 && !(await more.evaluate((el) => el === document.activeElement)); i++) {
    await page.keyboard.press('Tab')
  }
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
})
