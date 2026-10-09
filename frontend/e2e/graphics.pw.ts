import { execFileSync } from 'node:child_process'
import { chmodSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { expect, type Page, test } from '@playwright/test'
import { openGameSelect } from './app.ts'
import { serverEnv } from './sandbox.ts'

// The sandbox holds a stand-in PEAK (selftest.sh fake-install), so nothing here downloads a loader: the profile gets
// the few files Mortar checks for, and a launch prefix stands in for the game, recording what it was started with.

const dir = process.env.MORTAR_E2E_DIR ?? ''
let env: Record<string, string> = {}

const cli = (...args: string[]) =>
  execFileSync(`${dir}/mortar-server`, args, { env, encoding: 'utf8' }).trim()

const FAKE_GAME = `#!/bin/bash
printf '%s\\n' "$@" >'${dir}/peak-args.txt'
exec -a "$1" sleep 2
`

let profile = ''
const runs = (): unknown[] => JSON.parse(cli('runs', 'peak', profile, '--json')) ?? []
const graphicsSetting = () =>
  (cli('settings', 'get', '--game', 'peak', 'graphicsApi').split('\n').pop() ?? '').split(
    /\s+/,
  )[1] ?? ''

test.beforeAll(() => {
  env = serverEnv(dir)
  profile = cli('profile', 'create', 'peak', 'Graphics').split('\t')[0] ?? ''
  const root = `${env.HOME}/.local/share/mortar/profiles/peak/${profile}`
  mkdirSync(`${root}/BepInEx/core`, { recursive: true })
  writeFileSync(`${root}/BepInEx/core/BepInEx.Preloader.dll`, 'stub')
  writeFileSync(`${root}/winhttp.dll`, 'stub')
  writeFileSync(`${root}/doorstop_config.ini`, 'stub')
  writeFileSync(`${root}/.mortar-bepinex.json`, '{"version":"5.4.75301","doorstop":4}')
  const fake = `${dir}/peak-fake-game.sh`
  writeFileSync(fake, FAKE_GAME)
  chmodSync(fake, 0o755)
  cli('profile', 'set', 'peak', profile, 'launchPrefix', fake)
  cli('profile', 'set', 'peak', profile, 'skipPlayCheck', 'true')
  cli('settings', 'set', '--game', 'peak', 'defaultLaunchMethod', 'direct')
})

/** The Play half of the sidebar's split button, which is named by its label only while the label shows. */
const playButton = (page: Page) =>
  page
    .getByRole('group')
    .filter({ has: page.getByRole('button', { name: 'More play options' }) })
    .getByRole('button')
    .first()

/** The stand-in game exits seconds after starting, which Mortar reports as a launch that failed; the report is closed. */
async function closeFailure(page: Page) {
  const failure = page.getByRole('dialog', { name: 'PEAK did not start' })
  await expect(failure).toBeVisible({ timeout: 15_000 })
  await failure.getByRole('button', { name: 'Close' }).click()
}

/** Opens PEAK's Graphics profile from a fresh start, past the welcome screen and the first-run tour. */
async function openPeak(page: Page) {
  await page.goto('/')
  const welcome = page.getByRole('button', { name: 'Continue' })
  const tabs = page.getByRole('tablist', { name: 'Profile sections' })
  const tile = page.getByRole('button', { name: 'Open PEAK' })
  await expect(welcome.or(tabs).or(tile).first()).toBeVisible({ timeout: 30_000 })
  if (await welcome.isVisible()) {
    await welcome.click()
  }
  const skip = page.getByRole('button', { name: 'Skip tour' })
  if (await skip.isVisible({ timeout: 1500 }).catch(() => false)) {
    await skip.click()
  }
  // Mortar reopens the last game, which an earlier spec may have left on another one.
  const onGraphics = page.getByRole('button', { name: /^Switch profile: Graphics/ })
  if (!(await onGraphics.isVisible())) {
    if (!(await tile.isVisible())) {
      await openGameSelect(page)
    }
    await tile.click({ position: { x: 8, y: 8 } })
    const card = page.getByRole('button', { name: /^(Open )?Graphics/ }).first()
    await expect(onGraphics.or(card).first()).toBeVisible()
    await expect(onGraphics)
      .toBeVisible({ timeout: 8000 })
      .catch(() => card.click())
  }
  await expect(playButton(page)).toBeVisible()
}

test('Play asks for a graphics API, saves the answer and launches with its arguments', async ({
  page,
}) => {
  rmSync(`${dir}/peak-args.txt`, { force: true })
  await openPeak(page)
  await playButton(page).click()
  const dialog = page.getByRole('dialog', { name: /Choose a graphics API for PEAK/ })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByRole('button', { name: 'Vulkan' })).toBeVisible()
  await expect(dialog.getByRole('button', { name: /^The BepInEx pack for PEAK/ })).toBeVisible()
  await expect(dialog.getByText('https://')).toHaveCount(0)
  await dialog.getByRole('button', { name: 'DirectX 12 (Recommended)' }).click()
  await expect.poll(graphicsSetting).toBe('dx12')
  await expect
    .poll(
      () =>
        existsSync(`${dir}/peak-args.txt`) ? readFileSync(`${dir}/peak-args.txt`, 'utf8') : '',
      { timeout: 15_000 },
    )
    .toContain('-dx12')
  await closeFailure(page)
})

test('a game that closes right after starting under Vulkan brings the question back', async ({
  page,
}) => {
  cli('settings', 'set', '--game', 'peak', 'graphicsApi', '')
  const before = runs().length
  await openPeak(page)
  await playButton(page).click()
  const dialog = page.getByRole('dialog', { name: /Choose a graphics API for PEAK/ })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText('The game closed right after starting last time.')).toHaveCount(0)
  await dialog.getByRole('button', { name: 'Vulkan' }).click()
  await expect.poll(graphicsSetting).toBe('vulkan')
  // The stand-in game exits after 2s, well inside the early-exit window.
  await closeFailure(page)
  expect(runs().length).toBe(before + 1)
  await playButton(page).click()
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText('The game closed right after starting last time.')).toBeVisible()
  await dialog.getByRole('button', { name: 'Cancel' }).click()
})
