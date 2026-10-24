import { execFileSync } from 'node:child_process'
import { copyFileSync, mkdirSync } from 'node:fs'
import { basename, join } from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'
import { chromium, type Page } from '@playwright/test'
import { dwell } from './app.ts'
import { releaseScenes } from './screenScenes.ts'

// Headless screenshots of the main screens, for a person whose own browser cannot capture the sandbox. Not a spec:
//   bun frontend/e2e/screens.ts --url http://127.0.0.1:9455 --out /var/tmp/screens
// A screen whose controls are not found is skipped with a note.
// --set site instead takes the product screenshots, each at its own size, and puts them where they are cited: WebP in
// web/public/img (cwebp on PATH), and the metainfo and README ones as PNG in build/linux/screenshots. --only a,b takes
// just those scenes.

const arg = (name: string, fallback: string) => {
  const i = process.argv.indexOf(`--${name}`)
  return i >= 0 ? (process.argv[i + 1] ?? fallback) : fallback
}
const url = arg('url', 'http://127.0.0.1:9455')
const out = arg('out', '/var/tmp/screens')
const STEP_MS = 5000
const CHECK_MS = 60_000
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
  await dwell(page, SETTLE_MS)
}

async function shot(page: Page, name: string, steps: () => Promise<void>, file = `${name}.png`) {
  try {
    await steps()
    // A list that is still loading, or a toast about the copied data, would be in the picture.
    for (const loader of [
      page.getByText('Loading…'),
      page.locator('.MuiSkeleton-root, .MuiCircularProgress-root, .MuiLinearProgress-root'),
    ]) {
      await loader
        .first()
        .waitFor({ state: 'hidden', timeout: 15_000 })
        .catch(() => undefined)
    }
    // Toasts are hidden rather than dismissed: the copied data's checks can raise one at any moment.
    await page.addStyleTag({
      content: '[aria-label="Notifications"] { visibility: hidden !important; }',
    })
    await settle(page)
    const path = `${out}/${file}`
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
  await click(page, 'button', /^(Switch game|Choose a game)/u)
  await click(page, 'menuitem', /^All games/u)
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
  const farm = page.getByRole('button', { name: /^(Open )?Seed Farm/u }).first()
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
  file?: string
  width: number
  height: number
  init?: string
  profile?: { game: string; name: string }
  steps: (page: Page) => Promise<void>
}

const DESK = { width: 1280, height: 800 }
const WIDE = { width: 1400, height: 840 }

// The sandbox remembers the last opened profile, so each scene names the one it needs.
async function openNamedProfile(page: Page, game: string, name: string) {
  const gameButton = page.getByRole('button', { name: /^Switch game/u })
  await gameButton.waitFor({ timeout: STEP_MS })
  if (!(await gameButton.getAttribute('aria-label'))?.includes(game)) {
    await gameButton.click()
    await click(page, 'menuitemradio', game)
  }
  await click(page, 'button', /^Switch profile/u)
  await click(page, 'menuitemradio', new RegExp(`^${name} ·`))
  await page.getByRole('tab', { name: 'Mods' }).waitFor({ timeout: STEP_MS })
}

async function settings(page: Page, section: string) {
  await click(page, 'button', 'Mortar menu')
  await click(page, 'menuitem', /^Mortar settings/u)
  await click(page, 'button', section)
}

const STARDEW = { game: 'Stardew Valley', name: 'Main' }
const LETHAL = { game: 'Lethal Company', name: 'Main' }
// The maintainer's measured profile, whose runs logged patch errors.
const LETHAL_MEASURED = { game: 'Lethal Company', name: 'Guided Test' }

// The served sandbox has no Wails environment, so a Linux-only setting needs the runtime told which system it is on.
const AS_LINUX = `window._wails = { environment: { OS: 'linux' } }`

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
  await dwell(page, 150)
  await page.evaluate(() => {
    ;(globalThis as unknown as { padDown: number }).padDown = -1
  })
  await dwell(page, 150)
}

