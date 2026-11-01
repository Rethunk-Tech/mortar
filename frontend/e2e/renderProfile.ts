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

/** The open game's name, read from the title bar's game switcher. */
export async function currentGame(page: Page): Promise<string> {
  const label = await page
    .getByRole('button', { name: /^Switch game: / })
    .getAttribute('aria-label')
  return (label ?? '').replace(/^Switch game: /, '')
}

/** Opens a game from the title bar's game menu, which reopens that game's last profile. */
export async function openGame(page: Page, workspaceTabs: Locator, name: string) {
  if ((await currentGame(page)) === name) {
    return
  }
  await page.getByRole('button', { name: /^Switch game: / }).click()
  await page.getByRole('menuitem', { name: new RegExp(`^${name}`) }).click()
  await workspaceTabs.waitFor({ timeout: 30_000 })
}

let mainGame = ''

/** Notes the game the run landed in, which holds the profile with the most mods. */
export async function rememberMainGame(page: Page) {
  mainGame = await currentGame(page)
}

/** Reopens the run's game when `rows` finds none in the open profile; false when nothing changed. */
export async function returnToMainGame(
  page: Page,
  workspaceTabs: Locator,
  rows: () => Promise<number>,
) {
  if (mainGame === '' || (await currentGame(page)) === mainGame || (await rows()) > 0) {
    return false
  }
  await openGame(page, workspaceTabs, mainGame)
  return true
}

/**
 * Visits a game whose share menu offers Thunderstore when the open game's lacks it. `from` hands the run's game back,
 * so the cases after keep the profile the run measures.
 */
export function thunderstoreVisit(
  page: {
    share: (p: Page) => Promise<void>
    close: (p: Page) => Promise<void>
    home: (p: Page) => Promise<void>
  },
  workspaceTabs: (p: Page) => Locator,
) {
  let left = ''
  return {
    to: async (p: Page) => {
      await page.home(p)
      await page.share(p)
      // The menu mounts after the click, and a close that looks too early finds nothing to close.
      await p.getByRole('dialog').last().waitFor()
      const offered = await p
        .getByRole('button', { name: /^Thunderstore/ })
        .isVisible({ timeout: 1500 })
        .catch(() => false)
      await page.close(p)
      if (!offered) {
        left = await currentGame(p)
        await openGame(p, workspaceTabs(p), 'Lethal Company')
        // Opening a game can raise a dialog of its own over the workspace.
        await page.close(p)
        await page.home(p)
      }
    },
    from: async (p: Page) => {
      if (left !== '') {
        await page.close(p)
        await openGame(p, workspaceTabs(p), left)
        left = ''
      }
    },
  }
}
