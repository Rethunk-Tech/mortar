import type { Page } from '@playwright/test'
import { openGameSelect, openSettings } from './app.ts'
import type { Case } from './render-budget.ts'
import type { Helpers } from './renderCases.ts'

const CLOSE_TRIES = 4
const CLOSE_WAIT_MS = 1000
const WIZARD_BUDGET = { updates: 3000 }
const TYPING_BUDGET = { updates: 2000 }

/**
 * Every dialog a profile can reach, and every tab, step and view inside one, so the report pictures each for review.
 * Each case is named dialog-…, so MORTAR_RENDER_ONLY=dialog runs only these.
 */
function dialogKit(h: Helpers) {
  const { toWorkspace, tab } = h

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
  // Cases that start from Home go there in their setup: switching tabs is the screens' cost, not the dialog's.
  const atHome = (page: Page) => tab(page, 'Home')
  const share = async (page: Page, destination?: RegExp) => {
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

  // New mod's steps after the kind (a code mod): a valid name and description open the author's fields, those the
  // features, and each Next from there the next step.
  // Everything before the Next that opens step `steps` (0: the mod step, from the kind): the case measures only
  // stepping into it.
  const newModTo = async (p: Page, steps: number) => {
    await tab(p, 'Develop')
    await button(p, /^New mod…/)
    if (steps === 0) {
      return
    }
    await inDialog(p, /^Next$/)
    const box = (name: string) =>
      p.getByRole('dialog').last().getByRole('textbox', { name }).first()
    await box('Mod name').fill('RenderCheck')
    await box('Description').fill('Checks how the wizard looks.')
    if (steps > 1) {
      await inDialog(p, /^Next$/)
      await box('Author').fill('Render Check')
      await box('Thunderstore team').fill('RenderCheck')
      await box('Id prefix').fill('com.rendercheck')
    }
    for (let at = 2; at < steps; at++) {
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
    newModTo,
    atHome,
    tab,
  }
}

// Share's destinations, each opened from Home.
function shareCases(kit: ReturnType<typeof dialogKit>): Case[] {
  const { dialog, share, shareNearby, inDialog, atHome } = kit
  return [
    dialog('share', (p) => share(p), undefined, atHome),
    dialog('share-mortar-link', (p) => share(p, /^Mortar/), undefined, atHome),
    dialog(
      'share-mortar-file',
      async (p) => {
        await share(p, /^Mortar/)
        await inDialog(p, /^File$/)
      },
      undefined,
      atHome,
    ),
    dialog('share-nexus', (p) => share(p, /^Nexus/), undefined, atHome),
    dialog('share-thunderstore', (p) => share(p, /^Thunderstore/), undefined, atHome),
    // It looks again every 5 s, so one look can land in the watch, and it may find a computer the first did not.
    dialog('share-nearby', shareNearby, { idleCalls: 4, idleUpdates: 100, idleCommits: 3 }, atHome),
    dialog('share-mod-list', (p) => share(p, /^Mod list/), undefined, atHome),
    ...['Plain text', 'Discord', 'Thunderstore'].map((format) =>
      dialog(
        `share-mod-list-${format.toLowerCase().replace(' ', '-')}`,
        async (p) => {
          await share(p, /^Mod list/)
          await inDialog(p, new RegExp(`^${format}`))
        },
        undefined,
        atHome,
      ),
    ),
  ]
}

// Home's Tools menu and the Develop tab's New mod wizard, each step measured on its own.
function toolAndWizardCases(kit: ReturnType<typeof dialogKit>): Case[] {
  const { dialog, button, pick, inDialog, newModTo, atHome, tab } = kit
  return [
    dialog(
      'manage-tools',
      async (p) => {
        await button(p, /^Tools/)
        await pick(p, /^Manage tools…/)
      },
      undefined,
      atHome,
    ),
    // Manage tools is open first: the case measures the editor opening over it.
    dialog(
      'tool-editor',
      (p) => inDialog(p, /^Add tool…/),
      undefined,
      async (p) => {
        await atHome(p)
        await button(p, /^Tools/)
        await pick(p, /^Manage tools…/)
      },
    ),
    // Filling the fields and stepping through is one action, so it re-renders the wizard several times over.
    {
      name: 'develop-new-mod',
      setup: (p) => tab(p, 'Develop'),
      // The header's button; an empty Develop page offers a second one.
      run: (p) => p.getByRole('button', { name: 'New mod…' }).first().click(),
    },
    {
      name: 'develop-game-code',
      // Game code… is in the header only while the open profile has a project; an earlier case may have opened one
      // without, so this opens the first profile the Develop tab lists a project in.
      setup: async (p) => {
        await tab(p, 'Develop')
        const game = p.getByRole('button', { name: 'Game code…' })
        if (!(await game.isVisible())) {
          await p.getByRole('button', { name: 'Open profile' }).first().click()
          await tab(p, 'Develop')
        }
      },
      run: (p) => p.getByRole('button', { name: 'Game code…' }).click(),
    },
    {
      name: 'develop-clone',
      // The add menu opens in setup, so the action is the dialog alone.
      setup: async (p) => {
        await tab(p, 'Develop')
        await p.getByRole('button', { name: 'More ways to add a mod project' }).click()
      },
      run: (p) => p.getByRole('menuitem', { name: 'Clone from GitHub…' }).click(),
    },
    {
      ...dialog(
        '',
        (p) =>
          p
            .getByRole('dialog')
            .last()
            .getByRole('button', { name: /^Suits/ })
            .click(),
        WIZARD_BUDGET,
        (p) => newModTo(p, 0),
      ),
      name: 'develop-new-mod-asset',
    },
    ...[
      ['mod', 0],
      ['author', 1],
      ['features', 2],
      ['dependencies', 3],
      ['project', 4],
    ].map(([step, n]) => ({
      ...dialog(
        '',
        (p) => inDialog(p, /^Next$/),
        WIZARD_BUDGET,
        (p) => newModTo(p, Number(n)),
      ),
      // Named after the first step's case, so the gallery shows the wizard's steps together.
      name: `develop-new-mod-${step}`,
    })),
  ]
}

export function dialogCases(h: Helpers): Case[] {
  const kit = dialogKit(h)
  const { view, row, typeInto } = h
  const { dialog, profileMenu, switcher, editProfile, importStep, pick, button, inDialog, tabIn } =
    kit
  return [
    dialog('edit-profile-appearance', editProfile),
    ...['Launch'].map((name) =>
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
    dialog('notifications', (p) => button(p, /^Notifications/)),
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
    ...shareCases(kit),
    ...toolAndWizardCases(kit),
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
      await button(p, /^Mortar menu$/)
      await pick(p, /^Report a bug…/)
    }),
    dialog('suggest-feature', async (p) => {
      await button(p, /^Mortar menu$/)
      await pick(p, /^Suggest a feature…/)
    }),
    dialog(
      'licences',
      (p) => button(p, /^View licences…/),
      undefined,
      async (p) => {
        await openSettings(p)
        await p
          .getByRole('navigation', { name: 'Settings sections' })
          .getByRole('button', { name: 'About', exact: true })
          .click()
      },
    ),
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
