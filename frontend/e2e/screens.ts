import { mkdirSync } from 'node:fs'
import process from 'node:process'
import { chromium, type Page } from '@playwright/test'

// Headless screenshots of the main screens, for a person whose own browser cannot capture the sandbox. Not a spec:
//   bun frontend/e2e/screens.ts --url http://127.0.0.1:9455 --out /var/tmp/screens
// A screen whose controls are not found is skipped with a note.
// --set site instead writes the product screenshots the site, README and Linux metainfo cite (site/img, build/linux/screenshots),
// each at its own size, as PNGs in --out; convert with
//   for f in $out/*.png; do cwebp -q 82 $f -o site/img/$(basename $f .png).webp; done
// and copy mods, problems, performance and settings to build/linux/screenshots as PNG.

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
  await page.evaluate(() => document.fonts.ready)
  await page.waitForFunction(
    () =>
      [...document.images].every((img) => {
        const r = img.getBoundingClientRect()
        const offscreen = r.bottom < 0 || r.top > innerHeight || r.right < 0 || r.left > innerWidth
        return offscreen || (img.complete && img.naturalWidth > 0)
      }),
    undefined,
    { timeout: 15_000 },
  )
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
    .getByRole('group', { name: 'Mods with settings' })
    .getByRole('button', { name: mod })
    .click({ timeout: STEP_MS })
}

interface Scene {
  name: string
  width: number
  height: number
  pad?: boolean
  profile?: { game: string; name: string }
  steps: (page: Page) => Promise<void>
}

const DESK = { width: 1280, height: 800 }
const WIDE = { width: 1400, height: 840 }

// The sandbox remembers the last opened profile, so each scene names the one it needs.
async function openNamedProfile(page: Page, game: string, name: string) {
  const gameButton = page.getByRole('button', { name: /^Switch game/ })
  await gameButton.waitFor({ timeout: STEP_MS })
  if (!(await gameButton.getAttribute('aria-label'))?.includes(game)) {
    await gameButton.click()
    await click(page, 'menuitemradio', game)
  }
  await click(page, 'button', /^Switch profile/)
  await click(page, 'menuitemradio', new RegExp(`^${name} ·`))
  await page.getByRole('tab', { name: 'Mods' }).waitFor({ timeout: STEP_MS })
}

async function settings(page: Page, section: string) {
  await click(page, 'button', 'Mortar menu')
  await click(page, 'menuitem', /^Mortar settings/)
  await click(page, 'button', section)
}

const STARDEW = { game: 'Stardew Valley', name: 'Main' }
const LETHAL = { game: 'Lethal Company', name: 'Main' }

// Steam Deck mode is the roomy theme a gamepad's first press turns on, so the scene drives a fake standard-mapping pad.
const DPAD_DOWN = 13
const FAKE_PAD = `
  window.padDown = -1
  navigator.getGamepads = () => [{
    connected: true,
    axes: [0, 0],
    buttons: Array.from({ length: 17 }, (_, i) => ({ pressed: i === window.padDown })),
  }]
`

async function padPress(page: Page, button: number) {
  await page.evaluate((b) => {
    ;(globalThis as unknown as { padDown: number }).padDown = b
  }, button)
  await page.waitForTimeout(150)
  await page.evaluate(() => {
    ;(globalThis as unknown as { padDown: number }).padDown = -1
  })
  await page.waitForTimeout(150)
}

