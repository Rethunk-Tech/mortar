import type { Locator, Page } from '@playwright/test'
import { openGameSelect, openProfile } from './app.ts'
import type { Case } from './render-budget.ts'

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
  /** Right-clicks the open profile in the title bar and picks an item of its menu. */
  async function profileMenu(page: Page, item: RegExp) {
    await toWorkspace(page)
    await page.getByRole('button', { name: /^Switch profile/ }).click({ button: 'right' })
    await page.getByRole('menuitem', { name: item }).click()
  }

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
    },
    {
      name: 'config-search',
      // Each keystroke re-renders the list of every mod with settings once.
      budget: { updates: 3000 },
      setup: (p) => tab(p, 'Config'),
      run: (p) => typeInto(p, /^Filter mods/, 'speed'),
    },
    {
      name: 'freeze',
      // Freezing pins every entry, so every row, panel and tab that reads the profile renders again.
      budget: { updates: 6000 },
      setup: toWorkspace,
      run: async (p) => {
        await profileMenu(p, /^Freeze this profile/)
        await tab(p, 'Mods')
        await p.getByText(/This profile is frozen/).waitFor()
      },
    },
    {
      name: 'unfreeze',
      // Lifting the holds rewrites every entry, and the toast that says so leaves once, inside the idle watch.
      budget: { updates: 5000, idleUpdates: 10, idleCommits: 1 },
      setup: (p) => tab(p, 'Mods'),
      run: (p) => p.getByRole('button', { name: /^Unfreeze/ }).click(),
    },
    {
      name: 'profile-folder',
      setup: toWorkspace,
      run: (p) => profileMenu(p, /^Profile folder…/),
    },
    {
      name: 'add-game',
      // Opens Game select first, which is its own case's cost.
      budget: { updates: 1500 },
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
      name: 'coop-list',
      setup: (p) => inGame(p, 'Guided Test'),
      run: (p) => profileMenu(p, /^Multiplayer mod list…/),
    },
    {
      name: 'modpack-form',
      setup: (p) => inGame(p, 'Guided Test'),
      run: async (p) => {
        await shareThunderstore(p)
        await p.getByRole('button', { name: /^Export as Thunderstore modpack…/ }).click()
        await p.getByRole('textbox', { name: 'Name' }).waitFor()
      },
    },
  ]
}
