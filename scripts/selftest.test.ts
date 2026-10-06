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

test('the matrix reads the Console first once a heartbeat is in, then again once the probe has logged its error and beaten since', () => {
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
        if [ "$n" -lt 4 ]; then echo '[{"message":"loading"}]'
        elif [ "$n" -lt 7 ]; then echo '[{"message":"matrix heartbeat 1"},{"message":"matrix error line"}]'
        else echo '[{"message":"matrix heartbeat 1"},{"message":"matrix error line"},{"message":"matrix heartbeat 2"}]'; fi
      }
      mx_base=p
      mx_console_reads
      echo "$mx_lines1 | $mx_lines2 | $(cat "${counter}")"
      echo "$mx_read1_at $mx_read2_at"`)
    const [reads, times] = shell.stdout.trim().split('\n')
    expect(reads).toBe(
      '[{"message":"matrix heartbeat 1"},{"message":"matrix error line"}] | ' +
        '[{"message":"matrix heartbeat 1"},{"message":"matrix error line"},{"message":"matrix heartbeat 2"}] | 7',
    )
    expect(times).toMatch(/^\d\d:\d\d:\d\d\.\d{3} \d\d:\d\d:\d\d\.\d{3}$/)
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
})

test('the matrix keeps the Unity logs of each launch beside its LogOutput.log, and Player-prev.log only when there is one', () => {
  const dir = mkdtempSync(join(tmpdir(), 'mx-logs-'))
  try {
    const saves = join(dir, 'saves')
    mkdirSync(saves)
    writeFileSync(join(dir, 'LogOutput.log'), 'bepinex a')
    writeFileSync(join(saves, 'Player.log'), 'unity a')
    writeFileSync(join(saves, 'Player-prev.log'), 'unity before a')
    const shell = matrixShell(`
      ROOT="${dir}" mx_log="${join(dir, 'LogOutput.log')}" mx_saves="${saves}"
      mx_keep_logs a
      rm "${join(saves, 'Player-prev.log')}"; echo "unity c" >"${join(saves, 'Player.log')}"
      mx_keep_logs c; echo "status=$?"`)
    expect(shell.stdout.trim()).toBe('status=0')
    expect(readFileSync(join(dir, 'LogOutput-a.log'), 'utf8')).toBe('bepinex a')
    expect(readFileSync(join(dir, 'Player-a.log'), 'utf8')).toBe('unity a')
    expect(readFileSync(join(dir, 'Player-prev-a.log'), 'utf8')).toBe('unity before a')
    expect(readFileSync(join(dir, 'Player-c.log'), 'utf8')).toBe('unity c\n')
    expect(existsSync(join(dir, 'Player-prev-c.log'))).toBe(false)
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
})

// The sandbox starts the game under bwrap --unshare-net (scripts/launch-guard.sh); a host without unprivileged user
// namespaces cannot stand one up, and the bridge test needs one.
const netns = spawnSync('bwrap', ['--dev-bind', '/', '/', '--unshare-net', 'true']).status === 0

test.skipIf(!netns)(
  "the matrix asks the bridge at the main menu, from inside the game's network namespace, and only notes whether its component survived",
  () => {
    const dir = mkdtempSync(join(tmpdir(), 'mx-bridge-'))
    try {
      const log = join(dir, 'LogOutput.log')
      writeFileSync(
        log,
        '[Info   :Mortar BepInEx Bridge] Bridge plugin alive after scene InitSceneLaunchOptions: False\n' +
          '[Info   :Mortar BepInEx Bridge] Bridge plugin alive after scene MainMenu: False\n',
      )
      const state = join(
        dir,
        'profiles/lethal-company/up/BepInEx/config/mortar-bepinex-bridge.json',
      )
      mkdirSync(join(dir, 'profiles/lethal-company/up/BepInEx/config'), { recursive: true })
      // A one-shot stand-in for the bridge, on a loopback of its own as the game is: it writes the state file once
      // listening and answers one status query.
      const fake = `import json, os, socket, sys
srv = socket.socket(); srv.bind(("127.0.0.1", 0)); srv.listen(1)
json.dump({"port": srv.getsockname()[1], "token": "t", "pid": os.getpid()}, open(sys.argv[1], "w"))
c, _ = srv.accept(); f = c.makefile("rw")
ok = f.readline().strip() == "t" and f.readline().strip() == "status"
f.write('ok {"gameVersion":"v81","scene":"MainMenu","plugins":[]}\\n' if ok else "error: unauthorized\\n"); f.flush()`
      writeFileSync(join(dir, 'fake.py'), fake)
      matrixShell(
        `ROOT="${dir}"; mx_data="${dir}"; mx_log="${log}"
      mx_game_pids() { python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["pid"])' "${state}"; }
      bwrap --dev-bind / / --unshare-net -- python3 "${dir}/fake.py" "${state}" & until [ -s "${state}" ]; do sleep 0.1; done
      mx_bridge_reachable up up; mx_bridge_reachable down down; mx_bridge_survived up "${log}"; wait`,
      )
      const rows = readFileSync(join(dir, 'matrix.tsv'), 'utf8')
        .trim()
        .split('\n')
        .map((l) => l.split('\t'))
      expect(rows.map((r) => `${r[0]} ${r[1]}`)).toEqual([
        'bridge.reachable.up PASS',
        'bridge.reachable.down FAIL',
        'bridge.survived.up INFO',
      ])
      expect(rows[2]?.[2]).toContain('alive after 0 scene load(s) and destroyed after 2')
    } finally {
      rmSync(dir, { recursive: true, force: true })
    }
  },
)
