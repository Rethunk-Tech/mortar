import type { Locator, Page } from '@playwright/test'
import { close, open } from './menuKit.ts'
import type { Case } from './render-budget.ts'
import type { dialogKit } from './renderDialogs.ts'
import { openGame } from './renderProfile.ts'

const GROUP = 'Render group'

/** The Mods tab's menus, on Stardew Valley: the copied data's game with mods, an update pending and a saved profile. */
export function modMenuCases(
  { ready, tab }: ReturnType<typeof dialogKit>,
  tabs: (page: Page) => Locator,
): Case[] {
  // An earlier case may have left Lethal Company open.
  const stardew = async (p: Page) => {
    await ready(p)
    await openGame(p, tabs(p), 'Stardew Valley')
  }
  let previous = 'None'
  return [
    {
      name: 'menu-selection-bar',
      setup: async (p) => {
        await stardew(p)
        await tab(p, 'Mods')
        await p.getByRole('button', { name: 'List view' }).click()
        const rows = p.locator('[data-mod-id]')
        await rows.nth(0).click()
        await rows.nth(1).click({ modifiers: ['Shift'] })
        await p.getByRole('button', { name: 'More actions' }).last().waitFor()
      },
      run: async (p) => {
        await p.getByRole('button', { name: 'More actions' }).last().click()
        await open(p)
      },
      teardown: async (p) => {
        await close(p)
        await p.getByRole('button', { name: 'Clear selection' }).click()
      },
    },
    {
      name: 'menu-update-review',
      setup: async (p) => {
        await stardew(p)
        await tab(p, 'Mods')
        await p.getByRole('button', { name: /^Review updates/ }).click()
        await p.getByRole('dialog').last().waitFor()
      },
      run: async (p) => {
        await p
          .getByRole('dialog')
          .last()
          .getByRole('button', { name: /^More actions for/ })
          .first()
          .click()
        await open(p)
      },
      teardown: async (p) => {
        await close(p)
        await p.keyboard.press('Escape')
      },
    },
    {
      name: 'menu-group',
      setup: async (p) => {
        await stardew(p)
        await tab(p, 'Mods')
        await p.getByRole('button', { name: 'List view' }).click()
        // The copied data holds no group, so the case makes one and deletes it again.
        const rows = p.locator('[data-mod-id]')
        await rows.nth(0).click()
        await rows.nth(1).click({ modifiers: ['Shift'] })
        await p.getByRole('button', { name: 'More actions' }).last().click()
        await p.getByRole('menuitem', { name: /^Add to group…/ }).click()
        const dialog = p.getByRole('dialog').last()
        await dialog.getByRole('textbox', { name: 'New group' }).fill(GROUP)
        await dialog.getByRole('button', { name: 'Add', exact: true }).click()
        await dialog.waitFor({ state: 'detached' })
        await p.getByRole('button', { name: 'Clear selection' }).click()
        // The details panel of the clicked mod would cover the header's menu button.
        await p.keyboard.press('Escape')
        await p.getByRole('button', { name: 'Group by' }).click()
        previous = (await p.getByRole('menuitemradio', { checked: true }).textContent()) ?? 'None'
        await p.getByRole('menuitemradio', { name: 'Group', exact: true }).click()
        await p.getByRole('button', { name: 'Group actions' }).first().waitFor()
      },
      run: async (p) => {
        await p.getByRole('button', { name: 'Group actions' }).first().click()
        await open(p)
      },
      teardown: async (p) => {
        await close(p)
        await p.getByRole('button', { name: 'Group actions' }).first().click()
        await p.getByRole('menuitem', { name: /^Delete group/ }).click()
        await p.getByRole('dialog').last().getByRole('button', { name: 'Delete group' }).click()
        await p.getByRole('button', { name: 'Group by' }).click()
        await p.getByRole('menuitemradio', { name: previous, exact: true }).click()
      },
    },
    {
      name: 'menu-config-presets',
      setup: async (p) => {
        await stardew(p)
        await tab(p, 'Config')
        const mods = p.getByRole('group', { name: 'Mods with settings' }).getByRole('button')
        const presets = p.getByRole('button', { name: 'Presets', exact: true })
        const tries = Math.min(await mods.count(), 12)
        for (let n = 0; n < tries && (await presets.count()) === 0; n++) {
          await mods.nth(n).click()
          await presets.waitFor({ timeout: 1500 }).catch(() => undefined)
        }
        if ((await presets.count()) === 0) {
          throw new Error('no mod among the first twelve with settings offers Presets')
        }
      },
      run: async (p) => {
        await p.getByRole('button', { name: 'Presets', exact: true }).click()
        await open(p)
      },
      teardown: close,
    },
  ]
}
