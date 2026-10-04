import { execFileSync } from 'node:child_process'
import { rmSync } from 'node:fs'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const SCRIPT = fileURLToPath(new URL('../../scripts/selftest.sh', import.meta.url))
const ROOT = '/var/tmp/mortar-selftest-e2e'
const ENV = { ...process.env, MORTAR_SELFTEST_DIR: ROOT, MORTAR_SELFTEST_PORT: '9465' }

export function selftest(...args: string[]) {
  execFileSync(SCRIPT, args, { env: ENV, stdio: 'inherit' })
}

/** Specs mutate the sandbox (moved strays, added mods, trimmed history), so every run seeds a fresh one. */
export function freshSandbox() {
  selftest('stop')
  rmSync(ROOT, { recursive: true, force: true })
  selftest('start')
  selftest('seed')
}
