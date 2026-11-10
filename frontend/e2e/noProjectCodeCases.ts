import type { Page } from '@playwright/test'
import { close } from './menuKit.ts'
import type { Case } from './render-budget.ts'
import type { dialogKit } from './renderDialogs.ts'

const SWITCHER = /^Switch profile/
const SPARE = 'No project'
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
    // The menu's rows mount as it opens and read the profile list then, so wait for a profile row, not any row.
    const rows = p.locator('[role^=menuitem]:visible').filter({ hasText: PROFILE_ROW })
    await rows.first().waitFor()
    const names = (await rows.allTextContents()).map(profileName)
    development = names.find((n) => open.includes(n)) ?? ''
    // The copied data's other profiles hold no project: only the development profile does. Data with no other profile
    // gets an empty one, made in the sandbox copy and left there for the next case.
    const other = names.find((n) => n !== development)
    if (other) {
      await close(p)
      await openProfile(p, other)
    } else {
      await p.getByRole('menuitem', { name: /^New profile…/ }).click()
      const dialog = p.getByRole('dialog', { name: 'New profile' })
      await dialog.getByRole('textbox', { name: 'Profile name' }).fill(SPARE)
      await dialog.getByRole('button', { name: 'Create' }).click()
      await dialog.waitFor({ state: 'hidden' })
      await p.getByRole('button', { name: SWITCHER }).filter({ hasText: SPARE }).waitFor()
    }
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
