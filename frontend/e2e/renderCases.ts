import type { Locator, Page } from '@playwright/test'
import { openGameSelect, openSettings, switchToOtherProfile } from './app.ts'
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
  const { toWorkspace, tab, view, typeInto } = h
  /** Right-clicks the open profile in the title bar, which opens its menu; the case then measures picking an item. */
  async function openProfileMenu(page: Page) {
    await toWorkspace(page)
    await page.getByRole('button', { name: /^Switch profile/ }).click({ button: 'right' })
  }
  const pick = (page: Page, item: RegExp) => page.getByRole('menuitem', { name: item }).click()

  // The case measures a second profile, so it skips itself when the game has only one.
  let inOther = false
  let shared = false
  let versioned = false
  async function inGame(page: Page) {
    await toWorkspace(page)
    inOther = await switchToOtherProfile(page)
  }

  /** Select the mod at row n and open its details panel. */
  const shareThunderstore = async (page: Page) => {
    await tab(page, 'Home')
    await page.getByRole('button', { name: /^Share…/ }).click()
    const target = page.getByRole('button', { name: /^Thunderstore/ })
    if (await target.isVisible({ timeout: 1500 }).catch(() => false)) {
      await target.click()
      return true
    }
    await page.keyboard.press('Escape')
    return false
  }

  return [
    ...largeDownloadCases(),
    ...pausedDownloadCases(),
    {
      name: 'change-version',
      setup: async (p) => {
        versioned = await selectVersioned(p, h)
      },
      run: async (p) => {
        if (versioned) {
          await p
            .getByRole('button', { name: /^Change version…/ })
            .first()
            .click()
        }
      },
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
    hiddenModsCase(h),
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
        await inGame(p)
        if (inOther) {
          await openProfileMenu(p)
        }
      },
      run: async (p) => {
        const item = p.getByRole('menuitem', { name: /^Multiplayer mod list…/ })
        if (inOther && (await item.count()) > 0) {
          await item.click()
        }
      },
    },
    {
      name: 'modpack-form',
      // Home and the share dialog open in setup, so the action is the form alone.
      setup: async (p) => {
        await inGame(p)
        shared = inOther && (await shareThunderstore(p))
      },
      run: async (p) => {
        if (!shared) {
          return
        }
        await p.getByRole('button', { name: /^Modpack/ }).click()
        await p.getByRole('textbox', { name: 'Name' }).waitFor()
      },
    },
  ]
}

/** Clicks the nth match, or does nothing when a copied profile has fewer: the case then measures nothing. */
export async function clickIfPresent(
  items: Locator,
  n: number,
  how: 'click' | 'dispatch' = 'click',
) {
  if ((await items.count()) > n) {
    await (how === 'click' ? items.nth(n).click() : items.nth(n).dispatchEvent('click'))
  }
}

/** Browse hides one result so the settings list has something; the section is absent while nothing is hidden. */
function hiddenModsCase({ tab, typeInto, settleMs }: Helpers): Case {
  let hidden = false
  return {
    name: 'hidden-mods',
    setup: async (p) => {
      await tab(p, 'Browse')
      await typeInto(p, /^Search/, 'farm')
      await p.waitForTimeout(settleMs)
      // Browse results come from the network, so the case skips itself when none arrived.
      hidden = (await p.locator('[data-hit]').count()) > 0
      if (hidden) {
        await p.locator('[data-hit]').first().click({ button: 'right' })
        await p.getByRole('menuitem', { name: /^Hide from Browse/ }).click()
      }
      await tab(p, 'Home')
    },
    run: async (p) => {
      if (!hidden) {
        return
      }
      await p.getByRole('button', { name: /settings$/ }).click()
      await p.getByText('Mods', { exact: true }).last().click()
      await p.getByText('Hidden mods', { exact: true }).waitFor()
    },
  }
}

// The first mod that offers other versions: a mod built in Develop or added from an archive has none.
async function selectVersioned(
  p: Page,
  { view, row }: Pick<Helpers, 'view' | 'row'>,
): Promise<boolean> {
  await view(p, 'List view')
  const rows = await p.locator('[data-mod-id]').count()
  for (let n = 0; n < rows; n++) {
    await row(p, n).click()
    if (
      await p
        .getByRole('button', { name: /^Change version…/ })
        .first()
        .isVisible()
    ) {
      return true
    }
  }
  return false
}

/** The game settings page opened from a profile's Home, and its switch to every profile's values. */
export function gameSettingsCases(h: Helpers): Case[] {
  const { toWorkspace, tab, tabs } = h
  return [
    {
      // Only a loader with a devTemplate has the Develop tab.
      name: 'develop',
      setup: (p) => tab(p, 'Home'),
      run: async (p) => {
        if (
          await tabs(p)
            .getByRole('tab', { name: /^Develop/ })
            .count()
        ) {
          await tab(p, 'Develop')
        }
      },
    },
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
      // A changed row shows the dot after its label and a reset beside its control.
      name: 'game-settings-profile-changed',
      setup: async (p) => {
        await toWorkspace(p)
        await tab(p, 'Home')
        await p
          .getByRole('button', { name: / settings$/ })
          .last()
          .click()
        await p.getByRole('button', { name: /^This profile: / }).waitFor()
      },
      run: async (p) => {
        await p.getByRole('button', { name: 'Play', exact: true }).click()
        await p.getByRole('switch', { name: 'Update mods before Play on new profiles' }).click()
        await p.getByRole('button', { name: /^Reset to the game's: / }).waitFor()
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
    {
      // Mods holds the settings every profile shares, greyed under one line on the profile's scope.
      name: 'game-settings-mods',
      setup: async (p) => {
        await toWorkspace(p)
        await tab(p, 'Home')
        await p
          .getByRole('button', { name: / settings$/ })
          .last()
          .click()
        await p.getByRole('button', { name: /^This profile: / }).waitFor()
      },
      run: async (p) => {
        await p.getByRole('button', { name: 'Mods', exact: true }).click()
        await p.getByText('Preferred source').scrollIntoViewIfNeeded()
      },
    },
  ]
}
