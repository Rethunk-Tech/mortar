import { execFileSync } from 'node:child_process'
import { mkdirSync, mkdtempSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { expect, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

// A pasted Patreon post opens in the browser and is remembered; the file the player then saves is added from the
// Downloads dialog under that post. The browser open is answered by the test, so the sandbox opens nothing.

const dir = process.env.MORTAR_E2E_DIR ?? ''
const POST = 'https://www.patreon.com/posts/cool-mod-4242'
const POST_URL = 'https://www.patreon.com/posts/4242'
const ARCHIVE = 'Seed.Patreon.zip'

function writeArchive(into: string) {
  const src = mkdtempSync(`${dir}/tmp/patreon-`)
  mkdirSync(`${src}/Seed.Patreon`)
  writeFileSync(
    `${src}/Seed.Patreon/manifest.json`,
    '{"Name":"Seed Patreon","Author":"Self-test","Version":"1.0.0","Description":"Fixture mod","UniqueID":"Seed.Patreon","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}\n',
  )
  mkdirSync(into, { recursive: true })
  execFileSync('python3', ['-m', 'zipfile', '-c', `${into}/${ARCHIVE}`, `${src}/Seed.Patreon`])
}

test('a pasted Patreon post opens, then the saved file installs under that post', async ({
  page,
}) => {
  const calls: string[] = []
  await page.route('**/wails/runtime', async (route) => {
    const body = route.request().postData() ?? ''
    calls.push(body)
    if (body.includes(POST_URL)) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: 'null' })
      return
    }
    await route.continue()
  })
  await openSeedFarm(page)
  await page.getByRole('button', { name: /^Switch profile/ }).click()
  await page.getByRole('menuitem', { name: 'From a link or file…' }).click()
  const dialog = page.getByRole('dialog', { name: 'Import a profile' })
  await dialog.getByLabel('Share link').fill(POST)
  await dialog.getByRole('button', { name: 'Preview' }).click()
  await expect(dialog.getByRole('status')).toContainText('Opened the Patreon post')
  expect(calls.some((c) => c.includes(POST_URL))).toBe(true)

  writeArchive(`${dir}/home/.local/share/mortar/downloads`)
  await dialog.getByLabel('Close').click()
  await page.getByRole('button', { name: 'More ways to add mods' }).click()
  await page.getByRole('menuitem', { name: 'The downloads folder…' }).click()
  const downloads = page.getByRole('dialog', { name: 'From the downloads folder' })
  await downloads.getByRole('option', { name: new RegExp(ARCHIVE) }).click()
  await downloads.getByRole('button', { name: 'Add to profile' }).click()
  await page.getByRole('tab', { name: 'Mods' }).click()
  await expect(page.getByText('Seed Patreon', { exact: true })).toBeVisible()
  expect(calls.some((c) => c.includes(ARCHIVE) && c.includes('"4242"'))).toBe(true)
})
