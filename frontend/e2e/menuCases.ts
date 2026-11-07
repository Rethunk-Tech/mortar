import type { Locator, Page } from '@playwright/test'
import { codeSymbolCase } from './codeSymbolCase.ts'
import { close, open } from './menuKit.ts'
import { modMenuCases } from './modMenuCases.ts'
import type { Case } from './render-budget.ts'
import type { dialogKit } from './renderDialogs.ts'
import { returnToMainGame } from './renderProfile.ts'

/** Menus no screen case opens, pictured open so their grouping and rules can be reviewed. */
export function menuCases(
  kit: ReturnType<typeof dialogKit>,
  tabs: (page: Page) => Locator,
): Case[] {
  const { ready, tab, developTab, profileMenu } = kit
  let previous = 'None'
  return [
    {
      name: 'menu-play-options',
      setup: async (p) => {
        await ready(p)
        await tab(p, 'Home')
      },
      run: async (p) => {
        await p.getByRole('button', { name: 'More play options' }).click()
        await open(p)
      },
      teardown: close,
    },
    {
      name: 'menu-develop-project',
      setup: developTab,
      run: async (p) => {
        await p.getByRole('main').getByRole('button', { name: 'More actions' }).first().click()
        await open(p)
      },
      teardown: close,
    },
    {
      name: 'menu-group-by-category',
      setup: async (p) => {
        await ready(p)
        await tab(p, 'Mods')
        // The develop menus before it leave the game whose development profile holds no mods to group.
        if (await returnToMainGame(p, tabs(p), () => p.locator('[data-mod-id]').count())) {
          await tab(p, 'Mods')
        }
        await p.getByRole('button', { name: 'Group by' }).click()
        // The grouping is a stored setting, so the case puts back the one it found.
        previous = (await p.getByRole('menuitemradio', { checked: true }).textContent()) ?? 'None'
        await p.getByRole('menuitemradio', { name: 'Category', exact: true }).click()
      },
      run: async (p) => {
        await p.getByRole('button', { name: 'Group by' }).click()
        await open(p)
      },
      teardown: async (p) => {
        await close(p)
        await p.getByRole('button', { name: 'Group by' }).click()
        await p.getByRole('menuitemradio', { name: previous, exact: true }).click()
      },
    },
    {
      name: 'menu-game-switcher',
      setup: ready,
      run: async (p) => {
        await p.getByRole('button', { name: /^Switch game/ }).click()
        await open(p)
      },
      teardown: close,
    },
    {
      name: 'menu-develop-test-copies',
      setup: developTab,
      run: async (p) => {
        await p.getByRole('button', { name: 'How many copies to test with' }).first().click()
        await open(p)
      },
      teardown: close,
    },
    {
      name: 'menu-compare-picker',
      setup: async (p) => {
        await ready(p)
        await profileMenu(p, /^Compare…/)
        await p.getByRole('dialog').last().waitFor()
      },
      run: async (p) => {
        await p.getByRole('dialog').last().locator('button[aria-haspopup="menu"]').first().click()
        await open(p)
      },
      teardown: async (p) => {
        await close(p)
        await close(p)
      },
    },
    ...modMenuCases(kit, tabs),
    codeSymbolCase(kit),
  ]
}
