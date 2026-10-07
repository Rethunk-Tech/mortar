import AxeBuilder from '@axe-core/playwright'
import { expect, type Page, test } from '@playwright/test'
import {
  openGameSelect,
  openGameSettings,
  openSettings as openMortarSettings,
  openSeedFarm,
} from './app.ts'

const WCAG_AA = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']
const THEMES = ['Dark', 'Light'] as const
const GAMES = [
  { name: 'Stardew Valley', profile: 'Seed Farm' },
  { name: 'Lethal Company', profile: 'Seed Lobby' },
]

// Only colour depends on the theme, so the Light pass runs the contrast rule alone and the Dark pass every rule.
let onlyRules: string[] | null = null

/** Axe's WCAG 2.1 A and AA rules on the page as it stands, as "screen: rule: targets" lines. */
async function scan(page: Page, screen: string): Promise<string[]> {
  await page.evaluate(
    () => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))),
  )
  const axe = new AxeBuilder({ page })
    // Mortar draws no frames, and the default mode opens a blank page per scan to merge frame results.
    .setLegacyMode()
    // Experimental, so off unless named.
    .options({
      rules: { 'label-content-name-mismatch': { enabled: true } },
      // Passes and incomplete results are never read, and serialising them doubles a scan.
      resultTypes: ['violations'],
    })
  const { violations } = await (onlyRules
    ? axe.withRules(onlyRules)
    : axe.withTags(WCAG_AA)
  ).analyze()
  return violations.map(
    (v) => `${screen}: ${v.id}: ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`,
  )
}

async function setTheme(page: Page, theme: (typeof THEMES)[number]) {
  await openMortarSettings(page)
  await page.getByRole('textbox', { name: 'Search settings' }).fill('Theme')
  await page.getByRole('group', { name: 'Theme' }).getByRole('button', { name: theme }).click()
  await page.keyboard.press('Escape')
}

/** The open settings screen's first page, or every page. */
async function scanSettings(page: Page, screen: string, every = true): Promise<string[]> {
  const nav = page.getByRole('navigation', { name: 'Settings sections' })
  const found: string[] = []
  const pages = await nav.locator('button:not([aria-label])').all()
  for (const button of every ? pages : pages.slice(0, 1)) {
    await button.click()
    found.push(...(await scan(page, `${screen} › ${await button.innerText()}`)))
  }
  await page.keyboard.press('Escape')
  return found
}

async function scanGame(
  page: Page,
  game: (typeof GAMES)[number],
  theme: (typeof THEMES)[number],
): Promise<string[]> {
  const found: string[] = []
  // The views, the profile editor's tabs and the settings pages are the same screens for every game and theme, so
  // only the first game in Dark walks them all; the rest scan what is specific to them.
  const deep = theme === 'Dark' && game === GAMES[0]
  await page.getByRole('button', { name: `Open ${game.name}` }).click({ position: { x: 8, y: 8 } })
  await expect(page.getByRole('tab', { name: 'Mods' })).toBeVisible()
  // Mods is scanned below in each of its views.
  const tabs = page
    .getByRole('tablist', { name: 'Profile sections' })
    .getByRole('tab')
    .filter({ hasNotText: /^Mods$/ })
  for (const tab of await tabs.all()) {
    await tab.click()
    const [name] = (await tab.innerText()).split('\n')
    found.push(...(await scan(page, `${game.name} › ${name}`)))
  }

  await page.getByRole('tab', { name: 'Mods' }).click()
  for (const view of deep ? ['List view', 'Grid view'] : ['Grid view']) {
    await page.getByRole('button', { name: view }).click()
    found.push(...(await scan(page, `${game.name} › Mods ${view}`)))
  }
  await page
    .getByRole('button', { name: /^Details of / })
    .first()
    .click()
  await expect(page.getByRole('complementary', { name: 'Selected mod' })).toBeVisible()
  found.push(...(await scan(page, `${game.name} › Mods details`)))
  await page.keyboard.press('Escape')

  await page.getByRole('tab', { name: 'Home' }).click()
  await page.getByRole('button', { name: 'Profile', exact: true }).click()
  await page.getByRole('menuitem', { name: 'Edit profile' }).click()
  const editor = page.getByRole('dialog', { name: 'Edit profile' })
  await expect(editor).toBeVisible()
  const editorTabs = await editor
    .getByRole('tablist', { name: 'Profile settings' })
    .getByRole('tab')
    .all()
  for (const tab of deep ? editorTabs : editorTabs.slice(0, 1)) {
    await tab.click()
    found.push(...(await scan(page, `${game.name} › Edit profile › ${await tab.innerText()}`)))
  }
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog', { name: 'Edit profile' })).toBeHidden()

  await openGameSettings(page, game.name)
  await expect(page.getByRole('navigation', { name: 'Settings sections' })).toBeVisible()
  // The game's settings pages are built from the same rows as Mortar's, which are scanned in both themes.
  found.push(...(await scanSettings(page, `${game.name} settings`, deep)))
  return found
}

test('every main screen of Stardew Valley and Lethal Company passes axe in dark and light', async ({
  page,
}) => {
  test.setTimeout(120_000)
  // Mortar then drops its transitions, so no surface is scanned halfway through fading in.
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await openSeedFarm(page)
  const found: string[] = []
  for (const theme of THEMES) {
    onlyRules = theme === 'Light' ? ['color-contrast'] : null
    await setTheme(page, theme)
    await openGameSelect(page)
    await expect(page.locator('[data-tile]').first()).toBeVisible()
    found.push(...(await scan(page, `${theme} › Game select`)))
    for (const game of GAMES) {
      found.push(...(await scanGame(page, game, theme)).map((v) => `${theme} › ${v}`))
      await openGameSelect(page)
    }
    await page
      .getByRole('button', { name: 'Open Stardew Valley' })
      .click({ position: { x: 8, y: 8 } })

    // Downloads and notifications float over any screen, and the Mortar settings are the same for every game.
    await page.keyboard.press('Control+j')
    await expect(page.getByRole('dialog', { name: 'Downloads' })).toBeVisible()
    found.push(...(await scan(page, `${theme} › Downloads`)))
    await page.keyboard.press('Escape')
    await page.getByRole('button', { name: /^Notifications/ }).click()
    await expect(page.getByRole('dialog', { name: 'Notification history' })).toBeVisible()
    found.push(...(await scan(page, `${theme} › Notifications`)))
    await page.keyboard.press('Escape')
    await openMortarSettings(page)
    found.push(...(await scanSettings(page, `${theme} › Mortar settings`)))
  }
  onlyRules = null
  await setTheme(page, 'Dark')
  expect(found, 'axe violations').toEqual([])
})
