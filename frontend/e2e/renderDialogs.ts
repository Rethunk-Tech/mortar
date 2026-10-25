import type { Page } from '@playwright/test'
import { openGameSelect, openProfile, openSettings } from './app.ts'
import type { Case } from './render-budget.ts'
import type { Helpers } from './renderCases.ts'

const PROFILE = 'Guided Test'
const CLOSE_TRIES = 4
const CLOSE_WAIT_MS = 1000
const WIZARD_BUDGET = { updates: 3000 }
const TYPING_BUDGET = { updates: 2000 }

/**
 * Every dialog a profile can reach, and every tab, step and view inside one, so the report pictures each for review.
 * Each case is named dialog-…, so MORTAR_RENDER_ONLY=dialog runs only these.
 */
function dialogKit(h: Helpers) {
  const { toWorkspace, tab, tabs } = h

  async function closeAll(page: Page) {
    for (let i = 0; i < CLOSE_TRIES && (await page.getByRole('dialog').count()) > 0; i++) {
      await page.keyboard.press('Escape')
      await page
        .getByRole('dialog')
        .last()
        .waitFor({ state: 'detached', timeout: CLOSE_WAIT_MS })
        .catch(() => undefined)
    }
  }
  // The report pictures each case after it ends, so a case closes the dialog the one before it left open.
  async function ready(page: Page) {
    await closeAll(page)
    await toWorkspace(page)
    const current = page.getByRole('button', { name: /^Switch profile/ })
    if (!((await current.textContent()) ?? '').includes(PROFILE)) {
      await openProfile(page, PROFILE)
      await tabs(page).waitFor()
    }
  }
  const shown = (page: Page) => page.getByRole('dialog').last().waitFor()
  const pick = (page: Page, item: RegExp) => page.getByRole('menuitem', { name: item }).click()
  const button = (page: Page, name: RegExp) => page.getByRole('button', { name }).last().click()
  const inDialog = (page: Page, name: RegExp) =>
    page.getByRole('dialog').last().getByRole('button', { name }).first().click()
  const tabIn = (page: Page, name: string) =>
    page.getByRole('dialog').last().getByRole('tab', { name, exact: true }).click()

  /** A case that opens a dialog (and moves to one of its views), pictured once it shows. */
  // prep runs in the unmeasured setup, so a case can measure one step inside a dialog (a tab) and not its opening.
  const dialog = (
    name: string,
    steps: (page: Page) => Promise<void>,
    budget?: Case['budget'],
    prep?: (page: Page) => Promise<void>,
  ): Case => ({
    name: `dialog-${name}`,
    setup: async (p) => {
      await ready(p)
      await prep?.(p)
    },
    run: async (p) => {
      await steps(p)
      await shown(p)
    },
    ...(budget ? { budget } : {}),
  })

  const profileMenu = async (page: Page, item: RegExp) => {
    await page.getByRole('button', { name: /^Switch profile/ }).click({ button: 'right' })
    await pick(page, item)
  }
  const switcher = async (page: Page, item: RegExp) => {
    await page.getByRole('button', { name: /^Switch profile/ }).click()
    await pick(page, item)
  }
  const share = async (page: Page, destination?: RegExp) => {
    await tab(page, 'Home')
    await button(page, /^Share…/)
    if (destination) {
      await inDialog(page, destination)
    }
  }
  const editProfile = (page: Page) => profileMenu(page, /^Edit profile…/)
  const importStep = async (page: Page, step: RegExp) => {
    await switcher(page, /^Import…/)
    await inDialog(page, step)
  }

  // New mod's later steps: a valid name and description open the second, the author's fields the third (dependencies),
  // and Next from there the project step.
  const newModStep = async (p: Page, steps: number) => {
    await tab(p, 'Develop')
    await button(p, /^New mod…/)
    const box = (name: string) =>
      p.getByRole('dialog').last().getByRole('textbox', { name }).first()
    await box('Mod name').fill('RenderCheck')
    await box('Description').fill('Checks how the wizard looks.')
    await inDialog(p, /^Next$/)
    if (steps > 1) {
      await box('Author').fill('Render Check')
      await box('Thunderstore team').fill('RenderCheck')
      await box('Id prefix').fill('com.rendercheck')
      await inDialog(p, /^Next$/)
    }
    if (steps > 2) {
      await inDialog(p, /^Next$/)
    }
  }
  // The first look for computers takes about 2 s; the action ends when it does, so the idle watch is idle.
  const shareNearby = async (p: Page) => {
    await share(p, /^Nearby computer/)
    await p.getByRole('dialog').getByRole('button', { name: 'Refresh', disabled: false }).waitFor()
  }
  return {
    dialog,
    shareNearby,
    profileMenu,
    switcher,
    share,
    editProfile,
    importStep,
    pick,
    button,
    inDialog,
    tabIn,
    newModStep,
  }
}

