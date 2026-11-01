import type { Locator, Page } from '@playwright/test'

// "Name · 12 mods": the label of each profile in the switcher.
const MOD_COUNT = /·\s*(\d+) mods?/u

/** Opens the open game's profile with the most mods, so the Mods cases have rows however the copied data reopens. */
export async function openBiggestProfile(page: Page, workspaceTabs: Locator) {
  await page.getByRole('button', { name: /^Switch profile/ }).click()
  const items = page.getByRole('menuitemradio')
  await items.first().waitFor()
  let best = { index: -1, mods: -1, checked: false }
  for (const [index, item] of (await items.all()).entries()) {
    const mods = Number(MOD_COUNT.exec(await item.innerText())?.[1] ?? '-1')
    if (mods > best.mods) {
      best = { index, mods, checked: (await item.getAttribute('aria-checked')) === 'true' }
    }
  }
  if (best.index < 0 || best.checked) {
    await page.keyboard.press('Escape')
    return
  }
  await items.nth(best.index).click()
  await workspaceTabs.waitFor({ timeout: 30_000 })
}
