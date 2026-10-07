import { execFileSync } from 'node:child_process'
import { mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { dirname } from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'
import { expect, type Page, test } from '@playwright/test'
import { openGameSelect, openSeedFarm } from './app.ts'
import { serverEnv } from './sandbox.ts'

// The Problems sentences that need compiled assemblies (internal/dotnet/testdata/src/E2E, rebuilt by build.sh): a
// Stardew mod whose writes sit inside another's, and a Lethal Company plugin that declares another one incompatible.

const dir = process.env.MORTAR_E2E_DIR ?? ''
const FIXTURES = fileURLToPath(new URL('../../internal/dotnet/testdata/e2e/', import.meta.url))
const FARM = 'Assembly Texts'
const LOBBY = 'Plugin Texts'
// The overlap check scores a profile only from this many mods that write game members.
const FILLERS = 19
let env: Record<string, string> = {}
const ids: Record<string, string> = {}
const cli = (...args: string[]) =>
  execFileSync(`${dir}/mortar-server`, args, { env, encoding: 'utf8' }).trim()

function zip(root: string, name: string, folder: string, files: Record<string, string | Buffer>) {
  const at = `${root}/${name}/${folder}`
  mkdirSync(at, { recursive: true })
  for (const [path, body] of Object.entries(files)) {
    mkdirSync(dirname(`${at}/${path}`), { recursive: true })
    writeFileSync(`${at}/${path}`, body)
  }
  const out = `${root}/${name}.zip`
  execFileSync('python3', ['-m', 'zipfile', '-c', out, folder], { cwd: `${root}/${name}` })
  return out
}

function smapiMod(root: string, id: string, name: string, dll: string) {
  const entry = `${id}.dll`
  const manifest = JSON.stringify({
    Name: name,
    Author: `Author ${id}`,
    Version: '1.0.0',
    UniqueID: `Asm.${id}`,
    EntryDll: entry,
  })
  return zip(root, id, `Asm.${id}`, { 'manifest.json': manifest, [entry]: fixture(dll) })
}

const fixture = (dll: string) => readFileSync(`${FIXTURES}${dll}.dll`)

function plugin(root: string, name: string, dll: string) {
  const manifest = JSON.stringify({
    name,
    version_number: '1.0.0',
    author: 'Em',
    website_url: '',
    description: 'Fixture package',
    dependencies: [],
  })
  return zip(root, name, name, {
    'manifest.json': manifest,
    [`plugins/${name}.dll`]: fixture(dll),
  })
}

test.beforeAll(() => {
  env = serverEnv(dir)
  const root = `${dir}/problem-assemblies`
  rmSync(root, { recursive: true, force: true })
  mkdirSync(root, { recursive: true })
  ids.farm = cli('profile', 'create', 'stardew', FARM).split('\t')[0] ?? ''
  const mods = [
    smapiMod(root, 'Small', 'Asm Small', 'Small'),
    smapiMod(root, 'Large', 'Asm Large', 'Large'),
  ]
  for (let i = 0; i < FILLERS; i++) {
    mods.push(smapiMod(root, `Filler${i}`, `Asm Filler ${i}`, 'Filler'))
  }
  for (const z of mods) {
    cli('install', 'stardew', FARM, z)
  }
  ids.lobby = cli('profile', 'create', 'lethal-company', LOBBY).split('\t')[0] ?? ''
  for (const z of [plugin(root, 'E2EAlpha', 'Alpha'), plugin(root, 'E2EBeta', 'Beta')]) {
    cli('install', 'lethal-company', LOBBY, z)
  }
})

test.afterAll(() => {
  if (ids.farm) {
    cli('profile', 'delete', 'stardew', ids.farm)
  }
  if (ids.lobby) {
    cli('profile', 'delete', 'lethal-company', ids.lobby)
  }
})

async function expectSentence(page: Page, section: string, sentence: string | RegExp) {
  await page.locator(`[data-section="${section}"]`).click()
  const row = page.getByText(sentence).first()
  await expect(row).toBeVisible()
  const text = await row.innerText()
  for (const broken of ['undefined', '{0}', '{1}', 'for ;', ' ;']) {
    expect(text).not.toContain(broken)
  }
}

test('an overlapping C# mod and a BepInEx incompatibility read as whole sentences', async ({
  page,
}) => {
  await openSeedFarm(page)
  await page
    .getByRole('button', { name: new RegExp(`^(Open )?${FARM}`) })
    .first()
    .click()
  await page.getByRole('tab', { name: /^Problems/ }).click()
  await expectSentence(page, 'redundant', /Overlaps with Asm Large: both change \S.*\S/)

  await openGameSelect(page)
  await page
    .getByRole('button', { name: 'Open Lethal Company' })
    .click({ position: { x: 8, y: 8 } })
  await page
    .getByRole('button', { name: new RegExp(`^(Open )?${LOBBY}`) })
    .first()
    .click()
  await page.getByRole('tab', { name: /^Problems/ }).click()
  await expectSentence(
    page,
    'loadFailures',
    'E2E Alpha declares itself incompatible with E2EBeta, which is also enabled.',
  )
})
