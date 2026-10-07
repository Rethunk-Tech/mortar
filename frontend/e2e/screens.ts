import { mkdirSync } from 'node:fs'
import process from 'node:process'
import { chromium, type Page } from '@playwright/test'

// Headless screenshots of the main screens, for a person whose own browser cannot capture the sandbox. Not a spec:
//   bun frontend/e2e/screens.ts --url http://127.0.0.1:9455 --out /var/tmp/screens
// A screen whose controls are not found is skipped with a note.

const arg = (name: string, fallback: string) => {
  const i = process.argv.indexOf(`--${name}`)
  return i >= 0 ? (process.argv[i + 1] ?? fallback) : fallback
}
const url = arg('url', 'http://127.0.0.1:9455')
const out = arg('out', '/var/tmp/screens')
const STEP_MS = 5000
const SETTLE_MS = 400

const written: string[] = []
const skipped: string[] = []

async function settle(page: Page) {
  await page.waitForTimeout(SETTLE_MS)
}

async function shot(page: Page, name: string, steps: () => Promise<void>) {
  try {
    await steps()
    await settle(page)
    const path = `${out}/${name}.png`
    await page.screenshot({ path })
    written.push(path)
  } catch (e) {
    skipped.push(`${name}: ${String(e).split('\n')[0]}`)
  }
  await page.keyboard.press('Escape').catch(() => undefined)
}

const click = (page: Page, role: Parameters<Page['getByRole']>[0], name: string | RegExp) =>
  page.getByRole(role, { name }).first().click({ timeout: STEP_MS })

async function openGameSelect(page: Page) {
  await click(page, 'button', /^(Switch game|Choose a game)/)
  await click(page, 'menuitem', /^All games/)
}

async function openProfile(page: Page, profile: RegExp) {
  const button = page.getByRole('button', { name: profile }).first()
  await button.waitFor({ timeout: STEP_MS })
  await button.click()
  await page.getByRole('tab', { name: 'Mods' }).waitFor({ timeout: STEP_MS })
}

async function start(page: Page) {
  await page.goto(url)
  const welcome = page.getByRole('button', { name: 'Continue' })
  const tabs = page.getByRole('tablist', { name: 'Profile sections' })
  const farm = page.getByRole('button', { name: /^(Open )?Seed Farm/ }).first()
  await welcome.or(farm).or(tabs).first().waitFor({ timeout: 30_000 })
  if (await welcome.isVisible()) {
    await welcome.click()
  }
  const skip = page.getByRole('button', { name: 'Skip tour' })
  if (await skip.isVisible({ timeout: 1500 }).catch(() => false)) {
    await skip.click()
  }
}

async function configOf(page: Page, mod: RegExp) {
  await click(page, 'tab', 'Config')
  await page
    .getByRole('list', { name: 'Mods with settings' })
    .getByRole('button', { name: mod })
    .click({ timeout: STEP_MS })
}

async function main() {
  mkdirSync(out, { recursive: true })
  const browser = await chromium.launch()
  const page = await (
    await browser.newContext({ viewport: { width: 1400, height: 840 }, colorScheme: 'dark' })
  ).newPage()
  await start(page)

  await shot(page, 'game-select', () => openGameSelect(page))
  await shot(page, 'seed-farm-home', async () => {
    if (!(await page.getByRole('tab', { name: 'Mods' }).isVisible())) {
      await openGameSelect(page)
      await openProfile(page, /^(Open )?Seed Farm/)
    }
    await click(page, 'tab', 'Home')
  })
  await shot(page, 'seed-farm-mods', () => click(page, 'tab', 'Mods'))
  await shot(page, 'command-palette', async () => {
    await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur())
    await page.keyboard.press('Control+k')
    await page.getByRole('dialog').getByRole('combobox').waitFor({ timeout: STEP_MS })
  })
  await shot(page, 'share-dialog', async () => {
    await click(page, 'tab', 'Home')
    await click(page, 'button', 'Share…')
    await page.getByRole('dialog', { name: /^Share / }).waitFor({ timeout: STEP_MS })
  })
  await shot(page, 'seed-farm-config-alpha', () => configOf(page, /Seed Alpha/))
  await shot(page, 'seed-farm-config-beta', () => configOf(page, /Seed Beta/))
  await shot(page, 'seed-farm-problems', () => click(page, 'tab', 'Problems'))
  await shot(page, 'seed-farm-saves', () => click(page, 'tab', 'Saves'))
  await shot(page, 'seed-farm-history', async () => {
    await click(page, 'button', /^Notifications/)
    await click(page, 'button', 'All changes…')
    await page.getByRole('dialog', { name: 'History' }).waitFor({ timeout: STEP_MS })
  })
  await shot(page, 'settings-general', async () => {
    await click(page, 'button', 'Mortar menu')
    await click(page, 'menuitem', /^Mortar settings/)
    await click(page, 'button', 'General')
    await page.getByText('Antivirus', { exact: true }).first().scrollIntoViewIfNeeded()
  })
  await shot(page, 'lethal-company-config', async () => {
    await page.goto(url)
    await page.getByRole('tab', { name: 'Mods' }).waitFor({ timeout: STEP_MS })
    await click(page, 'button', 'Stardew Valley')
    await click(page, 'menuitemradio', 'Lethal Company')
    await page.getByRole('button', { name: 'Seed Lobby' }).first().waitFor({ timeout: STEP_MS })
    await click(page, 'tab', 'Config')
  })

  await browser.close()
  console.log(`wrote ${written.length}:\n${written.join('\n')}`)
  if (skipped.length > 0) {
    console.log(`skipped ${skipped.length}:\n${skipped.join('\n')}`)
  }
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
