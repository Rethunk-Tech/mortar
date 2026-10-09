import { execFileSync } from 'node:child_process'
import { existsSync, mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { expect, test } from '@playwright/test'
import { openSeedFarm, switchProfile } from './app.ts'
import { serverEnv } from './sandbox.ts'

// A profile holding one mod id twice: Resolve on its Problems row must open the dialog without a visit to Mods.

const dir = process.env.MORTAR_E2E_DIR ?? ''
const PROFILE = 'Duplicate Dialog'
const NAME = 'Twin Pack'
let env: Record<string, string> = {}
let profileId = ''
const cli = (...args: string[]) =>
  execFileSync(`${dir}/mortar-server`, args, { env, encoding: 'utf8' }).trim()

function pack(root: string, id: string, version: string) {
  const folder = `${root}/${id}/${id}`
  mkdirSync(folder, { recursive: true })
  writeFileSync(
    `${folder}/manifest.json`,
    JSON.stringify({
      Name: NAME,
      Author: 'Author Twin',
      Version: version,
      UniqueID: id,
      ContentPackFor: { UniqueID: 'Pathoschild.ContentPatcher' },
    }),
  )
  writeFileSync(`${folder}/content.json`, JSON.stringify({ Changes: [] }))
  const zip = `${root}/${id}.zip`
  execFileSync('python3', ['-m', 'zipfile', '-c', zip, id], { cwd: `${root}/${id}` })
  return zip
}

const dataDir = () => `${env.XDG_DATA_HOME || `${env.HOME}/.local/share`}/mortar`

test.beforeAll(() => {
  env = serverEnv(dir)
  const root = `${dir}/duplicates`
  rmSync(root, { recursive: true, force: true })
  profileId = cli('profile', 'create', 'stardew', PROFILE).split('\t')[0] ?? ''
  cli('install', 'stardew', PROFILE, pack(root, 'Twin.One', '1.0.0'))
  cli('install', 'stardew', PROFILE, pack(root, 'Twin.Two', '2.0.0'))
  // Install replaces a mod by id, so the second copy is made by renaming its id in the profile's record.
  const home = `${dataDir()}/profiles/stardew/${profileId}`
  const record = `${home}/profile.json`
  const held = readdirSync(`${home}/mods`).find((d) => existsSync(`${home}/mods/${d}/Twin.Two`))
  const manifest = `${home}/mods/${held}/Twin.Two/manifest.json`
  writeFileSync(manifest, readFileSync(manifest, 'utf8').replace('Twin.Two', 'Twin.One'))
  writeFileSync(record, readFileSync(record, 'utf8').replaceAll('smapi:Twin.Two', 'smapi:Twin.One'))
})

test.afterAll(() => {
  if (profileId !== '') {
    cli('profile', 'delete', 'stardew', profileId)
  }
})

test('Resolve on a duplicate in the Problems tab opens the copies dialog', async ({ page }) => {
  await openSeedFarm(page)
  await switchProfile(page, PROFILE)
  await page.getByRole('tab', { name: /^Problems/ }).click()
  await page.locator('[data-section="duplicates"]').click()
  await page.getByRole('button', { name: 'Resolve' }).first().click()
  await expect(page.getByRole('dialog', { name: `2 copies of ${NAME}` })).toBeVisible()
})
