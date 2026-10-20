// Records the backend answers the website's demo replays, from a fresh e2e sandbox: bun frontend/e2e/record-demo.ts
import { writeFileSync } from 'node:fs'
import process from 'node:process'
import { fileURLToPath } from 'node:url'
import { chromium, type Page } from '@playwright/test'
import { openProfile } from './app.ts'
import { CREW, dropBlank, SOLO, seedBlank, seedDemo, standInPath, switchOff } from './demo-seed.ts'
import { Tape } from './demo-tape.ts'
import { missed, switchTo, tab, walkChrome, walkSections } from './demo-walk.ts'
import { freshSandbox, sandboxPort } from './sandbox.ts'

const OUT = fileURLToPath(new URL('../src/demo/generated/recording.json', import.meta.url))
const GAME_NAME = 'Lethal Company'
const BLANK = 'New profile'
const STEP_MS = 10_000
const SETTLE_MS = 1500
// The run is stopped once its startup has finished and the game has logged for a while.
const PLAY_MS = 8000
const LOADED_MS = 40_000

async function toggled(page: Page, tape: Tape) {
  await tab(page, 'Problems')
  tape.phase = 'toggle'
  await page.getByRole('button', { name: 'Enable' }).first().click({ timeout: STEP_MS })
  await page.waitForTimeout(SETTLE_MS)
  for (const name of ['Problems', 'Mods', 'Home', 'Load order']) {
    await tab(page, name)
  }
  tape.phase = 'off'
  switchOff()
  await page.waitForTimeout(SETTLE_MS)
}

async function played(page: Page, tape: Tape) {
  await tab(page, 'Console')
  tape.phase = 'run'
  await page.getByRole('button', { name: 'Play', exact: true }).click({ timeout: STEP_MS })
  // The missing dependency is the profile's problem, which the check before Play names.
  await page.getByRole('button', { name: 'Play anyway' }).click({ timeout: STEP_MS })
  const stop = page.getByRole('button', { name: 'Stop game' }).first()
  await stop.waitFor({ timeout: LOADED_MS })
  await page.waitForTimeout(PLAY_MS)
  await tab(page, 'Performance')
  await tab(page, 'Console')
  await page.waitForTimeout(PLAY_MS)
  await stop.click({ timeout: STEP_MS })
  await page
    .getByRole('dialog')
    .getByRole('button', { name: 'Stop game' })
    .click({ timeout: STEP_MS })
  await page.getByRole('button', { name: 'Play', exact: true }).waitFor({ timeout: LOADED_MS })
  await page.waitForTimeout(SETTLE_MS)
  tape.phase = 'after'
  await walkSections(page, false)
}

async function blank(page: Page, tape: Tape): Promise<string> {
  tape.phase = 'off'
  const id = seedBlank(BLANK)
  await page.waitForTimeout(SETTLE_MS)
  tape.phase = 'blank'
  await switchTo(page, BLANK)
  await walkSections(page, false)
  tape.phase = 'off'
  await switchTo(page, CREW)
  dropBlank(id)
  return id
}

// The sandbox's own paths read as a player's home; the stand-in for the game is not shown at all.
function redact(text: string): string {
  const dir = process.env.MORTAR_E2E_DIR ?? ''
  // Wine's view of the same folder, as JSON escapes it.
  const wine = dir.replaceAll('/', '\\\\')
  return text
    .replaceAll(standInPath(), '')
    .replaceAll(`${wine}\\\\home`, '\\\\home\\\\player')
    .replaceAll(`${dir}/home`, '/home/player')
    .replaceAll(dir, '/home/player/.mortar')
    .replaceAll(`127.0.0.1:${sandboxPort()}`, '127.0.0.1:9455')
}

async function record() {
  const crew = await seedDemo()
  const browser = await chromium.launch()
  const context = await browser.newContext({
    baseURL: `http://127.0.0.1:${sandboxPort()}`,
    viewport: { width: 1280, height: 800 },
    colorScheme: 'dark',
  })
  const page = await context.newPage()
  // The first visit answers the welcome and the tour; the recorded one reopens straight on the profile, as the demo does.
  await openProfile(page, CREW)
  await tab(page, 'Mods')
  const tape = new Tape(page)
  tape.phase = 'base'
  await page.reload()
  await page.getByRole('tab', { name: 'Mods' }).click({ timeout: 30_000 })
  await walkSections(page, true)
  await walkChrome(page, GAME_NAME)
  await openProfile(page, CREW)
  await switchTo(page, SOLO)
  await walkSections(page, true)
  await switchTo(page, CREW)
  await toggled(page, tape)
  await played(page, tape)
  const blankId = await blank(page, tape)
  await browser.close()
  const text = redact(JSON.stringify(await tape.assemble(crew, blankId)))
  writeFileSync(OUT, `${text}\n`)
  console.log(`recorded ${(text.length / 1024).toFixed(0)} KiB into ${OUT}`)
  if (missed.length > 0) {
    throw new Error(`the walk missed:\n${missed.join('\n')}`)
  }
}

const teardown = freshSandbox()
try {
  await record()
} finally {
  teardown()
}
