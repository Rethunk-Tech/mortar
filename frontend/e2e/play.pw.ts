import { type ChildProcess, execFileSync, spawn } from 'node:child_process'
import { chmodSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { expect, type Page, test } from '@playwright/test'
import { selftest, serverEnv } from './sandbox.ts'

// A --play start runs its own server on the sandbox's port, so each test stops the shared one and starts its binary again
// after, with the environment it ran with (read from /proc), without the rebuild `selftest.sh start` does.

const dir = process.env.MORTAR_E2E_DIR ?? ''
// Filled once the shared server runs: test files load before the global setup starts it.
let env: Record<string, string> = {}

const EXIT_WAIT_MS = 20_000

// Stands in for SMAPI and the game: writes SMAPI's log as a start does, then runs under SMAPI's process name with
// the profile's arguments for a few seconds, so Mortar sees the game run and close.
const FAKE_GAME = `#!/bin/bash
log="$HOME/.config/StardewValley/ErrorLogs/SMAPI-latest.txt"
mkdir -p "$(dirname "$log")"
printf '[00:00:00 INFO  SMAPI] SMAPI 4.1.10 with Stardew Valley 1.6.15 on Unix\\n[00:00:01 INFO  SMAPI] Loaded 0 mods:\\n' >"$log"
exec -a StardewModdingAPI bash -c 'sleep "$0"; :' 3 "$@"
`

const cli = (...args: string[]) =>
  execFileSync(`${dir}/mortar-server`, args, { env, encoding: 'utf8' }).trim()

const profileId = (name: string) =>
  (
    cli('profiles', 'stardew')
      .split('\n')
      .find((line) => line.includes(name)) ?? ''
  ).split(/\s+/)[0] ?? ''

/** Starts the shared server again from its built binary, detached so init reaps it, and waits until it serves. */
async function restartShared(page: Page) {
  const pid = execFileSync('sh', ['-c', 'nohup ./mortar-server >>server.log 2>&1 & echo $!'], {
    cwd: dir,
    env,
    encoding: 'utf8',
  })
  writeFileSync(`${dir}/server.pid`, pid)
  await served(page)
}

/** Starts the sandbox's binary with args and resolves with its exit code. */
function mortar(args: string[]) {
  const child = spawn(`${dir}/mortar-server`, args, { cwd: dir, env, stdio: 'ignore' })
  return { child, exited: new Promise<number | null>((done) => child.on('exit', done)) }
}

/** Only a process this test started, by its own pid. */
function stop(child: ChildProcess) {
  if (child.exitCode === null && child.signalCode === null && child.pid) {
    process.kill(child.pid, 'SIGTERM')
  }
}

/** Runs `mortar` with a play request for the profile in place of the shared server, which is back up when this
 * returns. Steam shortcuts add --steam-session; desktop shortcuts do not. */
async function playing(
  page: Page,
  profile: string,
  check: (exited: Promise<number | null>) => Promise<void>,
  steam = true,
) {
  selftest(dir, 'stop')
  const { child, exited } = mortar([
    `--play=stardew/${profile}`,
    ...(steam ? ['--steam-session'] : []),
  ])
  try {
    await check(exited)
  } finally {
    stop(child)
    await restartShared(page)
  }
}

const within = <T>(p: Promise<T>) =>
  Promise.race([p, new Promise((r) => setTimeout(r, EXIT_WAIT_MS, 'still running'))])

async function served(page: Page) {
  await expect(async () => {
    await page.goto('/')
  }).toPass({ timeout: 15_000 })
}

// JSON, since the table adds a line under a run that has an error or a cause.
const runs = (): { outcome: string }[] => JSON.parse(cli('runs', 'stardew', fine, '--json')) ?? []

let fine = ''
let blocked = ''

test.beforeAll(() => {
  env = serverEnv(dir)
  const fake = `${dir}/fake-game.sh`
  writeFileSync(fake, FAKE_GAME)
  chmodSync(fake, 0o755)
  fine =
    profileId('Deck Play') || cli('profile', 'create', 'stardew', 'Deck Play').split('\t')[0] || ''
  blocked = profileId('Seed Farm')
  cli('profile', 'set', 'stardew', fine, 'launchPrefix', fake)
  cli('profile', 'set', 'stardew', fine, 'skipPlayCheck', 'true')
  cli('settings', 'set', '--game', 'stardew', 'defaultLaunchMethod', 'direct')
})

test('a Steam session with a blocked profile shows only the pre-Play prompt, and Cancel quits', async ({
  page,
}) => {
  await playing(page, blocked, async (exited) => {
    await served(page)
    const prompt = page.getByRole('dialog', { name: 'Before you play' })
    await expect(prompt).toBeVisible({ timeout: 15_000 })
    await expect(page.getByRole('button', { name: 'Mortar menu' })).toHaveCount(0)
    await prompt.getByRole('button', { name: 'Cancel' }).click()
    expect(await within(exited)).toBe(0)
  })
})

test('a Steam session with a blocked profile turns into a normal session when the user picks a fix', async ({
  page,
}) => {
  await playing(page, blocked, async (exited) => {
    await served(page)
    const prompt = page.getByRole('dialog', { name: 'Before you play' })
    await prompt.getByRole('button', { name: 'Open Problems' }).click()
    await expect(page.getByRole('button', { name: 'Mortar menu' })).toBeVisible()
    await expect(page.getByRole('tab', { name: /^Problems/ })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    expect(await Promise.race([exited, Promise.resolve('running')])).toBe('running')
  })
})

test('a Steam session with a fine profile launches the game and exits when it closes', async ({
  page,
}) => {
  const before = runs().length
  await playing(page, fine, async (exited) => {
    await served(page)
    await expect(page.getByRole('dialog')).toHaveCount(0)
    expect(await within(exited)).toBe(0)
  })
  const after = runs()
  expect(after.length).toBe(before + 1)
  expect(after[0]?.outcome).toBe('ran')
})

test('a desktop shortcut plays in the full window, where Cancel leaves Mortar open', async ({
  page,
}) => {
  await playing(
    page,
    blocked,
    async (exited) => {
      await served(page)
      const prompt = page.getByRole('dialog', { name: 'Before you play' })
      await expect(prompt).toBeVisible({ timeout: 15_000 })
      await prompt.getByRole('button', { name: 'Cancel' }).click()
      await expect(page.getByRole('button', { name: 'Mortar menu' })).toBeVisible()
      expect(await Promise.race([exited, Promise.resolve('running')])).toBe('running')
    },
    false,
  )
})

test('a Steam session sent to a running Mortar lasts until its game closes', async ({ page }) => {
  const before = runs().length
  await served(page)
  const { child, exited } = mortar([`--play=stardew/${fine}`, '--steam-session'])
  try {
    // The fake game runs for 3s, so a process that only forwarded the request would be gone by now.
    expect(await Promise.race([exited, new Promise((r) => setTimeout(r, 1500, 'running'))])).toBe(
      'running',
    )
    expect(await within(exited)).toBe(0)
  } finally {
    stop(child)
  }
  // The run's record lands just after the game goes idle, which is what the session waited for.
  await expect.poll(() => runs().length).toBe(before + 1)
})
