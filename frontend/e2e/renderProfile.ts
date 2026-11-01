import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import type { Locator, Page } from '@playwright/test'

/**
 * The name of the profile with the most mods in the copied data, across every game, so the Mods cases have rows
 * whichever game the data reopens. '' when the data holds no profile.
 */
export function biggestProfile(dataDir: string): string {
  let best = { name: '', mods: -1 }
  for (const game of readDirSafe(join(dataDir, 'profiles'))) {
    for (const id of readDirSafe(join(dataDir, 'profiles', game))) {
      try {
        const p = JSON.parse(
          readFileSync(join(dataDir, 'profiles', game, id, 'profile.json'), 'utf8'),
        ) as {
          name?: string
          entries?: { mods?: unknown[] }[]
        }
        const mods = (p.entries ?? []).reduce((n, e) => n + (e.mods?.length ?? 0), 0)
        if (p.name && mods > best.mods) {
          best = { name: p.name, mods }
        }
      } catch {
        // A folder without a readable profile.json is no profile.
      }
    }
  }
  return best.name
}

function readDirSafe(dir: string): string[] {
  try {
    return readdirSync(dir)
  } catch {
    return []
  }
}

/** The open game's name, read from the title bar's game switcher. */
async function currentGame(page: Page): Promise<string> {
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
  // The menu's games are radio rows, its other entries plain items.
  await page
    .getByRole('menuitemradio', { name: new RegExp(`^${name}`) })
    .or(page.getByRole('menuitem', { name: new RegExp(`^${name}`) }))
    .first()
    .click()
  // The old game's tabs are still up while the switch lands, so wait for the title bar to name the new one.
  await page.getByRole('button', { name: `Switch game: ${name}` }).waitFor({ timeout: 30_000 })
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
