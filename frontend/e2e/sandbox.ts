import { execFileSync } from 'node:child_process'
import { rmSync } from 'node:fs'
import { basename, join } from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const SCRIPT = fileURLToPath(new URL('../../scripts/selftest.sh', import.meta.url))
const PREFIX = 'mortar-e2e-'
const FREE_PORT = `const s = require('node:net').createServer().listen(0, '127.0.0.1', () => { console.log(s.address().port); s.close() })`

/** This run's sandbox folder and port, chosen once in the main process; Playwright's workers inherit them. */
export function sandboxEnv(): { dir: string; port: string } {
  process.env.MORTAR_E2E_DIR ??= join(process.env.TMPDIR ?? '/var/tmp', `${PREFIX}${process.pid}`)
  process.env.MORTAR_E2E_PORT ??= execFileSync(process.execPath, ['-e', FREE_PORT], {
    encoding: 'utf8',
  }).trim()
  return { dir: process.env.MORTAR_E2E_DIR, port: process.env.MORTAR_E2E_PORT }
}

export function selftest(...args: string[]) {
  const { dir, port } = sandboxEnv()
  execFileSync(SCRIPT, args, {
    env: { ...process.env, MORTAR_SELFTEST_DIR: dir, MORTAR_SELFTEST_PORT: port },
    stdio: 'inherit',
  })
}

/** Each run seeds its own sandbox, since specs mutate it (moved strays, added mods, trimmed history). */
export function freshSandbox() {
  selftest('start')
  selftest('seed')
}

/** Stops this run's server (selftest.sh checks the port's listener is its own binary) and removes only its folder. */
export function removeSandbox() {
  selftest('stop')
  const { dir } = sandboxEnv()
  if (basename(dir).startsWith(PREFIX)) {
    rmSync(dir, { recursive: true, force: true })
  }
}
