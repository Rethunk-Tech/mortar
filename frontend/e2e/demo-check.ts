// Checks a built website demo (bun scripts/site-demo.ts <dir>) served under /demo/: bun frontend/e2e/demo-check.ts <dir>
import { join, normalize } from 'node:path'
import process from 'node:process'
import { chromium, expect, type Page } from '@playwright/test'

const [, , dir] = process.argv
if (!dir) {
  console.error('usage: bun frontend/e2e/demo-check.ts <built demo dir>')
  process.exit(2)
}
const PREFIX = '/demo/'
const CREW = 'Friday Crew'
const SOLO = 'Solo Runs'
const GAME = 'Lethal Company'
// A view still loading or failed after this long is broken for a visitor.
const SETTLED_MS = 3000
const STEP_MS = 10_000
const RUN_MS = 20_000

const server = Bun.serve({
  hostname: '127.0.0.1',
  port: 0,
  fetch(request) {
    const path = new URL(request.url).pathname
    if (!path.startsWith(PREFIX)) {
      return new Response('outside the demo', { status: 404 })
    }
    const file = Bun.file(join(dir, normalize(path.slice(PREFIX.length) || 'index.html')))
    return file
      .exists()
      .then((found) => new Response(found ? file : 'missing', { status: found ? 200 : 404 }))
  },
})
const origin = `http://127.0.0.1:${server.port}`

const problems: string[] = []

async function check(name: string, run: () => Promise<void>) {
  try {
    await run()
  } catch (e) {
    problems.push(`${name}: ${String(e).split('\n')[0]}`)
  }
}

/** Fails a view that, once settled, shows the error screen or anything still loading. */
async function settled(page: Page, where: string) {
  await page.waitForTimeout(SETTLED_MS)
  if ((await page.getByText('Something went wrong').count()) > 0) {
    problems.push(`${where}: Something went wrong`)
  }
  const loading = await page.locator('[role="progressbar"]:visible').count()
  if (loading > 0) {
    problems.push(`${where}: ${loading} spinner(s) still showing`)
  }
}

const sidebar = (page: Page) => page.getByRole('tablist', { name: 'Profile sections' })
const tab = (page: Page, name: string) =>
  sidebar(page)
    .getByRole('tab', { name: new RegExp(`^${name}`) })
    .click({ timeout: STEP_MS })

async function everySection(page: Page) {
  const names = (await sidebar(page).getByRole('tab').allTextContents()).map((t) =>
    t.replace(/\d+$/, ''),
  )
  for (const name of names) {
    await check(`section ${name}`, async () => {
      await tab(page, name)
      await settled(page, name)
    })
  }
}

async function everySetting(page: Page, open: () => Promise<void>, where: string) {
  await check(where, async () => {
    await open()
    const nav = page.getByRole('navigation', { name: 'Settings sections' })
    for (const button of await nav.getByRole('button').all()) {
      const name = (await button.getAttribute('aria-label')) ?? (await button.textContent()) ?? ''
      if (!name.startsWith('Back')) {
        await button.click({ timeout: STEP_MS })
        await settled(page, `${where}: ${name}`)
      }
    }
    await page.keyboard.press('Escape')
    await expect(sidebar(page)).toBeVisible()
  })
}

async function switchTo(page: Page, name: string) {
  await page.getByRole('button', { name: /^Switch profile/ }).click({ timeout: STEP_MS })
  await page.getByRole('menuitemradio', { name: new RegExp(`^${name}`) }).click()
  await expect(page.getByRole('button', { name: /^Switch profile/ })).toContainText(name)
}

