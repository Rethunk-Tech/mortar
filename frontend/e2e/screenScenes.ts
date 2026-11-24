import type { Page } from '@playwright/test'
import { loadOrderStub, releaseEverywhereGoodRun, skippedPluginPatch } from './renderStubbed.ts'

interface Helpers {
  click: (
    page: Page,
    role: Parameters<Page['getByRole']>[0],
    name: string | RegExp,
  ) => Promise<void>
  openGameSelect: (page: Page) => Promise<void>
  desk: { width: number; height: number }
  wide: { width: number; height: number }
  profile: { game: string; name: string }
  // A profile whose game has mod projects in the Develop tab.
  developing: { game: string; name: string }
  baldurs: { game: string; name: string }
  coreKeeper: { game: string; name: string }
  cyberpunk: { game: string; name: string }
  settings: (page: Page, section: string) => Promise<void>
  stepMs: number
}

/** The news posts' scenes: 0.3.13's Change version, Add a game and Multiplayer mod list, and 0.3.14's Develop tab and
 * Preferred source, 0.5.0's mod.io sign-in, Load order and Release everywhere, 0.6.0's Manual sort, 0.7.0's
 * Notifications and mod details, 0.8.0's Translate and skipped load-order row, and 0.9.0's Cyberpunk 2077 tools and
 * Convert to REDmod. */
// Asks every site once, so each row of the picture shows the site answering.
async function testEveryAccount(page: Page, stepMs: number) {
  const main = page.getByRole('main')
  for (const name of ['Recheck', 'Test']) {
    const buttons = main.getByRole('button', { name, exact: true })
    for (const button of await buttons.all()) {
      if (await button.isEnabled()) {
        await button.click({ timeout: stepMs })
      }
    }
  }
  await main
    .getByText(/has not asked/u)
    .first()
    .waitFor({ state: 'detached', timeout: 30_000 })
}

const accountsScene = (h: Helpers) => ({
  name: 'accounts',
  file: 'accounts-0-6-0.png',
  ...h.desk,
  profile: h.developing,
  steps: async (page: Page) => {
    await h.settings(page, 'Accounts')
    await testEveryAccount(page, h.stepMs)
    await page
      .getByText('Nexus', { exact: true })
      .first()
      .evaluate((el) => el.scrollIntoView({ block: 'start' }))
  },
})

// Mid-drag in the Manual sort: a mod picked up by its handle and carried down a row, with the line where it would land.
const manualSortScene = (h: Helpers) => ({
  name: 'manual-sort',
  file: 'manual-sort-0-6-0.png',
  ...h.desk,
  profile: h.profile,
  steps: async (page: Page) => {
    await h.click(page, 'tab', 'Mods')
    await h.click(page, 'button', 'List view')
    await h.click(page, 'button', /^Sort$/u)
    await h.click(page, 'menuitemradio', /^Manual/u)
    await page
      .getByRole('button', { name: /^Move / })
      .nth(2)
      .focus({ timeout: h.stepMs })
    await page.keyboard.press('Space')
    await page.getByText(/^Picked up /u).waitFor({ state: 'attached', timeout: h.stepMs })
    // The sensor starts listening a moment after the pick-up is announced and drops an arrow pressed before then.
    const over = page.getByText(/ is over /u)
    for (let tries = 0; tries < 10 && (await over.count()) === 0; tries++) {
      await page.keyboard.press('ArrowDown')
      await over.waitFor({ state: 'attached', timeout: 300 }).catch(() => undefined)
    }
    await over.waitFor({ state: 'attached', timeout: h.stepMs })
  },
})

const requestCountsScene = (h: Helpers) => ({
  ...accountsScene(h),
  name: 'request-counts',
  file: 'request-counts-0-6-0.png',
  steps: async (page: Page) => {
    await accountsScene(h).steps(page)
    await page
      .getByText('GitHub', { exact: true })
      .first()
      .evaluate((el) => el.scrollIntoView({ block: 'start' }))
  },
})

