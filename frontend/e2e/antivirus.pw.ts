import { execFileSync } from 'node:child_process'
import { chmodSync, mkdirSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { expect, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'
import { serverEnv } from './sandbox.ts'

// The Custom command scanner stands in for a real antivirus: a script that flags any file holding a marker, so the whole
// refusal, Install anyway and History flow runs with no antivirus installed.

const dir = process.env.MORTAR_E2E_DIR ?? ''
const MARKER = ['MORTAR', 'TEST', 'THREAT'].join('-')
const DETECTION = 'Test.Marker.Threat'
let env: Record<string, string> = {}
const cli = (...args: string[]) =>
  execFileSync(`${dir}/mortar-server`, args, { env, encoding: 'utf8' }).trim()

test.beforeAll(() => {
  env = serverEnv(dir)
  const scanner = `${dir}/av-marker.sh`
  writeFileSync(
    scanner,
    `#!/bin/sh\nif grep -rq '${MARKER}' "$1"; then echo ${DETECTION}; exit 1; fi\n`,
  )
  chmodSync(scanner, 0o755)
  cli('settings', 'set', 'antivirus', 'command')
  cli('settings', 'set', 'antivirusCommand', `sh ${scanner} {path}`)
  const downloads = `${cli('data', 'location').split('\n')[0]}/downloads`
  mkdirSync(downloads, { recursive: true })
  const stage = `${dir}/av-stage/FlaggedMod`
  mkdirSync(stage, { recursive: true })
  writeFileSync(
    `${stage}/manifest.json`,
    '{"Name":"Flagged Test","UniqueID":"Mortar.FlaggedTest","Version":"1.0.0","MinimumApiVersion":"4.0.0"}',
  )
  writeFileSync(`${stage}/marker.txt`, MARKER)
  execFileSync('zip', ['-qr', `${downloads}/flagged-test.zip`, 'FlaggedMod'], {
    cwd: `${dir}/av-stage`,
  })
})

// Later specs share the sandbox, so the scanner setting and the mod go back out.
test.afterAll(() => {
  cli('mods', 'remove', 'stardew', 'Seed Farm', 'Mortar.FlaggedTest')
  cli('settings', 'reset', 'antivirus')
  cli('settings', 'reset', 'antivirusCommand')
})

test('a flagged archive is refused, installs after Install anyway, and History records it', async ({
  page,
}) => {
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'More ways to add mods' }).click()
  await page.getByRole('menuitem', { name: 'The downloads folder…' }).click()
  const dialog = page.getByRole('dialog', { name: 'From the downloads folder' })
  await dialog.getByRole('option', { name: /flagged-test\.zip/ }).click()
  await dialog.getByRole('button', { name: 'Add to profile' }).click()

  await expect(
    page.getByText('The antivirus flagged it, so Mortar did not install it.'),
  ).toBeVisible()
  await page.getByRole('button', { name: 'Install anyway…' }).click()
  const confirm = page.getByRole('dialog')
  await expect(confirm.getByText(new RegExp(`reports ${DETECTION}`))).toBeVisible()
  await confirm.getByRole('button', { name: 'Install anyway' }).click()

  await page.getByRole('tab', { name: 'Mods' }).click()
  await expect(page.getByText('Flagged Test', { exact: true })).toBeVisible()
  expect(cli('profile', 'history', 'stardew', 'Seed Farm')).toContain(
    'although the antivirus flagged it',
  )
})
