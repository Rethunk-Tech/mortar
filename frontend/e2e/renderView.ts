import type { Locator, Page } from '@playwright/test'
import { returnToMainGame } from './renderProfile.ts'

const ROWS_USED = 11

/**
 * The nth of the matches; in the small seeded profile, whose list is shorter than the indexes the cases click (they are
 * picked for the copied data), the last one instead. The first of an `or` is the earlier in the page, so it is the nth
 * when there is one.
 */
export function nthOrLast(all: Locator, n: number): Locator {
  const nth = all.nth(n)
  return process.env.MORTAR_RENDER_SEEDED ? nth.or(all.last()).first() : nth
}

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
    // A profile with no mods has no view switch; the caller then reopens the run's own profile.
    if (
      !(await button.waitFor({ timeout: 3000 }).then(
        () => true,
        () => false,
      ))
    ) {
      return
    }
    if ((await button.getAttribute('aria-pressed')) !== 'true') {
      await button.click()
    }
    // A group an earlier case collapsed (kept per game) hides the rows of a profile whose mods are all in it.
    if ((await page.locator('[data-mod-id]').count()) === 0) {
      for (const header of await page
        .locator('[aria-expanded="false"]:not([aria-haspopup])')
        .all()) {
        await header.click()
      }
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
