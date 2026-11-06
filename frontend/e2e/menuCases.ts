import type { Page } from '@playwright/test'
import type { Case } from './render-budget.ts'
import type { dialogKit } from './renderDialogs.ts'

/** Menus no screen case opens, pictured open so their grouping and rules can be reviewed. */
export function menuCases({ ready, tab, developTab }: ReturnType<typeof dialogKit>): Case[] {
  const open = (p: Page) => p.getByRole('menu').getByRole('menuitem').first().waitFor()
  let previous = 'None'
  const close = (p: Page) => p.keyboard.press('Escape')
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
  ]
}
