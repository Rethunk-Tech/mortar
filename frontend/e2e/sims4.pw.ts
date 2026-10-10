import { execFileSync } from 'node:child_process'
import { mkdirSync, mkdtempSync, readFileSync, utimesSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { expect, type Page, test } from '@playwright/test'
import { openGameSelect, openGameSettings, openSeedFarm } from './app.ts'

// The Sims 4 through the real UI against the sandbox's stand-in install (scripts/selftest.sh sims4-install): a Steam
// install with a placeholder TS4_x64.exe and the player's Documents tree. The game is disabled in the shipped catalog;
// the sandbox enables it for itself (sandbox.ts). The browser open of an itch.io page is answered by the test.

const dir = process.env.MORTAR_E2E_DIR ?? ''
const DOWNLOADS = `${dir}/home/.local/share/mortar/downloads`
const GAME = 'The Sims 4'
const PROFILE = 'Sims E2E'
const PAGE = 'https://someone.itch.io/cool-mod'

/** Writes `<name>.zip` holding the given files and returns its path. */
function writeZip(into: string, name: string, files: Record<string, string>) {
  const src = mkdtempSync(`${dir}/tmp/sims4-`)
  for (const [path, body] of Object.entries(files)) {
    mkdirSync(`${src}/${path.split('/').slice(0, -1).join('/') || '.'}`, { recursive: true })
    writeFileSync(`${src}/${path}`, body)
  }
  mkdirSync(into, { recursive: true })
  const out = `${into}/${name}.zip`
  execFileSync('python3', [
    '-m',
    'zipfile',
    '-c',
    out,
    ...Object.keys(files).map((f) => `${src}/${f}`),
  ])
  return out
}

/** The id of the NewDownloads binding, so the test can wait for the page's own call to it. */
const NEW_DOWNLOADS_ID = /NewDownloads\(game: string\)[\s\S]*?\$Call\.ByID\((\d+)/.exec(
  readFileSync(
    new URL(
      '../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/service.ts',
      import.meta.url,
    ),
    'utf8',
  ),
)?.[1]

/** Opens The Sims 4 from Game select, making its first profile when it has none. */
async function openSims(page: Page) {
  await openSeedFarm(page)
  await openGameSelect(page)
  await page.getByRole('button', { name: `Open ${GAME}` }).click()
  if (
    !(await page
      .getByRole('button', { name: new RegExp(`^Switch profile.*${PROFILE}`) })
      .isVisible({ timeout: 1500 })
      .catch(() => false))
  ) {
    await page.getByRole('button', { name: /^Switch profile/ }).click()
    await page.getByRole('menuitem', { name: 'New profile…' }).click()
    const dialog = page.getByRole('dialog', { name: 'New profile' })
    await dialog.getByLabel('Profile name').fill(PROFILE)
    await dialog.getByRole('button', { name: 'Create' }).click()
    await expect(page.getByText(new RegExp(`^Created ${PROFILE}`))).toBeVisible()
  }
}

/** Adds a saved archive through the downloads folder dialog. */
async function addFromDownloads(page: Page, archive: string) {
  await page.getByRole('button', { name: 'More ways to add mods' }).click()
  await page.getByRole('menuitem', { name: 'The downloads folder…' }).click()
  const downloads = page.getByRole('dialog', { name: 'From the downloads folder' })
  await downloads.getByRole('option', { name: new RegExp(archive) }).click()
  await downloads.getByRole('button', { name: 'Add to profile' }).click()
  await page.getByRole('tab', { name: 'Mods' }).click()
}

test('Game select lists The Sims 4, found from the stand-in Steam install', async ({ page }) => {
  await openSeedFarm(page)
  await openGameSelect(page)
  await expect(page.getByRole('button', { name: `Open ${GAME}` })).toBeVisible()
})

test('a new Sims 4 profile keeps its saves separate', async ({ page }) => {
  await openSims(page)
  await page.getByRole('tab', { name: 'Home' }).click()
  await page.getByRole('button', { name: 'Profile', exact: true }).click()
  await page.getByRole('menuitem', { name: 'Edit profile' }).click()
  const editor = page.getByRole('dialog', { name: 'Edit profile' })
  const separate = editor.getByLabel("Keep this profile's saves separate")
  for (const tab of await editor
    .getByRole('tablist', { name: 'Profile settings' })
    .getByRole('tab')
    .all()) {
    if (await separate.isVisible()) {
      break
    }
    await tab.click()
  }
  await expect(separate).toBeChecked()
})

test('a three-package archive lists three rows and a script archive one', async ({ page }) => {
  await openSims(page)
  await page.getByRole('tab', { name: 'Mods' }).click()
  const rows = page.getByRole('button', { name: /^Details of / })
  const before = await rows.count()
  writeZip(DOWNLOADS, 'Three.Packages', {
    'one.package': '1',
    'two.package': '2',
    'three.package': '3',
  })
  await addFromDownloads(page, 'Three.Packages')
  await expect(rows).toHaveCount(before + 3)
  writeZip(DOWNLOADS, 'Script.Mod', { 'Mod/m.ts4script': 's', 'Mod/m.package': 'p' })
  await addFromDownloads(page, 'Script.Mod')
  await expect(rows).toHaveCount(before + 4)
})

test('game settings offer the settings file mode and cache clearing for The Sims 4 and not for Stardew Valley', async ({
  page,
}) => {
  await openSims(page)
  await openGameSettings(page, GAME)
  await expect(page.getByText('Game settings file', { exact: true }).first()).toBeVisible()
  await expect(page.getByText('Clear game caches', { exact: true }).first()).toBeVisible()
  await openGameSelect(page)
  await page
    .getByRole('button', { name: 'Open Stardew Valley' })
    .click({ position: { x: 8, y: 8 } })
  await openGameSettings(page)
  await expect(page.getByRole('navigation', { name: 'Settings sections' })).toBeVisible()
  await expect(page.getByText('Game settings file', { exact: true })).toHaveCount(0)
  await expect(page.getByText('Clear game caches', { exact: true })).toHaveCount(0)
})

// The folder watcher's event reaches the page once per batch: the first NewDownloads call only sets the mark, the
// next offers what arrived since, and the toast's Add asks before recording the file against the opened page.
test('a pasted itch.io link opens the page, and the file saved from it asks before it is recorded', async ({
  page,
}) => {
  await page.route('**/wails/runtime', async (route) => {
    if ((route.request().postData() ?? '').includes(PAGE)) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: 'null' })
      return
    }
    await route.continue()
  })
  await openSims(page)
  await page.getByRole('button', { name: /^Switch profile/ }).click()
  await page.getByRole('menuitem', { name: 'From a link or file…' }).click()
  const dialog = page.getByRole('dialog', { name: 'Import a profile' })
  await dialog.getByLabel('Share link').fill(PAGE)
  await dialog.getByRole('button', { name: 'Preview' }).click()
  await expect(dialog.getByRole('status')).toContainText('Opened the itch.io page')
  await dialog.getByLabel('Close').click()

  const HourS = 3600
  const marked = page.waitForResponse(
    (r) =>
      r.url().endsWith('/wails/runtime') &&
      (r.request().postData() ?? '').includes(String(NEW_DOWNLOADS_ID)),
  )
  const old = writeZip(DOWNLOADS, 'Itch.Marker', { 'marker.package': 'm' })
  const long = Date.now() / 1000 - HourS
  utimesSync(old, long, long)
  await marked

  const saved = writeZip(DOWNLOADS, 'Itch.Saved', { 'saved.package': 's' })
  const soon = Date.now() / 1000 + HourS
  utimesSync(saved, soon, soon)
  const offer = page.getByRole('region', { name: 'Notifications' })
  await expect(offer.getByText(new RegExp(`^Add Itch.Saved.zip to ${PROFILE}\\?`))).toBeVisible({
    timeout: 15_000,
  })
  await offer.getByRole('button', { name: 'Add', exact: true }).last().click()
  const ask = page.getByRole('dialog', { name: 'Install Itch.Saved.zip as from someone/cool-mod?' })
  await expect(ask).toBeVisible()
  await ask.getByRole('button', { name: 'Yes' }).click()
  await page.getByRole('tab', { name: 'Mods' }).click()
  await expect(page.getByRole('button', { name: /^Details of saved\.package/ })).toBeVisible()
})
