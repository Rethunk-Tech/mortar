import { execFileSync } from 'node:child_process'
import { mkdirSync, rmSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { expect, test } from '@playwright/test'
import { openProfile } from './app.ts'
import { serverEnv } from './sandbox.ts'

// A watched folder set to Auto-install lands in the open profile when its files change, and a rebuild replaces it.

const dir = process.env.MORTAR_E2E_DIR ?? ''
const PROFILE = 'Build Output'
let env: Record<string, string> = {}
let profileId = ''
const build = () => `${dir}/auto-install/Author.BuildPack`
const cli = (...args: string[]) =>
  execFileSync(`${dir}/mortar-server`, args, { env, encoding: 'utf8' }).trim()

function write(version: string) {
  writeFileSync(
    `${build()}/manifest.json`,
    JSON.stringify({
      Name: 'Build Pack',
      Author: 'Author',
      Version: version,
      UniqueID: 'Author.BuildPack',
      EntryDll: 'BuildPack.dll',
    }),
  )
  // A build writes its assembly beside the manifest; the bytes need not be a real one.
  writeFileSync(`${build()}/BuildPack.dll`, `build ${version}`)
}

test.beforeAll(() => {
  env = serverEnv(dir)
  rmSync(`${dir}/auto-install`, { recursive: true, force: true })
  mkdirSync(build(), { recursive: true })
  profileId = cli('profile', 'create', 'stardew', PROFILE).split('\t')[0] ?? ''
  cli('settings', 'set', 'autoInstallFolders', build())
})

test.afterAll(() => {
  cli('settings', 'set', 'autoInstallFolders', '')
  if (profileId !== '') {
    cli('profile', 'delete', 'stardew', profileId)
  }
})

test('a build folder is added on change and replaced on rebuild', async ({ page }) => {
  await openProfile(page, PROFILE)
  write('1.0.0')
  // The view switch shows once the profile holds a mod.
  await page.getByRole('button', { name: 'List view' }).click({ timeout: 15_000 })
  const rows = page.getByRole('row', { name: /Build Pack/ })
  await expect(rows).toHaveCount(1)
  await expect(rows.first()).toContainText('1.0.0')
  write('1.0.1')
  await expect(rows.first()).toContainText('1.0.1', { timeout: 15_000 })
  await expect(rows).toHaveCount(1)
})
