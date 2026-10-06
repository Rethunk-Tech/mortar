import { execFileSync } from 'node:child_process'
import { mkdirSync, rmSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { expect, type Page, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'
import { serverEnv } from './sandbox.ts'

// One profile of small generated content packs, each pair shaped to give one of the Problems tab's explained rows,
// so every sentence is checked as the app renders it: in its section and with nothing left uninterpolated.

const dir = process.env.MORTAR_E2E_DIR ?? ''
const PROFILE = 'Problem Texts'
const SAVE = 'Texts_1'
let env: Record<string, string> = {}
let profileId = ''
const cli = (...args: string[]) =>
  execFileSync(`${dir}/mortar-server`, args, { env, encoding: 'utf8' }).trim()

type Pack = { author?: string; content: object; files?: Record<string, string> }

const changes = (...list: object[]) => ({ Changes: list })

const PACKS: Record<string, Pack> = {
  BalanceA: {
    content: changes({
      Action: 'EditData',
      Target: 'Data/Buildings',
      Fields: { Coop: { BuildCost: 1000 } },
    }),
  },
  BalanceB: {
    content: changes({
      Action: 'EditData',
      Target: 'Data/Buildings',
      Fields: { Coop: { BuildCost: 5000 } },
    }),
  },
  ShowAlways: {
    content: changes({ Action: 'EditData', Target: 'Data/Events/Town', Entries: { Show: 'a' } }),
  },
  ShowOneDay: {
    content: changes({
      Action: 'EditData',
      Target: 'Data/Events/Town',
      Entries: { Show: 'b' },
      When: { Season: 'Summer', Day: '28' },
    }),
  },
  ForestA: {
    content: changes({
      Action: 'EditMap',
      Target: 'Maps/Farm_Foraging',
      MapProperties: { Music: 'a' },
    }),
  },
  ForestB: {
    content: changes({
      Action: 'EditMap',
      Target: 'Maps/Farm_Foraging',
      MapProperties: { Music: 'b' },
    }),
  },
  EmsDinos: {
    author: 'Em',
    content: changes({
      Action: 'EditImage',
      Target: 'Animals/Dinosaur',
      FromFile: 'dino.png',
      ToArea: { X: 0, Y: 0, Width: 16, Height: 16 },
    }),
    files: { 'dino.png': 'dino' },
  },
  EmsAnimals: {
    author: 'Em',
    content: changes(
      {
        Action: 'EditImage',
        Target: 'Animals/Dinosaur',
        FromFile: 'dino.png',
        ToArea: { X: 0, Y: 0, Width: 16, Height: 16 },
      },
      {
        Action: 'EditImage',
        Target: 'Animals/Dinosaur',
        FromFile: 'dino.png',
        ToArea: { X: 16, Y: 0, Width: 16, Height: 16 },
      },
    ),
    files: { 'dino.png': 'dino' },
  },
  DwarfCave: {
    content: {
      ConfigSchema: { FarmCaveChange: { AllowValues: 'true, false', Default: 'false' } },
      ...changes({
        Action: 'Load',
        Target: 'Maps/FarmCave',
        FromFile: 'dwarf.json',
        Priority: 'Low',
        When: { FarmCaveChange: true },
      }),
    },
    files: { 'dwarf.json': '{"Tile":1}', 'config.json': '{"FarmCaveChange":"true"}' },
  },
  LivelyCave: {
    content: changes({ Action: 'Load', Target: 'Maps/FarmCave', FromFile: 'cave.json' }),
    files: { 'cave.json': '{"Tile":2}' },
  },
}

const NAMES: Record<string, string> = {
  BalanceA: 'Texts Balance A',
  BalanceB: 'Texts Balance B',
  ShowAlways: 'Texts Show Always',
  ShowOneDay: 'Texts Show One Day',
  ForestA: 'Texts Forest A',
  ForestB: 'Texts Forest B',
  EmsDinos: "Em's Dinos",
  EmsAnimals: "Em's Farm Animals",
  DwarfCave: 'Texts Dwarf Cave',
  LivelyCave: 'Texts Lively Cave',
}

function writePacks(root: string): string[] {
  return Object.entries(PACKS).map(([id, pack]) => {
    const folder = `${root}/${id}/Texts.${id}`
    mkdirSync(folder, { recursive: true })
    writeFileSync(
      `${folder}/manifest.json`,
      JSON.stringify({
        Name: NAMES[id],
        Author: pack.author ?? `Author ${id}`,
        Version: '1.0.0',
        UniqueID: `Texts.${id}`,
        ContentPackFor: { UniqueID: 'Pathoschild.ContentPatcher' },
      }),
    )
    writeFileSync(`${folder}/content.json`, JSON.stringify(pack.content))
    for (const [name, body] of Object.entries(pack.files ?? {})) {
      writeFileSync(`${folder}/${name}`, body)
    }
    const zip = `${root}/${id}.zip`
    execFileSync('python3', ['-m', 'zipfile', '-c', zip, `Texts.${id}`], { cwd: `${root}/${id}` })
    return zip
  })
}

const dataDir = () => `${env.XDG_DATA_HOME || `${env.HOME}/.local/share`}/mortar`
const savesDir = () => `${env.HOME}/.config/StardewValley/Saves`

test.beforeAll(() => {
  env = serverEnv(dir)
  const root = `${dir}/problem-texts`
  rmSync(root, { recursive: true, force: true })
  const zips = writePacks(root)
  profileId = cli('profile', 'create', 'stardew', PROFILE).split('\t')[0] ?? ''
  for (const zip of zips) {
    cli('install', 'stardew', PROFILE, zip)
  }
  // A Standard farm save, so the Forest-only overlap is about a farm none of the saves plays.
  mkdirSync(`${savesDir()}/${SAVE}`, { recursive: true })
  writeFileSync(`${savesDir()}/${SAVE}/${SAVE}`, '<SaveGame><whichFarm>0</whichFarm></SaveGame>')
  writeFileSync(`${savesDir()}/${SAVE}/SaveGameInfo`, '<Farmer/>')
  // A launch that never started leaves a failed run with no log.
  const runs = `${dataDir()}/profiles/stardew/${profileId}/runs`
  mkdirSync(runs, { recursive: true })
  writeFileSync(
    `${runs}/index.json`,
    JSON.stringify({ runs: [{ id: 'e2e-run', outcome: 'failed', error: 'exec format error' }] }),
  )
})

test.afterAll(() => {
  rmSync(`${savesDir()}/${SAVE}`, { recursive: true, force: true })
  if (profileId !== '') {
    cli('profile', 'delete', 'stardew', profileId)
  }
})

async function section(page: Page, id: string) {
  await page.locator(`[data-section="${id}"]`).click()
}

async function expectClean(page: Page) {
  const panel = page.getByRole('tabpanel').or(page.locator('main')).first()
  const text = await panel.innerText()
  for (const broken of ['undefined', '{0}', '{1}', 'for ;', ' ;']) {
    expect(text).not.toContain(broken)
  }
}

test('each explained Problems row reads as a whole sentence in its section', async ({ page }) => {
  await openSeedFarm(page)
  await page
    .getByRole('button', { name: new RegExp(`^(Open )?${PROFILE}`) })
    .first()
    .click()
  await page.getByRole('tab', { name: /^Problems/ }).click()

  const rows: [string, string][] = [
    ['cosmetic', "Balance only: one mod's prices win."],
    ['cosmetic', 'Only on Summer 28.'],
    ['cosmetic', 'Only on Forest Farm saves; none of yours is one.'],
    ['settings', "Texts Dwarf Cave's FarmCaveChange is true"],
    [
      'settings',
      'This setting has no effect because Texts Lively Cave loads maps/farmcave over it.',
    ],
    ['loadFailures', 'The last launch failed: exec format error.'],
    ['redundant', "Em's Farm Animals already includes everything it changes"],
  ]
  for (const [id, sentence] of rows) {
    await section(page, id)
    await expect(page.getByText(sentence, { exact: false }).first()).toBeVisible()
    await expectClean(page)
  }
})