export function dialogCases(h: Helpers): Case[] {
  const { tab, view, row, typeInto } = h
  const {
    dialog,
    shareNearby,
    profileMenu,
    switcher,
    share,
    editProfile,
    importStep,
    pick,
    button,
    inDialog,
    tabIn,
    newModStep,
  } = dialogKit(h)
  return [
    dialog('edit-profile-appearance', editProfile),
    ...['Launch', 'Overrides', 'Game settings'].map((name) =>
      dialog(
        `edit-profile-${name.toLowerCase().replace(' ', '-')}`,
        async (p) => {
          const t = p.getByRole('dialog').last().getByRole('tab', { name, exact: true })
          if ((await t.count()) > 0) {
            await tabIn(p, name)
          }
        },
        undefined,
        editProfile,
      ),
    ),
    dialog('history', (p) => profileMenu(p, /^History…/)),
    dialog('history-compare', async (p) => {
      await profileMenu(p, /^History…/)
      await inDialog(p, /^Compare two…/)
    }),
    dialog('compare', (p) => profileMenu(p, /^Compare…/)),
    dialog('save-template', (p) => profileMenu(p, /^Save as template…/)),
    dialog('add-saved-group', (p) => profileMenu(p, /^Add a saved group…/)),
    dialog('delete-profile', (p) => profileMenu(p, /^Delete profile…/)),
    dialog('new-profile', (p) => switcher(p, /^New profile…/)),
    dialog('import', (p) => switcher(p, /^Import…/)),
    dialog('import-link', (p) => importStep(p, /^Link or code/)),
    dialog('import-manager', (p) => importStep(p, /^Another mod manager/)),
    dialog('share', (p) => share(p)),
    dialog('share-mortar-link', (p) => share(p, /^Mortar/)),
    dialog('share-mortar-file', async (p) => {
      await share(p, /^Mortar/)
      await inDialog(p, /^File$/)
    }),
    dialog('share-nexus', (p) => share(p, /^Nexus/)),
    dialog('share-thunderstore', (p) => share(p, /^Thunderstore/)),
    // It looks again every 5 s, so one look can land in the watch beside the end of the one before.
    dialog('share-nearby', shareNearby, { idleCalls: 4 }),
    dialog('share-mod-list', (p) => share(p, /^Mod list/)),
    ...['Plain text', 'Discord', 'Thunderstore'].map((format) =>
      dialog(`share-mod-list-${format.toLowerCase().replace(' ', '-')}`, async (p) => {
        await share(p, /^Mod list/)
        await inDialog(p, new RegExp(`^${format}`))
      }),
    ),
    dialog('manage-tools', async (p) => {
      await tab(p, 'Home')
      await button(p, /^Tools/)
      await pick(p, /^Manage tools…/)
    }),
    dialog('tool-editor', async (p) => {
      await tab(p, 'Home')
      await button(p, /^Tools/)
      await pick(p, /^Manage tools…/)
      await inDialog(p, /^Add tool…/)
    }),
    // Filling the fields and stepping through is one action, so it re-renders the wizard several times over.
    dialog('new-mod-author', (p) => newModStep(p, 1), WIZARD_BUDGET),
    dialog('new-mod-dependencies', (p) => newModStep(p, 2), WIZARD_BUDGET),
    dialog('new-mod-project', (p) => newModStep(p, 3), WIZARD_BUDGET),
    // Showing the list and picking a mod is setup: only opening the dialog is measured.
    dialog(
      'mod-also-add',
      (p) =>
        p
          .getByRole('complementary', { name: 'Selected mod' })
          .getByText(/^Also add to…/)
          .click(),
      undefined,
      async (p) => {
        await view(p, 'List view')
        await row(p, 0).click()
      },
    ),
    dialog('report-bug', async (p) => {
      await button(p, /^Mortar/)
      await pick(p, /^Report a bug…/)
    }),
    dialog('suggest-feature', async (p) => {
      await button(p, /^Mortar/)
      await pick(p, /^Suggest a feature…/)
    }),
    dialog('licences', async (p) => {
      await openSettings(p)
      await p
        .getByRole('navigation', { name: 'Settings sections' })
        .getByRole('button', { name: 'About', exact: true })
        .click()
      await button(p, /^View licences…/)
    }),
    // Typing twelve characters re-renders the form once per character.
    dialog(
      'add-game-filled',
      (p) => typeInto(p, 'Name', 'Render Check'),
      TYPING_BUDGET,
      // The game library and the dialog opening are setup: only typing the name is measured.
      async (p) => {
        await openGameSelect(p)
        await button(p, /^Add a game…/)
      },
    ),
  ]
}