// Play runs the recorded launch: running, the console filling, Stop, and the run's results after.
async function play(page: Page) {
  await tab(page, 'Console')
  await page.getByRole('button', { name: 'Play', exact: true }).click()
  await page.getByRole('button', { name: 'Play anyway' }).click({ timeout: STEP_MS })
  const stop = page.getByRole('button', { name: 'Stop game' }).first()
  await expect(stop).toBeVisible({ timeout: STEP_MS })
  // The run's clock starts at Play, not when the run was recorded.
  await expect(page.getByText(/^0:0\d$/).first()).toBeVisible()
  const failed = page.getByText(/Could not load \[ReservedFlashlightSlot/).first()
  await expect(failed).toBeVisible({ timeout: RUN_MS })
  const search = page.getByRole('textbox', { name: 'Search the log' })
  await search.fill('HarmonyX')
  await expect(page.getByText(/AccessTools\.DeclaredMethod/).first()).toBeVisible()
  await expect(failed).toBeHidden()
  await search.fill('')
  await stop.click()
  await page.getByRole('dialog').getByRole('button', { name: 'Stop game' }).click()
  await expect(page.getByRole('button', { name: 'Play', exact: true })).toBeVisible({
    timeout: RUN_MS,
  })
  await tab(page, 'Performance')
  await expect(page.getByText('MoreCompany').first()).toBeVisible({ timeout: STEP_MS })
  await settled(page, 'Performance after the run')
}

async function toggles(page: Page) {
  await tab(page, 'Problems')
  await page.getByRole('button', { name: 'Enable' }).first().click({ timeout: STEP_MS })
  await expect(page.getByText(/ReservedFlashlightSlot needs/)).toBeHidden({ timeout: STEP_MS })
  await tab(page, 'Mods')
  await page.getByRole('button', { name: 'List view' }).click()
  await page.getByLabel('Disable More_Suits').click({ timeout: STEP_MS })
  await expect(page.getByLabel('Enable More_Suits')).toBeVisible()
  await settled(page, 'Mods after a switch')
}

async function profiles(page: Page) {
  await page.getByRole('button', { name: /^Switch profile/ }).click()
  await page.getByRole('menuitem', { name: /^New profile/ }).click()
  await page.getByRole('textbox', { name: 'Profile name' }).fill('Demo check')
  await page.getByRole('button', { name: 'Create', exact: true }).click()
  await expect(page.getByRole('button', { name: /^Switch profile/ })).toContainText('Demo check')
  await tab(page, 'Mods')
  await settled(page, 'a new profile')
  await switchTo(page, SOLO)
  await tab(page, 'Mods')
  await settled(page, SOLO)
  await switchTo(page, CREW)
}

// Downloads need the real app: the demo says so and links to it.
async function outside(page: Page) {
  await tab(page, 'Browse')
  await page.getByRole('button', { name: 'Add', exact: true }).first().click({ timeout: STEP_MS })
  const notice = page.getByRole('region', { name: 'Notifications' })
  await expect(notice.getByText('This is a demo')).toBeVisible({ timeout: STEP_MS })
  await expect(notice.getByRole('button', { name: 'Download Mortar' })).toBeVisible()
}

async function walk(page: Page) {
  await page.goto(`${origin}${PREFIX}`)
  await expect(page.getByRole('tab', { name: 'Mods' })).toHaveAttribute('aria-selected', 'true', {
    timeout: 15_000,
  })
  await expect(page.getByRole('button', { name: /^Switch profile/ })).toContainText(CREW)
  await expect(page.getByRole('button', { name: /^Details of / }).first()).toBeVisible()
  await everySection(page)
  await everySetting(
    page,
    async () => {
      await page.getByRole('button', { name: 'Mortar menu' }).click()
      await page.getByRole('menuitem', { name: /^Mortar settings/ }).click()
    },
    'Mortar settings',
  )
  await everySetting(
    page,
    async () => {
      await tab(page, 'Home')
      await page.getByRole('button', { name: `${GAME} settings` }).click()
    },
    `${GAME} settings`,
  )
  await check('play', () => play(page))
  await check('mod switches', () => toggles(page))
  await check('profiles', () => profiles(page))
  await check('outside the demo', () => outside(page))
}

const browser = await chromium.launch()
try {
  const page = await browser.newPage({
    viewport: { width: 1280, height: 800 },
    colorScheme: 'dark',
  })
  page.on('console', (m) => m.type() === 'error' && problems.push(`console: ${m.text()}`))
  page.on('pageerror', (e) => problems.push(`page error: ${e.message}`))
  page.on('request', (r) => {
    if (new URL(r.url()).origin !== origin && !r.url().startsWith('data:')) {
      problems.push(`left the origin: ${r.url()}`)
    }
  })
  page.on('response', (r) => r.status() >= 400 && problems.push(`${r.status()}: ${r.url()}`))
  await walk(page)
} finally {
  await browser.close()
  await server.stop()
}
if (problems.length > 0) {
  console.error(problems.join('\n'))
  process.exit(1)
}
console.log(
  'demo ok: every section and setting settles, Play runs and stops, mods and profiles change, downloads point to the app',
)
