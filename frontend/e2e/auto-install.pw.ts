import { execFileSync } from 'node:child_process'
import { existsSync, mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { expect, test } from '@playwright/test'
import { openProfile } from './app.ts'
import { serverEnv } from './sandbox.ts'

// A watched folder set to Auto-install lands in the open profile when its files change, and a rebuild replaces it.

const dir = process.env.MORTAR_E2E_DIR ?? ''
const PROFILE = 'Build Output'
let env: Record<string, string> = {}
let profileId = ''
const folder = () => `${dir}/auto-install/Author.BuildPack`
const cli = (...args: string[]) =>
  execFileSync(`${dir}/mortar-server`, args, { env, encoding: 'utf8' }).trim()

function write(version: string, build = version) {
  writeFileSync(
    `${folder()}/manifest.json`,
    JSON.stringify({
      Name: 'Build Pack',
      Author: 'Author',
      Version: version,
      UniqueID: 'Author.BuildPack',
      EntryDll: 'BuildPack.dll',
    }),
  )
  // A build writes its assembly beside the manifest; the bytes need not be a real one.
  writeFileSync(`${folder()}/BuildPack.dll`, `build ${build}`)
}

test.beforeAll(() => {
  env = serverEnv(dir)
  rmSync(`${dir}/auto-install`, { recursive: true, force: true })
  mkdirSync(folder(), { recursive: true })
  profileId = cli('profile', 'create', 'stardew', PROFILE).split('\t')[0] ?? ''
  cli('settings', 'set', 'autoInstallFolders', folder())
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
  // A rebuild that keeps the version is still installed: a developer's next build rarely bumps it.
  write('1.0.1', 'second')
  await expect.poll(installedDll, { timeout: 15_000 }).toBe('build second')
  await expect(rows).toHaveCount(1)
})

// The profile's copy of the build's assembly.
function installedDll() {
  const mods = `${env.XDG_DATA_HOME || `${env.HOME}/.local/share`}/mortar/profiles/stardew/${profileId}/mods`
  for (const entry of readdirSync(mods)) {
    const dll = `${mods}/${entry}/BuildPack.dll`
    if (existsSync(dll)) {
      return readFileSync(dll, 'utf8')
    }
  }
  return ''
}
