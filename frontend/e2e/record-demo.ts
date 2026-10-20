// Records the backend answers the website's demo replays, from a fresh e2e sandbox: bun frontend/e2e/record-demo.ts
import { writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { chromium, type Request } from '@playwright/test'
import { callKey, type Recording, type Reply } from '../src/demo/replay.ts'
import { openSeedFarm } from './app.ts'
import { freshSandbox, sandboxPort } from './sandbox.ts'

const OUT = fileURLToPath(new URL('../src/demo/generated/recording.json', import.meta.url))
const STEP_MS = 10_000
// Calls a click starts land a moment after the view shows.
const SETTLE_MS = 1500
// game.Service.List answers the whole catalog; the demo keeps the installed games, without Steam's art, which Mortar
// never re-hosts.
const GAME_LIST = '{"object":0,"method":0,"args":{"methodID":2813804068,"args":[]}}'

const isRecord = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null

function installedGames(games: unknown): unknown {
  return Array.isArray(games)
    ? games
        .filter(isRecord)
        .filter((g) => g.installed === true)
        .map((g) => ({ ...g, artUrl: '' }))
    : games
}

async function reply(request: Request): Promise<[string, Reply] | undefined> {
  const response = await request.response()
  const body = request.postData()
  if (!(response && body)) {
    return undefined
  }
  const key = callKey(body)
  const status = response.status()
  const text = await response.text()
  if (!(response.headers()['content-type'] ?? '').includes('application/json')) {
    return [key, { status, text }]
  }
  const json: unknown = JSON.parse(text)
  return [key, { status, json: key === GAME_LIST ? installedGames(json) : json }]
}

async function record() {
  const browser = await chromium.launch()
  const context = await browser.newContext({
    baseURL: `http://127.0.0.1:${sandboxPort()}`,
    viewport: { width: 1280, height: 800 },
    colorScheme: 'dark',
  })
  const page = await context.newPage()
  // The first visit answers the welcome and the tour; the recorded one reopens straight on Seed Farm, as the demo does.
  await openSeedFarm(page)
  await page.getByRole('tab', { name: 'Mods' }).click()
  await page.waitForTimeout(SETTLE_MS)

  const pending: Promise<[string, Reply] | undefined>[] = []
  page.on('requestfinished', (request) => {
    if (new URL(request.url()).pathname === '/wails/runtime') {
      pending.push(reply(request))
    }
  })
  await page.reload()
  await page.getByRole('tab', { name: 'Mods' }).click({ timeout: 30_000 })
  const details = page.getByRole('button', { name: /^Details of / })
  await details.first().waitFor({ timeout: STEP_MS })
  for (const button of await details.all()) {
    await button.click()
    await page.waitForTimeout(SETTLE_MS)
  }
  await page.getByRole('tab', { name: 'Problems' }).click()
  await page.getByRole('button', { name: 'Why?' }).first().click({ timeout: STEP_MS })
  await page.waitForTimeout(SETTLE_MS)
  await browser.close()

  const recording: Recording = {}
  for (const entry of await Promise.all(pending)) {
    // The first answer is the state the views open on; later repeats are the same call refreshed.
    if (entry && !(entry[0] in recording)) {
      recording[entry[0]] = entry[1]
    }
  }
  writeFileSync(OUT, `${JSON.stringify(recording)}\n`)
  console.log(`recorded ${Object.keys(recording).length} calls into ${OUT}`)
}

const teardown = freshSandbox()
try {
  await record()
} finally {
  teardown()
}
