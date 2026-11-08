import { expect, type Page, test } from '@playwright/test'
import { Button, installPad, press } from '@rethunk/dom-input-nav/playwright'
import { openSeedFarm } from './app.ts'

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

  // RB walks the sidebar in its own order, which differs by game (a load order tab or none).
  const order = await page
    .getByRole('tablist', { name: 'Profile sections' })
    .getByRole('tab')
    .allInnerTexts()
  const names = order.map((n) => n.trim().split('\n')[0] ?? '')
  const next = page.getByRole('tab', { name: names[names.indexOf('Mods') + 1], exact: true })
  await press(page, Button.rb)
  await expect(next).toHaveAttribute('aria-selected', 'true')
  await expect(next).toBeFocused()
  await expectRing(page)
  for (let i = names.indexOf('Mods') + 1; i < names.indexOf('Browse'); i += 1) {
    await press(page, Button.rb)
  }
  const browse = page.getByRole('tab', { name: 'Browse' })
  await expect(browse).toHaveAttribute('aria-selected', 'true')

  // Down leaves the tab row for the Browse toolbar.
  await press(page, Button.down)
  await expect(browse).not.toBeFocused()
  expect(
    await page.evaluate(() => document.querySelector('main')?.contains(document.activeElement)),
  ).toBe(true)
  await expectRing(page)

  for (let i = names.indexOf('Browse'); i > names.indexOf('Mods'); i -= 1) {
    await press(page, Button.lb)
  }
  await expect(mods).toHaveAttribute('aria-selected', 'true')
  await press(page, Button.down)
  const walked = new Set<string>()
  for (let i = 0; i < 4; i++) {
    walked.add(await focusedName(page))
    await press(page, Button.down)
  }
  expect(walked.size, 'down moves focus through the Mods tab').toBeGreaterThan(2)
  await expectRing(page)

  // A opens a mod's menu, the pad moves inside it, and B closes it and gives focus back.
  const more = page.getByRole('button', { name: 'More actions for Seed Alpha' })
  await more.focus()
  await press(page, Button.a)
  const menu = page.getByRole('menu')
  await expect(menu).toBeVisible()
  const first = await focusedName(page)
  await press(page, Button.down)
  expect(await focusedName(page)).not.toBe(first)
  expect(await menu.evaluate((m) => m.contains(document.activeElement))).toBe(true)
  await press(page, Button.b)
  await expect(menu).toBeHidden()
  await expect(more).toBeFocused()

  // In the app menu drawer the d-pad stays inside it and B closes it.
  const menuButton = page.getByRole('button', { name: 'Mortar menu' })
  await menuButton.focus()
  await press(page, Button.a)
  const drawer = page.locator('.MuiModal-root:not(.MuiModal-hidden)').last()
  await expect(drawer).toBeVisible()
  for (let i = 0; i < 3; i++) {
    await press(page, Button.down)
    expect(await drawer.evaluate((d) => d.contains(document.activeElement))).toBe(true)
  }
  await press(page, Button.right)
  expect(await drawer.evaluate((d) => d.contains(document.activeElement))).toBe(true)
  await expectRing(page)
  await press(page, Button.b)
  await expect(drawer).toBeHidden()

  // A dialog keeps the pad inside it too.
  await page.keyboard.press('Control+j')
  const downloads = page.getByRole('dialog', { name: 'Downloads' })
  await expect(downloads).toBeVisible()
  await press(page, Button.down)
  await press(page, Button.down)
  expect(await downloads.evaluate((d) => d.contains(document.activeElement))).toBe(true)
  await press(page, Button.b)
  await expect(downloads).toBeHidden()
})
