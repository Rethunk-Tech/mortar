import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import type { Locator, Page } from '@playwright/test'
import { escapeRegExp } from './app.ts'

const CLOSE_TRIES = 4
const CLOSE_WAIT_MS = 1000

// An open select or menu hides the dialogs under it from the accessibility tree, so it counts as open too.
const overlays = (page: Page) =>
  page.getByRole('dialog').or(page.getByRole('listbox')).or(page.getByRole('menu'))
// A form dialog with edits answers Escape with "Discard changes?", which Escape only dismisses again.
export async function closeAll(page: Page) {
  const discard = page.getByRole('dialog', { name: 'Discard changes?' }).getByRole('button', {
    name: 'Discard',
  })
  for (let i = 0; i < CLOSE_TRIES && (await overlays(page).count()) > 0; i++) {
    // `last()` would name the dialog under a stacked one once the top closes, so hold the top element itself.
    const top = await overlays(page)
      .last()
      .elementHandle({ timeout: CLOSE_WAIT_MS })
      .catch(() => null)
    if (!top) {
      continue
    }
    const text = await top.textContent()
    // A step change drops focus to the page until the focus trap restores it, and Escape then never reaches the dialog.
    await page
      .waitForFunction(
        (el) => {
          const modal = el instanceof Element ? (el.closest('.MuiModal-root') ?? el) : null
          return !modal?.isConnected || modal.contains(document.activeElement)
        },
        top,
        { timeout: CLOSE_WAIT_MS },
      )
      .catch(() => undefined)
    await page.keyboard.press('Escape')
    // A wizard dialog answers Escape by stepping back (its text changes) and only closes on the next one.
    const gone = page
      .waitForFunction(
        ([el, was]) =>
          !(el instanceof HTMLElement && el.isConnected) ||
          el.getClientRects().length === 0 ||
          getComputedStyle(el).visibility === 'hidden' ||
          el.textContent !== was,
        [top, text] as const,
        { timeout: CLOSE_WAIT_MS },
      )
      .then(
        () => undefined,
        () => undefined,
      )
    await Promise.race([gone, discard.waitFor({ timeout: CLOSE_WAIT_MS }).catch(() => undefined)])
    if (await discard.isVisible()) {
      await discard.click()
    }
  }
}

/** Whether Escape could still lead to the workspace: not there yet, and a game is open to go back to. Closes dialogs first, as they hide the tabs. */
export async function backable(page: Page, workspaceTabs: Locator) {
  await closeAll(page)
  return !(
    (await workspaceTabs.isVisible()) ||
    (await page.getByRole('button', { name: 'Choose a game' }).isVisible())
  )
}

interface ProfileFacts {
  game: string
  id: string
  name: string
  mods: number
}

/** Every readable profile in the copied data, with its game's id and how many mods it holds. */
function dataProfiles(dataDir: string): ProfileFacts[] {
  const found: ProfileFacts[] = []
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
        if (p.name) {
          found.push({ game, id, name: p.name, mods })
        }
      } catch {
        // A folder without a readable profile.json is no profile.
      }
    }
  }
  return found
}

// The seeded sandbox's Lethal Company profile ties Seed Farm on mod count (it counts the bridge), so it names its own.
const mostMods = (all: ProfileFacts[]) =>
  process.env.MORTAR_RENDER_SEEDED
    ? all.find((p) => p.name === 'Seed Farm')
    : all.reduce<ProfileFacts | undefined>(
        (best, p) => (!best || p.mods > best.mods ? p : best),
        undefined,
      )

/**
 * The name of the profile with the most mods in the copied data, across every game, so the Mods cases have rows
 * whichever game the data reopens. '' when the data holds no profile.
 */
export function biggestProfile(dataDir: string): string {
  return mostMods(dataProfiles(dataDir))?.name ?? ''
}

/**
 * The data folder the run measures: the copied data, the seeded sandbox's own, or the maintainer's. A seeded run
 * must not read the maintainer's profile ids, which name no profile of its sandbox.
 */
export function renderDataDir(): string {
  const sandbox = process.env.MORTAR_E2E_DIR ?? process.env.MORTAR_SELFTEST_DIR
  return (
    process.env.MORTAR_RENDER_DATA ??
    (process.env.MORTAR_RENDER_SEEDED && sandbox
      ? join(sandbox, 'home', '.local', 'share', 'mortar')
      : join(process.env.HOME ?? '', '.local', 'share', 'mortar'))
  )
}

/** The game and profile ids of the profile with the most mods, which the run opens. */
export function biggestProfileIds(dataDir: string): { game: string; profile: string } | undefined {
  const best = mostMods(dataProfiles(dataDir))
  return best && { game: best.game, profile: best.id }
}

/** The ids of the games with a profile in the copied data other than the one the run measures, in id order. */
export function otherGameIds(dataDir: string): string[] {
  const all = dataProfiles(dataDir)
  const main = mostMods(all)?.game
  return [...new Set(all.map((p) => p.game))]
    .filter((g) => g !== main)
    .sort((a, b) => a.localeCompare(b))
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
let mainProfile = ''

/** Notes the game the run landed in and the profile it opened, which holds the most mods. */
export async function rememberMainGame(page: Page, profile: string) {
  mainGame = await currentGame(page)
  mainProfile = profile
}

/**
 * Closes what a case left open and, from the game select page, opens the run's profile, without the reload that
 * lands the run again; false when neither shows the workspace.
 */
export async function reopenMainProfile(page: Page, workspaceTabs: Locator) {
  await closeAll(page)
  if (await workspaceTabs.isVisible()) {
    return true
  }
  if (mainProfile === '') {
    return false
  }
  const card = page
    .getByRole('button', { name: new RegExp(`^(Open )?${escapeRegExp(mainProfile)}`) })
    .first()
  if (!(await card.isVisible())) {
    return false
  }
  await card.click()
  return workspaceTabs
    .waitFor({ timeout: 5000 })
    .then(() => true)
    .catch(() => false)
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
      const offered = await p.getByRole('button', { name: /^Thunderstore/ }).isVisible()
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