const notificationsScene = (h: Helpers) => ({
  name: 'notifications',
  file: 'notifications-0-7-0.png',
  ...h.desk,
  profile: h.developing,
  steps: async (page: Page) => {
    // The title bar's bell is also a Notifications button, so the section is the last one.
    await h.settings(page, 'General')
    await page
      .getByRole('button', { name: 'Notifications', exact: true })
      .last()
      .click({ timeout: h.stepMs })
    await page
      .getByText('Something finishes well', { exact: true })
      .first()
      .evaluate((el) => el.scrollIntoView({ block: 'center' }))
  },
})

// A mod's details with Change version and Pin on the version's own line.
const modDetailsScene = (h: Helpers) => ({
  name: 'mod-details',
  file: 'mod-details-0-7-0.png',
  ...h.desk,
  profile: h.developing,
  steps: async (page: Page) => {
    await h.click(page, 'tab', 'Mods')
    await page
      .getByRole('button', { name: /^Details of /u })
      .nth(5)
      .click({ timeout: h.stepMs })
    await page.getByRole('button', { name: /^Change version…/u }).waitFor({ timeout: h.stepMs })
  },
})

// A listing's description in Browse with its Translate button.
const translateScene = (h: Helpers) => ({
  name: 'translate',
  file: 'translate-0-8-0.png',
  ...h.desk,
  profile: h.profile,
  steps: async (page: Page) => {
    await h.click(page, 'tab', 'Browse')
    await page.locator('[data-hit]').first().click({ timeout: 20_000 })
    await page.getByRole('button', { name: 'Translate', exact: true }).waitFor({ timeout: 20_000 })
  },
})

// A BepInEx plugin that will not load, with the reason on its row, in the Lethal Company profile's load order.
const skippedLoadOrderScene = (h: Helpers) => ({
  name: 'load-order-skipped',
  file: 'load-order-0-8-0.png',
  ...h.desk,
  profile: h.developing,
  steps: async (page: Page) => {
    await page.route('**/wails/runtime', skippedPluginPatch())
    await h.click(page, 'tab', 'Load order')
    await page
      .getByText('Ship Lights', { exact: true })
      .scrollIntoViewIfNeeded({ timeout: h.stepMs })
  },
})

// Cyberpunk 2077's own tools in the profile's Tools menu (selftest.sh fake-install and seed-cyberpunk).
const gameToolsScene = (h: Helpers) => ({
  name: 'game-tools',
  file: 'game-tools-0-9-0.png',
  ...h.desk,
  profile: h.cyberpunk,
  steps: async (page: Page) => {
    await h.click(page, 'tab', 'Home')
    await h.click(page, 'button', 'Tools')
    await page.getByRole('menuitem', { name: 'REDLauncher' }).waitFor({ timeout: h.stepMs })
  },
})

// An archive mod's menu in the Cyberpunk 2077 profile, with Convert to REDmod.
const convertRedmodScene = (h: Helpers) => ({
  name: 'convert-redmod',
  file: 'convert-redmod-0-9-0.png',
  ...h.desk,
  profile: h.cyberpunk,
  steps: async (page: Page) => {
    await h.click(page, 'tab', 'Mods')
    await page
      .getByText('Rain Puddles', { exact: true })
      .first()
      .click({ button: 'right', timeout: h.stepMs })
    await page.getByRole('menuitem', { name: 'Convert to REDmod' }).waitFor({ timeout: h.stepMs })
  },
})

// The rows are stubbed and the profile empty, so the counts are told to match what the picture shows.
const showStubbedModCount = (page: Page) =>
  page.evaluate(() => {
    for (const el of document.querySelectorAll('span, div')) {
      if (el.childElementCount === 0 && el.textContent === '0 mods') {
        el.textContent = '4 mods'
      }
    }
  })

