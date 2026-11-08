import { execFileSync } from 'node:child_process'
import { mkdirSync, readdirSync, readFileSync, readlinkSync, rmSync } from 'node:fs'
import process from 'node:process'
import { fileURLToPath } from 'node:url'
import { keepEvidence, type ServerState, shouldKeepEvidence } from './evidence.ts'

const SCRIPT = fileURLToPath(new URL('../../scripts/selftest.sh', import.meta.url))
const RM_SANDBOX = fileURLToPath(new URL('../../scripts/rm-sandbox.sh', import.meta.url))

/** Deletes a sandbox folder without deleting the user's files through the desktop mounts its hidden display leaves there. */
const removeSandbox = (dir: string) => execFileSync(RM_SANDBOX, [dir], { stdio: 'inherit' })
// The sandbox holds copies of the games (GBs), so it lives on disk: /tmp is a tmpfs.
const BASE = '/var/tmp'
/** Catalog ids the sandbox enables for itself (comma separated), as scripts/selftest.sh reads them. */
const SANDBOX_GAMES = 'lethal-company,sims4'
const PREFIX = 'mortar-e2e-'
const FREE_PORT = `const s = require('node:net').createServer().listen(0, '127.0.0.1', () => { console.log(s.address().port); s.close() })`
const STOP_WAIT_MS = 10_000
const STOP_POLL_MS = 100

/** This run's port, chosen once in the main process; Playwright's workers inherit it. */
function sandboxPort(): string {
  process.env.MORTAR_E2E_PORT ??= execFileSync(process.execPath, ['-e', FREE_PORT], {
    encoding: 'utf8',
  }).trim()
  return process.env.MORTAR_E2E_PORT
}

/** The launch-cap session of a run's sandbox, and its counter files under the cap folder selftest.sh keeps. */
const launchSession = (dir: string) => `e2e-${dir.slice(dir.lastIndexOf('-') + 1)}`
const launchCounter = (dir: string) => `${BASE}/mortar-launch-cap/${launchSession(dir)}`

/** Forgets the game launches a removed sandbox spent, so its session key starts clean. */
function dropLaunchCounter(dir: string) {
  rmSync(launchCounter(dir), { force: true })
  rmSync(`${launchCounter(dir)}.lock`, { force: true })
}

/** A sandbox folder is only ever named from a run's pid, so a path to delete is never read from anywhere. */
const sandboxDir = (runner: number, prefix = PREFIX) => `${BASE}/${prefix}${runner}`

function selftest(dir: string, ...args: string[]) {
  execFileSync(SCRIPT, args, {
    env: {
      ...process.env,
      MORTAR_SELFTEST_DIR: dir,
      MORTAR_SELFTEST_PORT: sandboxPort(),
      // play.pw.ts launches a stand-in for the game, which must not spend the session's real game launches.
      MORTAR_LAUNCH_SESSION: launchSession(dir),
      MORTAR_SELFTEST_LAUNCH_CAP: '12',
      // The Sims 4 is disabled in the shipped catalog; the sandbox enables it for itself beside the stand-in install.
      MORTAR_SELFTEST_ENABLE: SANDBOX_GAMES,
    },
    stdio: 'inherit',
  })
}

function alive(pid: number): boolean {
  try {
    process.kill(pid, 0)
    return true
  } catch {
    return false
  }
}

function exeOf(pid: number): string {
  try {
    return readlinkSync(`/proc/${pid}/exe`).replace(/ \(deleted\)$/, '')
  } catch {
    return ''
  }
}

/** Stops a dead run's server, when the pid it recorded still runs that sandbox's own binary, then removes it. */
function reap(runner: number, prefix: string) {
  const dir = sandboxDir(runner, prefix)
  let server = 0
  try {
    server = Number.parseInt(readFileSync(`${dir}/server.pid`, 'utf8'), 10)
  } catch {
    // No server ever started there.
  }
  if (server > 0 && exeOf(server) === `${dir}/mortar-server`) {
    process.kill(server, 'SIGTERM')
    const end = Date.now() + STOP_WAIT_MS
    while (alive(server) && Date.now() < end) {
      execFileSync('sleep', [String(STOP_POLL_MS / 1000)])
    }
    if (alive(server)) {
      throw new Error(`the stale e2e server ${server} in ${dir} did not stop`)
    }
  }
  // Its hidden display and any game it recorded outlive the server; destroy stops them with the folder.
  destroySandbox(dir)
  dropLaunchCounter(dir)
}

/** Sandboxes named `prefix` + a run's pid, left by runs that died before their teardown; a folder named for this
 * process is an older run's. */
function reapStale(prefix = PREFIX) {
  for (const name of readdirSync(BASE)) {
    const match = name.startsWith(prefix) ? /^(\d+)$/.exec(name.slice(prefix.length)) : null
    const runner = match ? Number(match[1]) : 0
    if (runner > 0 && (runner === process.pid || !alive(runner))) {
      reap(runner, prefix)
    }
  }
}

/** Stops what runs from a render or mem run's sandbox and removes it. */
function destroySandbox(dir: string) {
  try {
    selftest(dir, 'destroy')
  } catch {
    // A start that failed before the marker leaves destroy nothing to delete.
  }
  removeSandbox(dir)
}

