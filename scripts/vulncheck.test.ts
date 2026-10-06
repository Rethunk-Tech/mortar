import { expect, test } from 'bun:test'
import {
  chmodSync,
  cpSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

function run(root: string, env: Record<string, string>) {
  const out = Bun.spawnSync(['bash', join(root, 'scripts/vulncheck.sh')], {
    env: { PATH: `${join(root, 'bin')}:${process.env.PATH}`, ...env },
  })
  return { code: out.exitCode, text: out.stdout.toString() }
}

// A repo with go.mod, go.sum and a govulncheck that records each call and fails when told to.
function repo(): string {
  const root = mkdtempSync(join(tmpdir(), 'vulncheck-'))
  mkdirSync(join(root, 'scripts'))
  mkdirSync(join(root, 'bin'))
  cpSync(join(import.meta.dir, 'vulncheck.sh'), join(root, 'scripts/vulncheck.sh'))
  writeFileSync(join(root, 'go.mod'), 'a')
  writeFileSync(join(root, 'go.sum'), 'b')
  writeFileSync(
    join(root, 'bin/govulncheck'),
    `#!/bin/sh\necho ran >>"${root}/calls"\n[ -z "$FAIL_VULN" ]\n`,
  )
  chmodSync(join(root, 'bin/govulncheck'), 0o755)
  return root
}

const calls = (root: string) => {
  try {
    return readFileSync(join(root, 'calls'), 'utf8').trim().split('\n').length
  } catch {
    return 0
  }
}

test('govulncheck reruns only when go.mod or go.sum changed, CI or the force variable', () => {
  const root = repo()
  try {
    expect(run(root, {}).code).toBe(0)
    expect(calls(root)).toBe(1)
    expect(run(root, {}).text).toContain('govulncheck skipped')
    expect(calls(root)).toBe(1)
    run(root, { CI: 'true' })
    run(root, { MORTAR_GATE_VULNCHECK: '1' })
    expect(calls(root)).toBe(3)
    writeFileSync(join(root, 'go.sum'), 'changed')
    run(root, {})
    expect(calls(root)).toBe(4)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})

test('a failing govulncheck leaves no stamp, so the next run checks again', () => {
  const root = repo()
  try {
    expect(run(root, { FAIL_VULN: '1' }).code).toBe(1)
    expect(run(root, {}).code).toBe(0)
    expect(calls(root)).toBe(2)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})
