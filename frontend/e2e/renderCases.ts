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
      run: (p) => pick(p, /^Profile folder…/),
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
      name: 'coop-list',
      setup: async (p) => {
        await inGame(p, 'Guided Test')
        await openProfileMenu(p)
      },
      run: (p) => pick(p, /^Multiplayer mod list…/),
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
