import { execFileSync } from 'node:child_process'
import { mkdirSync, readdirSync, readFileSync, readlinkSync, rmSync } from 'node:fs'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const SCRIPT = fileURLToPath(new URL('../../scripts/selftest.sh', import.meta.url))
// The sandbox holds copies of the games (GBs), so it lives on disk: /tmp is a tmpfs.
const BASE = '/var/tmp'
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

/** A sandbox folder is only ever named from a run's pid, so a path to delete is never read from anywhere. */
const sandboxDir = (runner: number) => `${BASE}/${PREFIX}${runner}`

function selftest(dir: string, ...args: string[]) {
  execFileSync(SCRIPT, args, {
    env: {
      ...process.env,
      MORTAR_SELFTEST_DIR: dir,
      MORTAR_SELFTEST_PORT: sandboxPort(),
      // play.pw.ts launches a stand-in for the game, which must not spend the session's real game launches.
      MORTAR_LAUNCH_SESSION: launchSession(dir),
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
function reap(runner: number) {
  const dir = sandboxDir(runner)
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
  try {
    selftest(dir, 'destroy')
  } catch {
    // A run that died before its first start has no marker for destroy to accept.
  }
  rmSync(dir, { recursive: true, force: true })
  rmSync(launchCounter(dir), { force: true })
  rmSync(`${launchCounter(dir)}.lock`, { force: true })
}

/** Sandboxes left by runs that died before their teardown; a folder named for this process is an older run's. */
function reapStale() {
  for (const name of readdirSync(BASE)) {
    const match = /^mortar-e2e-(\d+)$/.exec(name)
    const runner = match ? Number(match[1]) : 0
    if (runner > 0 && (runner === process.pid || !alive(runner))) {
      reap(runner)
    }
  }
}

/** Seeds this run's own sandbox, since specs mutate it, and returns the teardown that removes exactly that one. */
function freshSandbox(): () => void {
  reapStale()
  const dir = sandboxDir(process.pid)
  // Specs that run the sandbox's own binary (play.pw.ts) find it here; workers inherit the main process's env.
  process.env.MORTAR_E2E_DIR = dir
  // Chromium and the workers write their scratch (profile dirs a killed browser leaves behind) under the sandbox,
  // which teardown and reapStale remove whole.
  process.env.TMPDIR = `${dir}/tmp`
  mkdirSync(process.env.TMPDIR, { recursive: true })
  let tornDown = false
  const teardown = () => {
    if (tornDown) {
      return
    }
    tornDown = true
    try {
      selftest(dir, 'destroy')
    } catch {
      // A setup that failed before the sandbox was marked leaves destroy nothing it may delete.
    }
    rmSync(dir, { recursive: true, force: true })
    rmSync(launchCounter(dir), { force: true })
    rmSync(`${launchCounter(dir)}.lock`, { force: true })
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

export { freshSandbox, sandboxPort, selftest, serverEnv }
