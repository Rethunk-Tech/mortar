// The views the website demo replays, walked on the recorder's sandbox so each one's backend answers are recorded.
import type { Locator, Page } from '@playwright/test'

const STEP_MS = 10_000
// Calls a click starts land a moment after the view shows.
const SETTLE_MS = 1200
const CLOSE_TRIES = 4

/** Steps whose controls were not found; the recorder fails on any. */
export const missed: string[] = []

let current: Page | undefined

// Closes whatever dialogs and menus a step left open.
async function closeAll(page: Page) {
  for (
    let i = 0;
    i < CLOSE_TRIES && (await page.locator('[role="dialog"], [role="menu"]').count()) > 0;
    i++
  ) {
    await page.keyboard.press('Escape')
    await page.waitForTimeout(SETTLE_MS)
  }
}

async function step(name: string, run: () => Promise<void>) {
  try {
    await run()
  } catch (e) {
    missed.push(`${name}: ${String(e).split('\n')[0]}`)
    if (current) {
      await closeAll(current)
    }
  }
}

const settle = (page: Page) => page.waitForTimeout(SETTLE_MS)

async function clickAll(page: Page, buttons: Locator) {
  for (const button of await buttons.all()) {
    await button.click({ timeout: STEP_MS })
    await settle(page)
  }
}

export async function tab(page: Page, name: string) {
  await page
    .getByRole('tablist', { name: 'Profile sections' })
    .getByRole('tab', { name: new RegExp(`^${name}`) })
    .click({ timeout: STEP_MS })
  await settle(page)
}

const pressEscape = async (page: Page) => {
  await page.keyboard.press('Escape')
  await settle(page)
}

/** The sidebar's sections, as their tabs name them. */
async function sections(page: Page): Promise<string[]> {
  const tabs = page.getByRole('tablist', { name: 'Profile sections' }).getByRole('tab')
  return (await tabs.allTextContents()).map((t) => t.replace(/\d+$/, ''))
}

// Each mod's details, from the grid's cards and the list's rows.
async function mods(page: Page) {
  await page.getByRole('button', { name: 'List view' }).click({ timeout: STEP_MS })
  const rows = page
    .getByRole('row')
    .filter({ has: page.getByRole('button', { name: 'Add to favourites' }) })
  await rows.first().waitFor({ timeout: STEP_MS })
  await clickAll(page, rows)
  await page.getByRole('button', { name: 'Grid view' }).click({ timeout: STEP_MS })
  const details = page.getByRole('button', { name: /^Details of / })
  await details.first().waitFor({ timeout: STEP_MS })
  await clickAll(page, details)
}

async function config(page: Page) {
  const list = page.getByRole('group', { name: 'Mods with settings' })
  await list.waitFor({ timeout: STEP_MS })
  await clickAll(page, list.locator('[role="button"]:not([aria-expanded])'))
}

async function browse(page: Page) {
  const sources = page.getByRole('group', { name: 'Source' }).getByRole('button')
  for (const source of await sources.all()) {
    // A source without its key (CurseForge's) cannot be picked.
    if (await source.isDisabled()) {
      continue
    }
    await source.click({ timeout: STEP_MS })
    await page
      .getByText(/results/)
      .first()
      .waitFor({ timeout: 20_000 })
      .catch(() => undefined)
    await settle(page)
  }
}

// Problems' own sections, each with its first "Why?" open.
async function problems(page: Page) {
  const why = page.getByRole('button', { name: 'Why?' })
  for (const list of await page.getByRole('tablist').all()) {
    if ((await list.getAttribute('aria-label')) === 'Profile sections') {
      continue
    }
    for (const t of await list.getByRole('tab').all()) {
      await t.click({ timeout: STEP_MS })
      await settle(page)
      if (await why.first().isVisible()) {
        await why.first().click()
        await settle(page)
      }
    }
  }
}

const EXTRAS: Record<string, (page: Page) => Promise<void>> = {
  Mods: mods,
  Config: config,
  Browse: browse,
  Problems: problems,
}

/** Every sidebar section of the open profile, and with `deep` the first level inside each. */
export async function walkSections(page: Page, deep: boolean) {
  current = page
  for (const name of await sections(page)) {
    await step(name, async () => {
      await tab(page, name)
      const extra = EXTRAS[name]
      if (deep && extra) {
        await extra(page)
      }
    })
  }
}