export const releaseScenes = (h: Helpers) => [
  {
    name: 'change-version',
    ...h.desk,
    profile: h.profile,
    steps: async (page) => {
      await h.click(page, 'tab', 'Mods')
      await page
        .getByRole('button', { name: /^Details of /u })
        .nth(5)
        .click({ timeout: h.stepMs })
      await h.click(page, 'button', /^Change version…/u)
      await page
        .getByRole('dialog')
        .getByRole('list', { name: 'Versions' })
        .waitFor({ timeout: 20_000 })
    },
  },
  {
    name: 'add-game',
    ...h.desk,
    steps: async (page) => {
      await h.openGameSelect(page)
      await h.click(page, 'button', /^Add a game…/u)
      await page.getByRole('dialog').waitFor({ timeout: h.stepMs })
    },
  },
  {
    name: 'multiplayer',
    ...h.wide,
    profile: h.profile,
    steps: async (page) => {
      await page
        .getByRole('button', { name: /^Switch profile/u })
        .click({ button: 'right', timeout: h.stepMs })
      await h.click(page, 'menuitem', /^Multiplayer mod list…/u)
      await page.getByRole('dialog').waitFor({ timeout: h.stepMs })
    },
  },
  {
    name: 'develop',
    ...h.desk,
    profile: h.developing,
    steps: async (page) => {
      await h.click(page, 'tab', 'Develop')
      await page.getByRole('button', { name: 'Game code…' }).waitFor({ timeout: h.stepMs })
      // Each card's icon arrives with the project's details, read after the list shows.
      await page.waitForFunction(
        () => document.querySelectorAll('main img').length >= 5,
        undefined,
        {
          timeout: h.stepMs,
        },
      )
    },
  },
  {
    name: 'source-order',
    ...h.desk,
    profile: h.profile,
    steps: async (page) => {
      await h.click(page, 'tab', 'Home')
      await page
        .getByRole('button', { name: / settings$/u })
        .last()
        .click({ timeout: h.stepMs })
      await page.getByRole('button', { name: /^All .* profiles$/u }).click({ timeout: h.stepMs })
      await page.getByRole('button', { name: 'Mods', exact: true }).click({ timeout: h.stepMs })
      await page
        .getByText('Preferred source', { exact: true })
        .evaluate((el) => el.closest('.settings-tiles > *')?.scrollIntoView({ block: 'start' }))
      // One source off, so the picture shows what the switches do (the sandbox's own setting).
      const curseforge = page.getByRole('switch', { name: 'Search CurseForge' })
      if (await curseforge.isChecked()) {
        await curseforge.click({ timeout: h.stepMs })
      }
    },
  },
  {
    name: 'modio',
    file: 'modio-0-5-0.png',
    ...h.desk,
    profile: h.coreKeeper,
    steps: async (page) => {
      await h.click(page, 'tab', 'Browse')
      // The source's tab is labelled with its brand name; the name match is a substring one, as before.
      await h.click(page, 'button', 'mod.io')
      await page
        .getByText(/results/u)
        .first()
        .waitFor({ timeout: 20_000 })
    },
  },
  accountsScene(h),
  requestCountsScene(h),
  manualSortScene(h),
  notificationsScene(h),
  modDetailsScene(h),
  translateScene(h),
  skippedLoadOrderScene(h),
  gameToolsScene(h),
  convertRedmodScene(h),
  {
    name: 'load-order',
    file: 'load-order-0-5-0.png',
    ...h.desk,
    profile: h.baldurs,
    steps: async (page) => {
      await page.route('**/wails/runtime', loadOrderStub())
      await h.click(page, 'tab', 'Load order')
      await page.getByText('Party Limit Begone', { exact: true }).waitFor({ timeout: h.stepMs })
      await showStubbedModCount(page)
    },
  },
  {
    name: 'release-everywhere',
    file: 'release-everywhere-0-5-0.png',
    ...h.desk,
    profile: h.developing,
    steps: async (page) => {
      await page.route('**/wails/runtime', releaseEverywhereGoodRun())
      await h.click(page, 'tab', 'Develop')
      await page.getByRole('main').getByRole('button', { name: 'More actions' }).first().click()
      await h.click(page, 'menuitem', /^Upload v/u)
      const dialog = page.getByRole('dialog').last()
      await dialog.getByRole('button', { name: 'Release everywhere' }).click()
      await dialog.getByText('Skipped: Not signed in').waitFor({ timeout: h.stepMs })
    },
  },
]