/** Readies a render or mem run's sandbox folder named `prefix` + this pid: sweeps the folders of runs that were killed,
 * and points the run's TMPDIR (so Chromium's profile dirs) into the folder, which teardown removes whole. */
function prepareRunSandbox(prefix: string): string {
  reapStale(prefix)
  const dir = sandboxDir(process.pid, prefix)
  process.env.TMPDIR = `${dir}/tmp`
  mkdirSync(process.env.TMPDIR, { recursive: true })
  return dir
}

/** Where Playwright keeps a failed test's artefacts; the server's logs go beside them. */
const EVIDENCE_DIR = fileURLToPath(new URL('../test-results/sandbox-server', import.meta.url))

/** The sandbox server's state, read before destroy removes it. */
function serverState(dir: string): ServerState {
  const server = Number.parseInt(readFileSyncOr(`${dir}/server.pid`), 10)
  const listening =
    execFileSync('ss', ['-ltn', `sport = :${sandboxPort()}`], { encoding: 'utf8' })
      .trim()
      .split('\n').length > 1
  return {
    failed: process.env.MORTAR_E2E_FAILED === '1',
    alive: server > 0 && exeOf(server) === `${dir}/mortar-server`,
    listening,
  }
}

const readFileSyncOr = (path: string): string => {
  try {
    return readFileSync(path, 'utf8')
  } catch {
    return ''
  }
}

/** Keeps the server's log, crash log and exit record in test-results when a test failed or the server stopped. */
function keepServerEvidence(dir: string) {
  try {
    const state = serverState(dir)
    if (shouldKeepEvidence(state)) {
      keepEvidence(dir, EVIDENCE_DIR, state)
    }
  } catch {
    // Evidence is best effort; teardown must still remove the sandbox.
  }
}

/** Keeps the server's logs and state in out on every run, so a run whose server stopped can be traced after teardown. */
function keepServerLogs(dir: string, out: string) {
  try {
    keepEvidence(dir, out, serverState(dir))
  } catch {
    // A start that failed early leaves no log to keep.
  }
}

/** Seeds this run's own sandbox, since specs mutate it, and returns the teardown that removes exactly that one. */
function freshSandbox(): () => void {
  const dir = prepareRunSandbox(PREFIX)
  // Specs that run the sandbox's own binary (play.pw.ts) find it here; workers inherit the main process's env.
  process.env.MORTAR_E2E_DIR = dir
  let tornDown = false
  const teardown = () => {
    if (tornDown) {
      return
    }
    tornDown = true
    keepServerEvidence(dir)
    destroySandbox(dir)
    dropLaunchCounter(dir)
  }
  // A run stopped by a signal (a timeout, Ctrl+C, a closed terminal) removes its sandbox, which stops the server and the
  // hidden display recorded in it, then lets the signal take its course so Playwright still stops its workers.
  for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP'] as const) {
    process.once(signal, () => {
      teardown()
      process.kill(process.pid, signal)
    })
  }
  try {
    // A stand-in PEAK, so graphics.pw.ts needs no download and no copy of a real game.
    selftest(dir, 'fake-install', 'peak')
    // A stand-in The Sims 4 (a placeholder TS4_x64.exe and the player's Documents tree), for sims4.pw.ts.
    selftest(dir, 'sims4-install')
    selftest(dir, 'start')
    selftest(dir, 'seed')
  } catch (e) {
    // Playwright runs no teardown after a failed setup.
    teardown()
    throw e
  }
  return teardown
}

/** The environment the shared server runs with, read from /proc, so the sandbox's own binary acts on its data. */
function serverEnv(dir: string): Record<string, string> {
  const pid = Number.parseInt(readFileSync(`${dir}/server.pid`, 'utf8'), 10)
  const env = Object.fromEntries(
    readFileSync(`/proc/${pid}/environ`, 'utf8')
      .split('\0')
      .filter((kv) => kv.includes('='))
      .map((kv) => [kv.slice(0, kv.indexOf('=')), kv.slice(kv.indexOf('=') + 1)]),
  )
  env.WAILS_SERVER_HOST = '127.0.0.1'
  env.WAILS_SERVER_PORT = sandboxPort()
  return env
}

/** A runner for the sandbox's own binary, trimmed. Both are read per call: a spec's env fills once its server runs. */
function sandboxCli(dir: () => string, env: () => Record<string, string>) {
  return (...args: string[]): string =>
    execFileSync(`${dir()}/mortar-server`, args, { env: env(), encoding: 'utf8' }).trim()
}

/** Creates a profile through the CLI and answers its id, the first column of the row `profile create` prints. */
const createProfile = (cli: (...args: string[]) => string, game: string, name: string) =>
  cli('profile', 'create', game, name).split('\t')[0] ?? ''

/** Mortar's data folder, as the sandbox server's environment places it. */
const dataDirOf = (env: Record<string, string | undefined>) =>
  `${env.XDG_DATA_HOME || `${env.HOME ?? ''}/.local/share`}/mortar`

export {
  createProfile,
  dataDirOf,
  destroySandbox,
  freshSandbox,
  keepServerLogs,
  prepareRunSandbox,
  sandboxCli,
  sandboxPort,
  selftest,
  serverEnv,
}
