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

test('the matrix fills the session cap of 3 launches with the base run and its two', () => {
  const matrix = readFileSync(join(import.meta.dir, 'regress-bepinex.sh'), 'utf8')
  const declared = Number(/^mx_launches\(\) \{ echo (\d+); \}$/m.exec(matrix)?.[1])
  const calls = matrix.split('\n').filter((l) => /mx_launch "/.test(l)).length
  expect(calls).toBe(declared)
  const base = 1
  expect(base + declared).toBe(3)
})

test('a regress that would need a fourth launch for an r2 code is refused up front, with the reason', () => {
  const refused = spawnSync(SCRIPT, ['regress', '--game', 'lethal-company'], {
    encoding: 'utf8',
    env: {
      ...process.env,
      MORTAR_REGRESS_MATRIX: '1',
      MORTAR_REGRESS_R2_CODE: 'not-uploaded',
      MORTAR_LAUNCH_SESSION: 'test-r2-refusal',
    },
  })
  expect(refused.status).toBe(3)
  expect(refused.stderr).toContain('MORTAR_REGRESS_R2_CODE needs launch 4, but a session allows 3')
})

const MATRIX = join(import.meta.dir, 'regress-bepinex.sh')

/** Runs a snippet with the matrix script's helpers sourced and its game-facing commands stubbed. */
function matrixShell(body: string) {
  return spawnSync('bash', ['-c', `source "${MATRIX}"; ${body}`], { encoding: 'utf8' })
}

test('the matrix waits for a profile file that arrives after the install returned, and gives up on one that never does', () => {
  const dir = mkdtempSync(join(tmpdir(), 'mx-wait-'))
  try {
    const late = join(dir, 'late.cfg')
    const shell = matrixShell(
      `(sleep 1; : >"${late}") & mx_wait_files 10 "${late}"; echo late=$?; mx_wait_files 1 "${join(dir, 'never')}"; echo never=$?`,
    )
    expect(shell.stdout.trim().split('\n')).toEqual(['late=0', 'never=1'])
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
})

test('the matrix reads the Console twice only once the first heartbeat is in', () => {
  const dir = mkdtempSync(join(tmpdir(), 'mx-console-'))
  try {
    const counter = join(dir, 'calls')
    writeFileSync(counter, '0')
    const shell = matrixShell(`
      sleep() { :; }
      mx_q() { echo "\\"$1\\""; }
      mx_wails() {
        local n=$(($(cat "${counter}") + 1))
        echo "$n" >"${counter}"
        if [ "$n" -lt 4 ]; then echo '[{"message":"loading"}]'; else echo '[{"message":"matrix heartbeat 1"}]'; fi
      }
      mx_base=p
      mx_console_reads
      echo "$mx_lines1 | $(cat "${counter}")"`)
    expect(shell.stdout.trim()).toBe('[{"message":"matrix heartbeat 1"}] | 5')
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
})

test('the matrix reports a bridge that survived every scene load, and one that did not or never said', () => {
  const dir = mkdtempSync(join(tmpdir(), 'mx-bridge-'))
  try {
    const log = (name: string, text: string) => {
      writeFileSync(join(dir, name), text)
      return join(dir, name)
    }
    const alive = log(
      'a.log',
      '[Info   :Mortar BepInEx Bridge] Bridge plugin alive after scene InitScene: True\n',
    )
    const dead = log(
      'b.log',
      '[Info   :Mortar BepInEx Bridge] Bridge plugin alive after scene InitScene: False\n',
    )
    const silent = log('c.log', '[Info   :BepInEx] Chainloader startup complete\n')
    matrixShell(
      `ROOT="${dir}"; mx_bridge_survived a "${alive}"; mx_bridge_survived b "${dead}"; mx_bridge_survived c "${silent}"`,
    )
    const rows = readFileSync(join(dir, 'matrix.tsv'), 'utf8')
      .trim()
      .split('\n')
      .map((l) => l.split('\t').slice(0, 2).join(' '))
    expect(rows).toEqual([
      'bridge.survived.a PASS',
      'bridge.survived.b FAIL',
      'bridge.survived.c FAIL',
    ])
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
})
