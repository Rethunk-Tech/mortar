import { expect, type Page } from '@playwright/test'

/** Opens the seeded Stardew Valley on the Seed Farm profile with the first-run tour dismissed. */
export async function openSeedFarm(page: Page) {
  await page.goto('/')
  const welcome = page.getByRole('button', { name: 'Continue' })
  const farm = page.getByRole('button', { name: /^(Open )?Seed Farm/ }).first()
  // Mortar reopens the last game, which an earlier spec may have left on another one.
  const otherGame = page.getByRole('tablist', { name: 'Profile sections' })
  await expect(welcome.or(farm).or(otherGame).first()).toBeVisible({ timeout: 30_000 })
  if (await welcome.isVisible()) {
    await welcome.click()
  }
  if (!(await farm.isVisible())) {
    await openGameSelect(page)
  }
  const mods = page.getByRole('tab', { name: 'Mods' })
  // A click can land while the list is still settling; clicking again opens it, as openProfile does.
  for (let attempt = 0; attempt < 3 && !(await mods.isVisible()); attempt++) {
    if (await farm.isVisible({ timeout: 5000 }).catch(() => false)) {
      await farm.click()
    }
    await mods.waitFor({ timeout: 8000 }).catch(() => undefined)
  }
  await expect(mods).toBeVisible()
  // The tour shows once per sandbox, so only the first spec sees it.
  const skip = page.getByRole('button', { name: 'Skip tour' })
  if (await skip.isVisible({ timeout: 1500 }).catch(() => false)) {
    await skip.click()
  }
}

const escapeRegExp = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

/** Opens the named profile of whichever game holds it, past the welcome screen and the first-run tour. */
export async function openProfile(page: Page, name: string) {
  await page.goto('/')
  const welcome = page.getByRole('button', { name: 'Continue' })
  const profile = page
    .getByRole('button', { name: new RegExp(`^(Open )?${escapeRegExp(name)}`) })
    .first()
  const tabs = page.getByRole('tablist', { name: 'Profile sections' })
  await welcome.or(profile).or(tabs).first().waitFor({ timeout: 30_000 })
  if (await welcome.isVisible()) {
    await welcome.click()
  }
  if (!(await profile.isVisible())) {
    await page.getByRole('button', { name: /^(Switch game|Choose a game)/ }).click()
    await page.getByRole('menuitem', { name: /^All games/ }).click()
  }
  const mods = page.getByRole('tab', { name: 'Mods' })
  // The tiles animate under the pointer, so a click can land between them; clicking again settles it.
  for (let attempt = 0; attempt < 3 && !(await mods.isVisible()); attempt++) {
    if (await profile.isVisible({ timeout: 5000 }).catch(() => false)) {
      await profile.click()
    }
    await mods.waitFor({ timeout: 8000 }).catch(() => undefined)
  }
  await mods.waitFor({ timeout: 1000 })
  const skip = page.getByRole('button', { name: 'Skip tour' })
  if (await skip.isVisible({ timeout: 1500 }).catch(() => false)) {
    await skip.click()
  }
}

/** Switches the open game's profile from the title bar's profile menu. */
export async function switchProfile(page: Page, name: string) {
  await page.getByRole('button', { name: /^Switch profile/ }).click()
  await page.getByRole('menuitemradio', { name: new RegExp(`^${name}`) }).click()
}

/** Opens the open game's settings from the Home tab. */
export async function openGameSettings(page: Page, game = 'Stardew Valley') {
  await page.getByRole('tab', { name: 'Home' }).click()
  await page.getByRole('button', { name: `${game} settings` }).click()
}

/** Opens Game select from the title bar's game menu. */
export async function openGameSelect(page: Page) {
  await page.getByRole('button', { name: /^(Switch game|Choose a game)/ }).click()
  await page.getByRole('menuitem', { name: /^All games/ }).click()
}

/** Opens Mortar's settings from the Mortar menu. */
export async function openSettings(page: Page) {
  await page.getByRole('button', { name: 'Mortar menu' }).click()
  await page.getByRole('menuitem', { name: /^Mortar settings/ }).click()
}

const isRecord = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null

// The arguments a Wails runtime call's request body passes to the bound method ({object, method, args: {methodID,
// args}}).
function callArgs(body: string): unknown[] {
  let parsed: unknown
  try {
    parsed = JSON.parse(body)
  } catch {
    return []
  }
  const call = isRecord(parsed) ? parsed.args : undefined
  return isRecord(call) && Array.isArray(call.args) ? call.args : []
}

// Whether a Wails runtime call's request body passes value as one of its arguments, compared exactly rather than found
// anywhere in the body's text.
export function calledWith(body: string, value: unknown): boolean {
  return callArgs(body).some((arg) => arg === value)
}

const NOTICE_WAIT_MS = 30_000
const NOTICE_QUIET_MS = 3000
const NOTICE_POLL_MS = 250

// Start-up notices (updates found once the first check finishes, checks paused) show and expire in the first seconds;
// they are not the first case's own, so measuring waits until none has shown for a while.
export async function waitForQuietNotices(page: Page) {
  const region = page.getByRole('region', { name: 'Notifications' })
  let lastSeen = Date.now()
  for (const end = Date.now() + NOTICE_WAIT_MS; Date.now() < end; ) {
    if ((await region.textContent().catch(() => '')) !== '') {
      lastSeen = Date.now()
    } else if (Date.now() - lastSeen > NOTICE_QUIET_MS) {
      return
    }
    await page.waitForTimeout(NOTICE_POLL_MS)
  }
}
