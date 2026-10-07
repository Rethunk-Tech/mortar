import { expect, type Page, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'
import { A, B, DOWN, installPad, LB, press, RB, RIGHT } from './pad.ts'

test.beforeEach(({ page }) => installPad(page))

const focusedName = (page: Page) =>
  page.evaluate(() => {
    const el = document.activeElement
    return el
      ? `${el.getAttribute('role') ?? el.tagName}:${el.getAttribute('aria-label') ?? el.textContent?.trim().slice(0, 40)}`
      : ''
  })

async function expectRing(page: Page) {
  const style = await page.evaluate(() => {
    const s = getComputedStyle(document.activeElement as Element)
    return `${s.outlineStyle} ${s.outlineWidth}`
  })
  expect(style, 'the focused control shows a ring').toBe('solid 2px')
}

test('a gamepad walks the tabs, Mods, Browse and a dialog with focus always shown', async ({
  page,
}) => {
  await openSeedFarm(page)
  const mods = page.getByRole('tab', { name: 'Mods' })
  await expect(mods).toHaveAttribute('aria-selected', 'true')

  await press(page, RB)
  // The sidebar's order is Home, Mods, Saves, Config, Browse, Problems, so RB from Mods lands on Saves.
  const saves = page.getByRole('tab', { name: 'Saves' })
  await expect(saves).toHaveAttribute('aria-selected', 'true')
  await expect(saves).toBeFocused()
  await expectRing(page)
  await press(page, RB)
  await press(page, RB)
  const browse = page.getByRole('tab', { name: 'Browse' })
  await expect(browse).toHaveAttribute('aria-selected', 'true')

  // Down leaves the tab row for the Browse toolbar.
  await press(page, DOWN)
  await expect(browse).not.toBeFocused()
  expect(
    await page.evaluate(() => document.querySelector('main')?.contains(document.activeElement)),
  ).toBe(true)
  await expectRing(page)

  await press(page, LB)
  await press(page, LB)
  await press(page, LB)
  await expect(mods).toHaveAttribute('aria-selected', 'true')
  await press(page, DOWN)
  const walked = new Set<string>()
  for (let i = 0; i < 4; i++) {
    walked.add(await focusedName(page))
    await press(page, DOWN)
  }
  expect(walked.size, 'down moves focus through the Mods tab').toBeGreaterThan(2)
  await expectRing(page)

  // A opens a mod's menu, the pad moves inside it, and B closes it and gives focus back.
  const more = page.getByRole('button', { name: 'More actions for Seed Alpha' })
  await more.focus()
  await press(page, A)
  const menu = page.getByRole('menu')
  await expect(menu).toBeVisible()
  const first = await focusedName(page)
  await press(page, DOWN)
  expect(await focusedName(page)).not.toBe(first)
  expect(await menu.evaluate((m) => m.contains(document.activeElement))).toBe(true)
  await press(page, B)
  await expect(menu).toBeHidden()
  await expect(more).toBeFocused()

  // In the app menu drawer the d-pad stays inside it and B closes it.
  const menuButton = page.getByRole('button', { name: 'Mortar menu' })
  await menuButton.focus()
  await press(page, A)
  const drawer = page.locator('.MuiModal-root').last()
  await expect(drawer).toBeVisible()
  for (let i = 0; i < 3; i++) {
    await press(page, DOWN)
    expect(await drawer.evaluate((d) => d.contains(document.activeElement))).toBe(true)
  }
  await press(page, RIGHT)
  expect(await drawer.evaluate((d) => d.contains(document.activeElement))).toBe(true)
  await expectRing(page)
  await press(page, B)
  await expect(drawer).toBeHidden()

  // A dialog keeps the pad inside it too.
  await page.keyboard.press('Control+j')
  const downloads = page.getByRole('dialog', { name: 'Downloads' })
  await expect(downloads).toBeVisible()
  await press(page, DOWN)
  await press(page, DOWN)
  expect(await downloads.evaluate((d) => d.contains(document.activeElement))).toBe(true)
  await press(page, B)
  await expect(downloads).toBeHidden()
})