async function settingsSections(page: Page) {
  const nav = page.getByRole('navigation', { name: 'Settings sections' })
  await nav.waitFor({ timeout: STEP_MS })
  const buttons = nav.getByRole('button')
  for (const button of await buttons.all()) {
    const name = (await button.getAttribute('aria-label')) ?? ''
    // The back button leaves the page.
    if (!name.startsWith('Back')) {
      await button.click({ timeout: STEP_MS })
      await settle(page)
    }
  }
}

async function leaveSettings(page: Page) {
  await pressEscape(page)
  await page.getByRole('tab', { name: 'Mods' }).waitFor({ timeout: STEP_MS })
}

/** The title bar's menus, the profile's dialogs and both settings pages. */
export async function walkChrome(page: Page, game: string) {
  current = page
  await step('game menu', async () => {
    await page.getByRole('button', { name: /^Switch game/ }).click({ timeout: STEP_MS })
    await settle(page)
    await pressEscape(page)
  })
  await step('profile menu', async () => {
    await page.getByRole('button', { name: /^Switch profile/ }).click({ timeout: STEP_MS })
    await settle(page)
    await page.getByRole('menuitem', { name: /^New profile/ }).click({ timeout: STEP_MS })
    await page.getByRole('combobox', { name: 'Start from' }).click({ timeout: STEP_MS })
    await settle(page)
    await pressEscape(page)
    await pressEscape(page)
  })
  await step('profiles page', async () => {
    await page.getByRole('button', { name: /^Switch profile/ }).click({ timeout: STEP_MS })
    await page.getByRole('menuitem', { name: /^Manage profiles/ }).click({ timeout: STEP_MS })
    await settle(page)
    await pressEscape(page)
  })
  await step('share', async () => {
    await tab(page, 'Home')
    await page.getByRole('button', { name: 'Share…' }).click({ timeout: STEP_MS })
    const dialog = page.getByRole('dialog')
    await dialog.waitFor({ timeout: STEP_MS })
    await settle(page)
    // Each way to share but the network one, which pairs with another computer.
    for (const way of ['Mortar', 'Nexus Mods', 'Thunderstore', 'Mod list']) {
      await dialog.getByRole('button', { name: new RegExp(`^${way}`) }).click({ timeout: STEP_MS })
      await settle(page)
      await closeAll(page)
      await page.getByRole('button', { name: 'Share…' }).click({ timeout: STEP_MS })
      await dialog.waitFor({ timeout: STEP_MS })
    }
    await pressEscape(page)
  })
  await step('edit profile', async () => {
    await page.getByRole('button', { name: /^Profile\b/ }).click({ timeout: STEP_MS })
    await page.getByRole('menuitem', { name: /^Edit profile/ }).click({ timeout: STEP_MS })
    await settle(page)
    await pressEscape(page)
  })
  await step('play menu', async () => {
    await page.getByRole('button', { name: 'More play options' }).click({
      timeout: STEP_MS,
    })
    await settle(page)
    await pressEscape(page)
  })
  await step('mortar settings', async () => {
    await page.getByRole('button', { name: 'Mortar menu' }).click({ timeout: STEP_MS })
    await page.getByRole('menuitem', { name: /^Mortar settings/ }).click({ timeout: STEP_MS })
    await settingsSections(page)
    await leaveSettings(page)
  })
  await step('game settings', async () => {
    await tab(page, 'Home')
    await page.getByRole('button', { name: `${game} settings` }).click({ timeout: STEP_MS })
    await settingsSections(page)
    await leaveSettings(page)
  })
  await step('all games', async () => {
    await page.getByRole('button', { name: /^Switch game/ }).click({ timeout: STEP_MS })
    await page.getByRole('menuitem', { name: /^All games/ }).click({ timeout: STEP_MS })
    await settle(page)
  })
}

/** Switches the open game's profile from the title bar's profile menu. */
export async function switchTo(page: Page, name: string) {
  await page.getByRole('button', { name: /^Switch profile/ }).click({ timeout: STEP_MS })
  await page
    .getByRole('menuitemradio', { name: new RegExp(`^${name}`) })
    .click({ timeout: STEP_MS })
  await page.getByRole('tab', { name: 'Mods' }).waitFor({ timeout: STEP_MS })
  await settle(page)
}
