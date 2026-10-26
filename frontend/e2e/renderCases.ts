import type { Locator, Page } from '@playwright/test'
import { openGameSelect, openProfile, openSettings } from './app.ts'
import { largeDownloadCases } from './largeDownloadCase.ts'
import { pausedDownloadCases } from './pausedDownloadCase.ts'
import type { Case } from './render-budget.ts'
import { VIEWPORT } from './renderViewport.ts'

/** What the render budget's own cases already define, which the cases of the newer screens reuse. */
export interface Helpers {
  toWorkspace: (page: Page) => Promise<void>
  tab: (page: Page, name: string) => Promise<void>
  view: (page: Page, which: 'List view' | 'Grid view') => Promise<void>
  row: (page: Page, n: number) => Locator
  typeInto: (page: Page, label: RegExp | string, text: string) => Promise<void>
  tabs: (page: Page) => Locator
  settleMs: number
}

/** Cases for the dialogs and screens added with the version picker, modpack form, profile folder, custom games, co-op
 * list, hidden mods, favourites, freezing and config search. */
export function newCases(h: Helpers): Case[] {
  const { toWorkspace, tab, view, row, typeInto, tabs } = h
  /** Right-clicks the open profile in the title bar, which opens its menu; the case then measures picking an item. */
  async function openProfileMenu(page: Page) {
    await toWorkspace(page)
    await page.getByRole('button', { name: /^Switch profile/ }).click({ button: 'right' })
  }
  const pick = (page: Page, item: RegExp) => page.getByRole('menuitem', { name: item }).click()

  async function inGame(page: Page, profile: string) {
    await toWorkspace(page)
    await openProfile(page, profile)
    await tabs(page).waitFor()
  }

  /** Select the mod at row n and open its details panel. */
  async function selectMod(page: Page, n: number) {
    await view(page, 'List view')
    await row(page, n).click()
  }

  const shareThunderstore = async (page: Page) => {
    await tab(page, 'Home')
    await page.getByRole('button', { name: /^Share…/ }).click()
    await page.getByRole('button', { name: /^Thunderstore/ }).click()
  }

  return [
    ...largeDownloadCases(),
    ...pausedDownloadCases(),
    {
      name: 'change-version',
      setup: (p) => selectMod(p, 6),
      run: (p) =>
        p
          .getByRole('button', { name: /^Change version…/ })
          .first()
          .click(),
    },
    {
      name: 'mods-favourites',
      setup: (p) => view(p, 'List view'),
      run: async (p) => {
        await p.getByRole('button', { name: /^Show/ }).first().click()
        await p.getByRole('menuitemradio', { name: /^Favourites/ }).click()
      },
      // The filter is remembered: later cases expect every mod listed.
      teardown: (p) => p.getByRole('button', { name: 'Show all mods' }).click(),
    },
    {
      name: 'config-search',
      setup: (p) => tab(p, 'Config'),
      run: (p) => typeInto(p, /^Filter mods/, 'speed'),
    },
    {
      name: 'freeze',
      // Picking from the menu closes it, which re-renders the menu's items (profile-folder shows the same ~650).
      budget: { updates: 1200 },
      setup: async (p) => {
        await tab(p, 'Mods')
        await openProfileMenu(p)
      },
      run: async (p) => {
        await pick(p, /^Freeze this profile/)
        await p.getByText(/This profile is frozen/).waitFor()
      },
    },
    {
      name: 'unfreeze',
      // The toast that says so leaves once, inside the idle watch.
      budget: { idleUpdates: 10, idleCommits: 1 },
      setup: (p) => tab(p, 'Mods'),
      run: (p) => p.getByRole('button', { name: /^Unfreeze/ }).click(),
    },
    {
      name: 'profile-folder',
      setup: openProfileMenu,
      run: (p) => pick(p, /^Move profile folder…/),
    },
    {
      name: 'add-game',
      setup: toWorkspace,
      run: async (p) => {
        await openGameSelect(p)
        await p.getByRole('button', { name: /^Add a game…/ }).click()
      },
    },
    {
      // Browse hides one result so the list has something; the section is absent while nothing is hidden.
      name: 'hidden-mods',
      setup: async (p) => {
        await tab(p, 'Browse')
        await typeInto(p, /^Search/, 'farm')
        await p.waitForTimeout(h.settleMs)
        await p.locator('[data-hit]').first().click({ button: 'right' })
        await p.getByRole('menuitem', { name: /^Hide from Browse/ }).click()
        await tab(p, 'Home')
      },
      run: async (p) => {
        await p.getByRole('button', { name: /settings$/ }).click()
        await p.getByText('Mods', { exact: true }).last().click()
        await p.getByText('Hidden mods', { exact: true }).waitFor()
      },
    },
    {
      // A page holds as many results as fit, so a large window mounts the most cards a Browse page ever does.
      name: 'browse-large-window',
      setup: async (p) => {
        await tab(p, 'Home')
        await p.setViewportSize({ width: 2560, height: 1440 })
      },
      run: (p) => tab(p, 'Browse'),
      teardown: (p) => p.setViewportSize(VIEWPORT),
    },
    {
      // No setting matches, so the search offers the open game's settings; picking it carries the text over.
      name: 'settings-search-game',
      setup: async (p) => {
        await toWorkspace(p)
        await openSettings(p)
        await typeInto(p, /^Search/, 'zzqx')
      },
      run: async (p) => {
        await p.getByRole('button', { name: /^Search .* settings$/ }).click()
        await p
          .getByRole('textbox', { name: /^Search/ })
          .first()
          .waitFor()
      },
    },
    {
      name: 'coop-list',
      setup: async (p) => {
        await inGame(p, 'Guided Test')
        await openProfileMenu(p)
      },
      run: (p) => pick(p, /^Multiplayer mod list…/),
    },
    {
      name: 'modpack-form',
      // Home and the share dialog open in setup, so the action is the form alone.
      setup: async (p) => {
        await inGame(p, 'Guided Test')
        await shareThunderstore(p)
      },
      run: async (p) => {
        await p.getByRole('button', { name: /^Modpack/ }).click()
        await p.getByRole('textbox', { name: 'Name' }).waitFor()
      },
    },
  ]
}

/** The game settings page opened from a profile's Home, and its switch to every profile's values. */
export function gameSettingsCases(h: Helpers): Case[] {
  const { toWorkspace, tab } = h
  return [
    {
      // Home opens the game's settings on the open profile's own values.
      name: 'game-settings-profile',
      setup: async (p) => {
        await toWorkspace(p)
        await tab(p, 'Home')
      },
      run: async (p) => {
        await p
          .getByRole('button', { name: / settings$/ })
          .last()
          .click()
        await p.getByRole('button', { name: /^This profile: / }).waitFor()
      },
    },
    {
      // The switch to every profile's values redraws each row from the profile's to the game's.
      name: 'game-settings-all-profiles',
      setup: async (p) => {
        await toWorkspace(p)
        await tab(p, 'Home')
        await p
          .getByRole('button', { name: / settings$/ })
          .last()
          .click()
        await p.getByRole('button', { name: /^This profile: / }).waitFor()
      },
      run: (p) => p.getByRole('button', { name: /^All .* profiles$/ }).click(),
    },
  ]
}
