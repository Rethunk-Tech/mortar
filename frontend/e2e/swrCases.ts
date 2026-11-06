import type { Page } from '@playwright/test'
import { openSettings } from './app.ts'
import type { Case } from './render-budget.ts'
import { openGame } from './renderProfile.ts'

const READ_MS = 60_000
const SETTLE_MS = 1500

/**
 * Records every element matching `selector` that appears from now on, so a placeholder that shows for a moment is
 * caught even when it is gone again by the time the case looks.
 */
async function watchFor(p: Page, selector: string) {
  await p.evaluate((css) => {
    const seen: string[] = []
    const scan = () => {
      for (const el of document.querySelectorAll(css)) {
        seen.push(el.getAttribute('aria-label') ?? el.tagName)
      }
    }
    const observer = new MutationObserver(scan)
    observer.observe(document.body, { childList: true, subtree: true, attributes: true })
    Reflect.set(globalThis, '__swrSeen', seen)
    Reflect.set(globalThis, '__swrStop', () => observer.disconnect())
    scan()
  }, selector)
}

/** Fails when a placeholder watchFor was set on showed up. */
async function expectNoneSeen(p: Page, what: string) {
  const seen = await p.evaluate(() => {
    const stop = Reflect.get(globalThis, '__swrStop')
    if (typeof stop === 'function') {
      stop()
    }
    const found = Reflect.get(globalThis, '__swrSeen')
    return Array.isArray(found) ? found.map(String) : []
  })
  if (seen.length > 0) {
    throw new Error(`${what} showed a loading placeholder (${[...new Set(seen)].join(', ')})`)
  }
}

const SKELETON = '[role="status"][aria-label^="Reading"], [role="status"][aria-label^="Checking"]'

/** Opens another profile of the open game from the title bar's switcher; returns the one that was open. */
async function switchAway(p: Page): Promise<string | null> {
  await p.getByRole('button', { name: /^Switch profile/ }).click()
  const rows = p.getByRole('menuitemradio')
  await rows.first().waitFor()
  const names = await rows.evaluateAll((els) =>
    els.map((el) => ({
      name: el.textContent ?? '',
      on: el.getAttribute('aria-checked') === 'true',
    })),
  )
  const here = names.find((n) => n.on)
  const other = names.findIndex((n) => !n.on)
  if (!here || other < 0) {
    await p.keyboard.press('Escape')
    await rows.first().waitFor({ state: 'detached' })
    return null
  }
  await rows.nth(other).click()
  return here.name
}

async function switchTo(p: Page, name: string) {
  await p.getByRole('button', { name: /^Switch profile/ }).click()
  await p.getByRole('menuitemradio', { name }).click()
}

/**
 * Stale-while-refresh: a view that was open before draws its last answer at once. Each case opens a view, leaves it,
 * and comes back while a watcher records any loading placeholder.
 */
export function swrCases(
  ready: (p: Page) => Promise<void>,
  tabs: (p: Page) => ReturnType<Page['locator']>,
  tab: (p: Page, name: string) => Promise<void>,
): Case[] {
  // Each case starts from the workspace with nothing open, whatever the case before it left.
  const open = async (p: Page, game: string) => {
    await ready(p)
    await openGame(p, tabs(p), game)
  }
  const stardew = (p: Page) => open(p, 'Stardew Valley')
  // The list view has a switch on every row; the grid has none.
  const toggleNinth = async (p: Page) => {
    await tab(p, 'Mods')
    await p.getByRole('switch').nth(8).click()
  }
  return [
    {
      name: 'swr-problems-switch-profile-and-back',
      // The profile switch re-renders both profiles' sidebars and lists.
      budget: { updates: 4000, idleUpdates: 20, idleCommits: 5 },
      setup: async (p) => {
        // The copied data's Lethal Company has two profiles to switch between; its Stardew Valley has one.
        await open(p, 'Lethal Company')
        await tab(p, 'Problems')
        await p
          .getByRole('status', { name: 'Checking the mods for problems…' })
          .waitFor({ state: 'detached', timeout: READ_MS })
      },
      run: async (p) => {
        const here = await switchAway(p)
        if (here === null) {
          return
        }
        await p.waitForTimeout(SETTLE_MS)
        await switchTo(p, here)
        await watchFor(p, SKELETON)
        await p.waitForTimeout(SETTLE_MS)
        await expectNoneSeen(p, 'Problems after switching back')
      },
    },
    {
      name: 'swr-load-order-toggle-a-mod',
      // The toggle changes the profile's mod list, which each visible row's dependency chips and name still read whole.
      budget: { updates: 1500, idleUpdates: 20, idleCommits: 2 },
      setup: async (p) => {
        await stardew(p)
        await tab(p, 'Mods')
        await p.getByRole('button', { name: 'List view' }).click()
        await tab(p, 'Load order')
        await p
          .getByRole('status', { name: 'Reading load order…' })
          .waitFor({ state: 'detached', timeout: READ_MS })
      },
      run: async (p) => {
        await toggleNinth(p)
        await watchFor(p, SKELETON)
        await tab(p, 'Load order')
        await p.waitForTimeout(SETTLE_MS)
        await expectNoneSeen(p, 'Load order after a mod was toggled')
      },
      teardown: async (p) => {
        await toggleNinth(p)
      },
    },
    {
      name: 'swr-downloads-dialog-reopen',
      setup: async (p) => {
        await stardew(p)
        await tab(p, 'Mods')
        await p.getByRole('button', { name: 'More ways to add mods' }).click()
        await p.getByRole('menuitem', { name: 'The downloads folder…' }).click()
        const dialog = p.getByRole('dialog', { name: 'From the downloads folder' })
        await dialog.getByRole('status', { name: /^Looking in the downloads folder/ }).waitFor({
          state: 'detached',
          timeout: READ_MS,
        })
        await dialog.getByRole('button', { name: 'Cancel' }).click()
        await dialog.waitFor({ state: 'detached' })
      },
      run: async (p) => {
        await watchFor(p, SKELETON)
        await p.getByRole('button', { name: 'More ways to add mods' }).click()
        await p.getByRole('menuitem', { name: 'The downloads folder…' }).click()
        const dialog = p.getByRole('dialog', { name: 'From the downloads folder' })
        await dialog.waitFor()
        await p.waitForTimeout(SETTLE_MS)
        await expectNoneSeen(p, 'The downloads dialog on its second open')
      },
      teardown: async (p) => {
        await p.keyboard.press('Escape')
      },
    },
    {
      name: 'swr-data-usage-leave-and-return',
      setup: async (p) => {
        await ready(p)
        await openSettings(p)
        const nav = p.getByRole('navigation', { name: 'Settings sections' })
        await nav.getByRole('button', { name: 'Storage', exact: true }).click()
        await p
          .getByText(/ used$/)
          .first()
          .waitFor({ timeout: READ_MS })
        await nav.getByRole('button', { name: 'General', exact: true }).click()
      },
      run: async (p) => {
        await watchFor(p, '[role="progressbar"]')
        await p
          .getByRole('navigation', { name: 'Settings sections' })
          .getByRole('button', { name: 'Storage', exact: true })
          .click()
        await p
          .getByText(/ used$/)
          .first()
          .waitFor()
        await p.waitForTimeout(SETTLE_MS)
        await expectNoneSeen(p, 'Storage after returning to it')
      },
    },
  ]
}