const SITE: Scene[] = [
  { name: 'games', ...WIDE, steps: openGameSelect },
  { name: 'profile', ...DESK, profile: STARDEW, steps: (page) => click(page, 'tab', 'Home') },
  {
    name: 'mods',
    ...DESK,
    profile: STARDEW,
    steps: async (page) => {
      await click(page, 'tab', 'Mods')
      await click(page, 'button', /^Details of Stardew Valley Expanded/)
      await page
        .getByRole('button', { name: 'Pin this version' })
        .or(page.getByRole('button', { name: 'Pin version' }))
        .first()
        .waitFor({ timeout: STEP_MS })
    },
  },
  { name: 'lethal-company', ...WIDE, profile: LETHAL, steps: (page) => click(page, 'tab', 'Mods') },
  {
    name: 'thunderstore',
    ...WIDE,
    profile: LETHAL,
    steps: async (page) => {
      await click(page, 'tab', 'Browse')
      await click(page, 'button', /^Thunderstore/)
      await page
        .getByText(/results/)
        .first()
        .waitFor({ timeout: 20_000 })
    },
  },
  {
    name: 'problems',
    ...DESK,
    profile: STARDEW,
    steps: async (page) => {
      await click(page, 'tab', 'Problems')
      await click(page, 'button', 'Why?')
    },
  },
  {
    name: 'performance',
    ...DESK,
    profile: STARDEW,
    steps: (page) => click(page, 'tab', 'Performance'),
  },
  {
    name: 'deck',
    ...DESK,
    pad: true,
    steps: async (page) => {
      await openGameSelect(page)
      await page.evaluate(() => dispatchEvent(new Event('gamepadconnected')))
      await padPress(page, DPAD_DOWN)
      await page
        .getByRole('button', { name: /Lethal Company/ })
        .first()
        .focus()
    },
  },
  {
    name: 'pairing',
    ...WIDE,
    profile: STARDEW,
    steps: async (page) => {
      await settings(page, 'General')
      await click(page, 'button', 'Pair a computer')
      await page
        .getByRole('dialog')
        .getByText(/works once/)
        .waitFor({ timeout: STEP_MS })
    },
  },
  {
    name: 'settings',
    ...DESK,
    profile: STARDEW,
    steps: async (page) => {
      await settings(page, 'General')
      await page
        .getByText('Antivirus', { exact: true })
        .first()
        .evaluate((h) => h.scrollIntoView({ block: 'start' }))
    },
  },
]

async function siteShots() {
  mkdirSync(out, { recursive: true })
  const browser = await chromium.launch()
  for (const scene of SITE) {
    const context = await browser.newContext({
      viewport: { width: scene.width, height: scene.height },
      colorScheme: 'dark',
    })
    const page = await context.newPage()
    if (scene.pad) {
      await page.addInitScript(FAKE_PAD)
    }
    await shot(page, scene.name, async () => {
      await start(page)
      if (scene.profile) {
        await openNamedProfile(page, scene.profile.game, scene.profile.name)
      }
      await scene.steps(page)
      await page.mouse.move(scene.width / 2, 8)
    })
    await context.close()
  }
  await browser.close()
  console.log(`wrote ${written.length}:\n${written.join('\n')}`)
  if (skipped.length > 0) {
    console.log(`skipped ${skipped.length}:\n${skipped.join('\n')}`)
  }
}

async function main() {
  if (arg('set', 'default') === 'site') {
    return siteShots()
  }
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
  await shot(page, 'merge-dialog', async () => {
    await click(page, 'tab', 'Home')
    await click(page, 'button', /^Profile\b/)
    await click(page, 'menuitem', /^Merge into/)
    const dialog = page.getByRole('dialog', { name: /^Merge / })
    await dialog.getByRole('combobox', { name: 'Target profile' }).click({ timeout: STEP_MS })
    await click(page, 'option', 'Seed From Template')
    await dialog.getByText(/^(Adds \d+ mods?|Nothing to merge)/).waitFor({ timeout: STEP_MS })
  })
  await shot(page, 'seed-farm-config-alpha', () => configOf(page, /Seed Alpha/))
  await shot(page, 'seed-farm-config-beta', () => configOf(page, /Seed Beta/))
  await shot(page, 'seed-farm-problems', () => click(page, 'tab', 'Problems'))
  await shot(page, 'seed-farm-performance', () => click(page, 'tab', 'Performance'))
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
