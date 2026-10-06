import { expect, test } from 'bun:test'
import { spawn, spawnSync } from 'node:child_process'
import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  utimesSync,
  writeFileSync,
} from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const SCRIPT = join(import.meta.dir, 'selftest.sh')
const DAY_AGO = new Date(Date.now() - 24 * 3600 * 1000)

function sandbox(base: string, name: string, marked: boolean): string {
  const dir = join(base, name)
  mkdirSync(join(dir, 'home'), { recursive: true })
  writeFileSync(join(dir, 'server.log'), 'x')
  const paths = [join(dir, 'server.log'), join(dir, 'home')]
  if (marked) {
    writeFileSync(join(dir, '.mortar-selftest'), '')
    paths.push(join(dir, '.mortar-selftest'))
  }
  for (const p of [...paths, dir]) {
    utimesSync(p, DAY_AGO, DAY_AGO)
  }
  return dir
}

function run(base: string, root: string, ...args: string[]) {
  return spawnSync(SCRIPT, args, {
    encoding: 'utf8',
    env: {
      ...process.env,
      MORTAR_SELFTEST_BASE: base,
      MORTAR_SELFTEST_DIR: root,
      MORTAR_SELFTEST_PORT: '1',
    },
  })
}

test('reap deletes only idle marked sandboxes, and only with --yes', () => {
  const base = mkdtempSync(join(tmpdir(), 'mortar-reap-test-'))
  const idle = sandbox(base, 'idle', true)
  const unmarked = sandbox(base, 'unmarked', false)
  const live = sandbox(base, 'live', true)
  const recent = join(base, 'recent')
  mkdirSync(recent)
  writeFileSync(join(recent, '.mortar-selftest'), '')
  const holder = spawn('sleep', ['30'], { cwd: live, stdio: 'ignore' })
  try {
    const dry = run(base, idle, 'reap')
    expect(dry.status).toBe(0)
    expect(dry.stdout).toMatch(new RegExp(`idle .*\\t${idle}\\n`))
    expect(dry.stdout).toContain(`live     ${live}`)
    expect(dry.stdout).toContain(`recent   ${recent}`)
    expect(dry.stdout).not.toContain(unmarked)
    expect(existsSync(idle)).toBe(true)

    expect(run(base, idle, 'reap', '--yes').status).toBe(0)
    expect(existsSync(idle)).toBe(false)
    expect(existsSync(unmarked)).toBe(true)
    expect(existsSync(live)).toBe(true)
    expect(existsSync(recent)).toBe(true)
  } finally {
    holder.kill()
    rmSync(base, { recursive: true, force: true })
  }
})

test('destroy refuses an unmarked folder and one outside the base', () => {
  const base = mkdtempSync(join(tmpdir(), 'mortar-destroy-test-'))
  const unmarked = sandbox(base, 'unmarked', false)
  const nested = sandbox(join(base, 'unmarked'), 'nested', true)
  const marked = sandbox(base, 'marked', true)
  try {
    expect(run(base, unmarked, 'destroy').status).toBe(1)
    expect(run(base, nested, 'destroy').status).toBe(1)
    expect(existsSync(nested)).toBe(true)
    const ok = run(base, marked, 'destroy')
    expect(ok.status).toBe(0)
    expect(existsSync(marked)).toBe(false)
  } finally {
    rmSync(base, { recursive: true, force: true })
  }
})

test("the matrix's declared launch count is its number of launches, which the session cap is checked against", () => {
  const matrix = readFileSync(join(import.meta.dir, 'regress-bepinex.sh'), 'utf8')
  const declared = Number(/^mx_launches\(\) \{ echo (\d+); \}$/m.exec(matrix)?.[1])
  const calls = matrix.split('\n').filter((l) => /mx_launch "/.test(l)).length
  expect(calls).toBe(declared)
})