const WEBP_QUALITY = '82'
const METAINFO = ['mods', 'problems', 'performance', 'settings']
const ROOT = fileURLToPath(new URL('../..', import.meta.url))

const SITE: Scene[] = [
  { name: 'profile', ...DESK, profile: STARDEW, steps: (page) => click(page, 'tab', 'Home') },
  {
    name: 'mods',
    ...DESK,
    profile: STARDEW,
    steps: async (page) => {
      await click(page, 'tab', 'Mods')
      // The list is virtualised, so a mod far down it has no row yet: the first row's details stand in.
      await page
        .getByRole('button', { name: /^Details of /u })
        .first()
        .click({ timeout: STEP_MS })
      await page
        .getByRole('button', { name: 'Pin this version' })
        .or(page.getByRole('button', { name: 'Pin version' }))
        .first()
        .waitFor({ timeout: STEP_MS })
    },
  },
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
      // A fresh sandbox's first check can wait on smapi.io and Nexus.
      await page.getByRole('heading', { level: 2 }).first().waitFor({ timeout: CHECK_MS })
      await click(page, 'button', 'Other fixes')
    },
  },
  {
    name: 'performance',
    ...DESK,
    profile: LETHAL_MEASURED,
    steps: async (page) => {
      await click(page, 'tab', 'Performance')
      await page.getByText('Slowest mods').waitFor({ timeout: STEP_MS })
    },
  },
  {
    name: 'patch-errors',
    ...DESK,
    profile: LETHAL_MEASURED,
    steps: async (page) => {
      await click(page, 'tab', 'Problems')
      // Copied data may hold these errors dismissed; restoring them touches only the sandbox.
      const dismissed = page.getByRole('button', { name: /^Dismissed/ })
      if ((await dismissed.count()) > 0) {
        await dismissed.click({ timeout: STEP_MS })
        const restore = page.getByRole('button', { name: 'Restore' })
        for (let left = await restore.count(); left > 0; left = await restore.count()) {
          await restore.first().click({ timeout: STEP_MS })
          await restore.nth(left - 1).waitFor({ state: 'detached', timeout: STEP_MS })
        }
      }
      await page
        .getByText(/threw it, at/u)
        .first()
        .waitFor({ timeout: STEP_MS })
      await page
        .getByRole('heading', { name: /^Errors while running/u })
        .evaluate((h) => h.scrollIntoView({ block: 'start' }))
    },
  },
  {
    name: 'deck',
    ...DESK,
    init: FAKE_PAD,
    steps: async (page) => {
      await openGameSelect(page)
      await page.evaluate(() => dispatchEvent(new Event('gamepadconnected')))
      await padPress(page, DPAD_DOWN)
      await page
        .getByRole('button', { name: /Lethal Company/u })
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
        .getByText(/works once/u)
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
  // Needs a folder in watchFolders: the folder picker is native, so the sandbox's setting is set beforehand.
  {
    name: 'watch-folders',
    ...DESK,
    profile: STARDEW,
    steps: async (page) => {
      await settings(page, 'Downloads')
      await page
        .getByText('Also watch these folders', { exact: true })
        .first()
        .evaluate((h) => h.scrollIntoView({ block: 'center' }))
    },
  },
  {
    name: 'language',
    ...DESK,
    profile: STARDEW,
    steps: async (page) => {
      await settings(page, 'General')
      await click(page, 'combobox', 'Language')
      await page.getByRole('listbox').waitFor({ timeout: STEP_MS })
    },
  },
  {
    name: 'graphics',
    ...DESK,
    profile: STARDEW,
    init: AS_LINUX,
    steps: async (page) => {
      await settings(page, 'Appearance')
      await page
        .getByText('Graphics', { exact: true })
        .first()
        .evaluate((h) => h.scrollIntoView({ block: 'center' }))
    },
  },
  {
    name: 'config',
    ...DESK,
    profile: STARDEW,
    steps: async (page) => {
      await click(page, 'tab', 'Config')
      // The first button is the group's header, which would fold the list away.
      await page
        .getByRole('group', { name: 'Mods with settings' })
        .getByRole('button')
        .nth(1)
        .click({ timeout: STEP_MS })
    },
  },
  ...releaseScenes({
    click,
    openGameSelect,
    desk: DESK,
    wide: WIDE,
    profile: LETHAL_MEASURED,
    stepMs: STEP_MS,
  }),
]

function publishSite() {
  const img = join(ROOT, 'web/public/img')
  for (const path of written) {
    const name = basename(path).replace(/\.png$/, '')
    execFileSync('cwebp', ['-quiet', '-q', WEBP_QUALITY, path, '-o', join(img, `${name}.webp`)])
    if (METAINFO.includes(name)) {
      copyFileSync(path, join(ROOT, 'build/linux/screenshots', `${name}.png`))
    }
  }
}

async function siteShots() {
  mkdirSync(out, { recursive: true })
  const browser = await chromium.launch()
  const only = arg('only', '').split(',').filter(Boolean)
  for (const scene of SITE.filter((s) => only.length === 0 || only.includes(s.name))) {
    const context = await browser.newContext({
      viewport: { width: scene.width, height: scene.height },
      colorScheme: 'dark',
    })
    const page = await context.newPage()
    if (scene.init) {
      await page.addInitScript(scene.init)
    }
    await shot(
      page,
      scene.name,
      async () => {
        await start(page)
        if (scene.profile) {
          await openNamedProfile(page, scene.profile.game, scene.profile.name)
        }
        await scene.steps(page)
        await page.mouse.move(scene.width / 2, 8)
      },
      scene.file,
    )
    await context.close()
  }
  await browser.close()
  publishSite()
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
      await openProfile(page, /^(Open )?Seed Farm/u)
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
    await page.getByRole('dialog', { name: /^Share /u }).waitFor({ timeout: STEP_MS })
  })
  await shot(page, 'merge-dialog', async () => {
    await click(page, 'tab', 'Home')
    await click(page, 'button', /^Profile\b/)
    await click(page, 'menuitem', /^Add these mods to another profile/u)
    const dialog = page.getByRole('dialog', { name: /^Add .+'s mods to/u })
    await dialog
      .getByRole('combobox', { name: 'Profile to add them to' })
      .click({ timeout: STEP_MS })
    await click(page, 'option', 'Seed From Template')
    await dialog
      .getByText(/^(\S+ gets \d+ new mods?|Nothing to add)/u)
      .waitFor({ timeout: STEP_MS })
  })
  await shot(page, 'game-mods-review', async () => {
    await openGameSelect(page)
    await openProfile(page, /^(Open )?Seed Farm/u)
    await click(page, 'tab', 'Mods')
    await click(page, 'button', 'Review…')
    await page.getByRole('dialog', { name: /^Game Mods folder vs /u }).waitFor({ timeout: STEP_MS })
  })
  await shot(page, 'seed-farm-config-alpha', () => configOf(page, /Seed Alpha/u))
  await shot(page, 'seed-farm-config-beta', () => configOf(page, /Seed Beta/u))
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
    await click(page, 'menuitem', /^Mortar settings/u)
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

  // Last, because the held call stays held for the rest of the page's life.
  // The backend finishes a few folders too fast to catch, so the call is held and the dialog is shot as it starts.
  await shot(page, 'game-mods-import-progress', async () => {
    await page.goto(url)
    await openGameSelect(page)
    await openProfile(page, /^(Open )?Seed Farm/u)
    await click(page, 'button', /^Switch profile/u)
    await click(page, 'menuitem', /^From the game's Mods folder/u)
    const dialog = page.getByRole('dialog', { name: /^Import from the game's Mods folder/u })
    await dialog.getByRole('button', { name: 'Import', exact: true }).waitFor({ timeout: STEP_MS })
    await page.route('**/wails/runtime', () => new Promise<void>(() => undefined))
    await dialog.getByRole('button', { name: 'Import', exact: true }).click()
    await dialog.getByRole('progressbar').waitFor({ timeout: STEP_MS })
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
