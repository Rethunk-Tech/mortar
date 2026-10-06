import { execFileSync } from 'node:child_process'
import { readdirSync, readFileSync, readlinkSync, rmSync } from 'node:fs'
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

/** A sandbox folder is only ever named from a run's pid, so a path to delete is never read from anywhere. */
const sandboxDir = (runner: number) => `${BASE}/${PREFIX}${runner}`

function selftest(dir: string, ...args: string[]) {
  execFileSync(SCRIPT, args, {
    env: { ...process.env, MORTAR_SELFTEST_DIR: dir, MORTAR_SELFTEST_PORT: sandboxPort() },
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
  rmSync(dir, { recursive: true, force: true })
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
  const teardown = () => {
    selftest(dir, 'stop')
    rmSync(dir, { recursive: true, force: true })
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

export { freshSandbox, sandboxPort, selftest }
