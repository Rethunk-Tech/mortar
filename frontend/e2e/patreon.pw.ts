import { execFileSync } from 'node:child_process'
import { mkdirSync, mkdtempSync, readFileSync, utimesSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { expect, type Page, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

// A pasted Patreon post opens in the browser and is remembered; the file the player then saves is added from the
// Downloads dialog under that post. The browser open is answered by the test, so the sandbox opens nothing.

const dir = process.env.MORTAR_E2E_DIR ?? ''
const POST = 'https://www.patreon.com/posts/cool-mod-4242'
const POST_URL = 'https://www.patreon.com/posts/4242'
const ARCHIVE = 'Seed.Patreon.zip'

/** Writes `<id>.zip` holding a content pack named like the id, and returns its path. */
/** Answers Yes to the question asked before a saved file is recorded against the post it came from. */
async function answerYes(page: Page, question: string) {
  const ask = page.getByRole('dialog', { name: question })
  await expect(ask).toBeVisible()
  await ask.getByRole('button', { name: 'Yes' }).click()
}

function writeArchive(into: string, id = 'Seed.Patreon') {
  const src = mkdtempSync(`${dir}/tmp/patreon-`)
  mkdirSync(`${src}/${id}`)
  writeFileSync(
    `${src}/${id}/manifest.json`,
    `{"Name":"${id.replace('.', ' ')}","Author":"Self-test","Version":"1.0.0","Description":"Fixture mod","UniqueID":"${id}","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}\n`,
  )
  mkdirSync(into, { recursive: true })
  execFileSync('python3', ['-m', 'zipfile', '-c', `${into}/${id}.zip`, `${src}/${id}`])
  return `${into}/${id}.zip`
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
  await answerYes(page, `Install ${ARCHIVE} as from Patreon post 4242?`)
  await page.getByRole('tab', { name: 'Mods' }).click()
  await expect(page.getByText('Seed Patreon', { exact: true })).toBeVisible()
  expect(calls.some((c) => c.includes(ARCHIVE) && c.includes('"4242"'))).toBe(true)
})

// The folder watcher's event reaches the page once per batch: the first NewDownloads call only sets the mark, the
// next offers what arrived since, and the toast's Add installs it under the Patreon post that was opened.
test('a file saved from a Patreon post is offered in a toast and installs under that post', async ({
  page,
}) => {
  const post = 'https://www.patreon.com/posts/toast-mod-7777'
  const postUrl = 'https://www.patreon.com/posts/7777'
  const downloads = `${dir}/home/.local/share/mortar/downloads`
  const calls: string[] = []
  await page.route('**/wails/runtime', async (route) => {
    const body = route.request().postData() ?? ''
    calls.push(body)
    if (body.includes(postUrl)) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: 'null' })
      return
    }
    await route.continue()
  })
  await openSeedFarm(page)
  await page.getByRole('button', { name: /^Switch profile/ }).click()
  await page.getByRole('menuitem', { name: 'From a link or file…' }).click()
  const dialog = page.getByRole('dialog', { name: 'Import a profile' })
  await dialog.getByLabel('Share link').fill(post)
  await dialog.getByRole('button', { name: 'Preview' }).click()
  await expect(dialog.getByRole('status')).toContainText('Opened the Patreon post')
  await dialog.getByLabel('Close').click()

  // An archive from long ago: its event makes the first call, which sets the mark and offers nothing.
  const HourS = 3600
  const marked = page.waitForResponse(
    (r) =>
      r.url().endsWith('/wails/runtime') &&
      (r.request().postData() ?? '').includes(String(NEW_DOWNLOADS_ID)),
  )
  const old = writeArchive(downloads, 'Seed.Marker')
  const long = Date.now() / 1000 - HourS
  utimesSync(old, long, long)
  await marked

  const saved = writeArchive(downloads, 'Seed.Toast')
  const soon = Date.now() / 1000 + HourS
  utimesSync(saved, soon, soon)
  const offer = page.getByRole('region', { name: 'Notifications' })
  await expect(offer.getByText('Add Seed.Toast.zip to Seed Farm?')).toBeVisible({ timeout: 15_000 })
  await offer.getByRole('button', { name: 'Add', exact: true }).last().click()
  await answerYes(page, 'Install Seed.Toast.zip as from Patreon post 7777?')
  await page.getByRole('tab', { name: 'Mods' }).click()
  await expect(page.getByText('Seed Toast', { exact: true })).toBeVisible()
  expect(calls.some((c) => c.includes('Seed.Toast.zip') && c.includes('"7777"'))).toBe(true)
})
