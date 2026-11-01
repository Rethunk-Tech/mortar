import type { Locator, Page } from '@playwright/test'
import { returnToMainGame } from './renderProfile.ts'

const ROWS_USED = 11

/**
 * Shows the Mods tab in a view with nothing filtered or selected. Mods cases click rows up to the tenth, so a profile
 * another case opened with fewer gives way to the run's own.
 */
export function modsView(
  tab: (page: Page, name: string) => Promise<void>,
  tabs: (page: Page) => Locator,
  row: (page: Page, n: number) => Locator,
) {
  async function show(page: Page, which: 'List view' | 'Grid view') {
    await tab(page, 'Mods')
    // A filter typed by an earlier case persists per profile and would leave a grid with fewer cards than a case clicks.
    const filter = page.getByRole('textbox', { name: /^Filter/ }).first()
    if ((await filter.isVisible()) && (await filter.inputValue()) !== '') {
      await filter.fill('')
    }
    const button = page.getByRole('button', { name: which })
    if ((await button.getAttribute('aria-pressed')) !== 'true') {
      await button.click()
    }
    // Closes the details panel, so a case that opens it starts from the same place every time.
    if (await page.getByRole('complementary', { name: 'Selected mod' }).isVisible()) {
      await page.keyboard.press('Escape')
    }
  }
  return async (page: Page, which: 'List view' | 'Grid view') => {
    await show(page, which)
    if (await returnToMainGame(page, tabs(page), () => row(page, ROWS_USED - 1).count())) {
      await show(page, which)
    }
  }
}
