import type { Page } from '@playwright/test'
import { close, open as menuOpen } from './menuKit.ts'
import type { Case } from './render-budget.ts'
import type { dialogKit } from './renderDialogs.ts'

const SWITCHER = /^Switch profile/
// A profile's row in the switcher reads "Name · N mods"; the game switcher's rows read otherwise.
const PROFILE_ROW = / · \d+ mods?$/

const profileName = (row: string) => (row.split(' · ')[0] ?? '').trim()

async function openProfile(p: Page, name: string) {
  await p.getByRole('button', { name: SWITCHER }).click()
  await p
    .locator('[role^=menuitem]:visible')
    .filter({ hasText: PROFILE_ROW })
    .filter({ hasText: name })
    .first()
    .click()
}

/**
 * Game code on a Develop tab whose open profile has no mod project (a player who has not made one, or on a machine
 * that cannot build one): the button is in the page actions and the dialog lists the game's assemblies.
 */
export function noProjectCodeCases({ developTab }: ReturnType<typeof dialogKit>): Case[] {
  let development = ''
  const noProjectProfile = async (p: Page) => {
    await developTab(p)
    const button = p.getByRole('button', { name: SWITCHER })
    const open = (await button.textContent()) ?? ''
    await button.click()
    await menuOpen(p)
    const names = (
      await p.locator('[role^=menuitem]:visible').filter({ hasText: PROFILE_ROW }).allTextContents()
    ).map(profileName)
    await close(p)
    development = names.find((n) => open.includes(n)) ?? ''
    // The copied data's other profiles hold no project: only the development profile does.
    const other = names.find((n) => n !== development)
    if (!other) {
      throw new Error('the render data has no second profile for the game')
    }
    await openProfile(p, other)
    await p.getByRole('tablist', { name: 'Profile sections' }).waitFor()
  }
  const back = async (p: Page) => {
    await close(p).catch(() => undefined)
    await openProfile(p, development)
  }
  return [
    {
      name: 'develop-no-projects',
      setup: noProjectProfile,
      run: async (p) => {
        await p.getByRole('button', { name: /^Game code…/ }).waitFor()
        if ((await p.getByRole('main').getByRole('button', { name: 'More actions' }).count()) > 0) {
          throw new Error('the open profile still lists a mod project')
        }
      },
      teardown: back,
    },
    {
      name: 'develop-game-code-no-project',
      setup: noProjectProfile,
      run: async (p) => {
        await p.getByRole('button', { name: /^Game code…/ }).click()
        const dialog = p.getByRole('dialog').last()
        await dialog.getByRole('combobox', { name: /^Search the game's code/ }).waitFor()
        await dialog.getByText('Assembly-CSharp.dll').first().waitFor()
      },
      teardown: async (p) => {
        await p.keyboard.press('Escape')
        await back(p)
      },
    },
  ]
}
