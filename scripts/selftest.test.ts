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
      expect(existsSync(join(dir, 'bridge-state-up.json'))).toBe(true)
      expect(existsSync(join(dir, 'bridge-state-down.json'))).toBe(false)
    } finally {
      rmSync(dir, { recursive: true, force: true })
    }
  },
)

test.skipIf(!netns)(
  "the relay lets Mortar's own bridge client reach a bridge on the game's loopback",
  () => {
    const dir = mkdtempSync(join(tmpdir(), 'mx-relay-'))
    try {
      const config = join(dir, 'profiles/lethal-company/p/BepInEx/config')
      mkdirSync(config, { recursive: true })
      const state = join(config, 'mortar-bepinex-bridge.json')
      const fake = `import json, os, socket, sys
srv = socket.socket(); srv.bind(("127.0.0.1", 0)); srv.listen(1)
json.dump({"port": srv.getsockname()[1], "token": "t", "pid": os.getpid()}, open(sys.argv[1], "w"))
c, _ = srv.accept(); f = c.makefile("rw")
token, command = f.readline().strip(), f.readline().strip()
f.write('ok {"measured":false}\\n' if (token, command) == ("t", "perf start") else "error: bad\\n"); f.flush()`
      writeFileSync(join(dir, 'fake.py'), fake)
      const shell = matrixShell(
        `ROOT="${dir}"; mx_data="${dir}"
      mx_game_pids() { python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["pid"])' "${state}"; }
      bwrap --dev-bind / / --unshare-net -- python3 "${dir}/fake.py" "${state}" & until [ -s "${state}" ]; do sleep 0.1; done
      mx_bridge_relay p || echo "no relay"
      python3 -c 'import json,socket,sys
st = json.load(open(sys.argv[1]))
s = socket.create_connection(("127.0.0.1", st["port"]), timeout=10); s.sendall(b"t\\nperf start\\n")
print(s.makefile().readline().strip())' "${state}"
      mx_relay_stop; wait`,
      )
      expect(shell.stdout.trim()).toBe('ok {"measured":false}')
    } finally {
      rmSync(dir, { recursive: true, force: true })
    }
  },
)

test('the measured launch rows read the startup report, Mortar naming its rows by package, and the live badges', () => {
  const dir = mkdtempSync(join(tmpdir(), 'mx-startup-'))
  try {
    const profile = join(dir, 'profiles/lethal-company/p')
    mkdirSync(join(profile, 'BepInEx/patchers/Rethunk-MortarBepInExBridge'), { recursive: true })
    mkdirSync(join(profile, 'startup'), { recursive: true })
    writeFileSync(
      join(profile, 'BepInEx/patchers/Rethunk-MortarBepInExBridge/MortarBepInExBridge.Patcher.dll'),
      '',
    )
    writeFileSync(join(dir, 'measure-requested'), '')
    const asked = new Date(Date.now() - 120_000)
    utimesSync(join(dir, 'measure-requested'), asked, asked)
    const log = join(dir, 'LogOutput.log')
    writeFileSync(
      log,
      "[Info   :Mortar Startup] Timing this launch's plugins until scene MainMenu.\n" +
        '[Info   :   BepInEx] Loading [A 1.0.0]\n[Info   :   BepInEx] Loading [B 1.0.0]\n',
    )
    const start = new Date(Date.now() - 60_000).toISOString()
    writeFileSync(
      join(profile, 'startup/20261006T180000Z.json'),
      JSON.stringify({
        processStart: start,
        phases: {
          bridgeEntry: 900,
          entryDone: 3000,
          gameLaunched: 3800,
          titleMenu: 5000,
          titleScreen: 9000,
        },
        entryMissed: 0,
        mods: [{ id: 'a' }, { id: 'b' }],
      }),
    )
    writeFileSync(
      join(dir, 'events-a.jsonl'),
      `${JSON.stringify({
        name: 'launch:live',
        data: {
          scene: 'MainMenu',
          mods: [
            { id: 'thunderstore:MortarMatrix-ProbeDupA', loaded: true },
            { id: 'thunderstore:MortarMatrix-ProbeDupB', loaded: false },
          ],
        },
      })}\n`,
    )
    writeFileSync(
      join(dir, 'mods-a.json'),
      JSON.stringify([
        { id: 'thunderstore:MortarMatrix-ProbeDupA', enabled: true },
        { id: 'thunderstore:MortarMatrix-ProbeDupB', enabled: true },
      ]),
    )
    matrixShell(
      `ROOT="${dir}"; mx_data="${dir}"
    mx_q() { echo "\\"$1\\""; }
    mx_wails() { echo '[{"mods":[{"id":"thunderstore:A-A"},{"id":"bepinex:loose"}]}]'; }
    mx_startup_rows p "${log}"; mx_live_rows p`,
    )
    const rows = readFileSync(join(dir, 'matrix.tsv'), 'utf8')
      .trim()
      .split('\n')
      .map((l) => l.split('\t'))
    expect(rows.map((r) => `${r[0]} ${r[1]}`)).toEqual([
      'startup.patcher PASS',
      'startup.report PASS',
      'startup.mortar FAIL',
      'live.scene PASS',
      'live.badges PASS',
    ])
    expect(rows[2]?.[2]).toContain('bepinex:loose')
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
})

test('the matrix reads the load order as plugin rows and checks it covers what launch (a) loaded', () => {
  const dir = mkdtempSync(join(tmpdir(), 'mx-order-'))
  try {
    const order = join(dir, 'order.json')
    const log = join(dir, 'LogOutput.log')
    writeFileSync(
      order,
      JSON.stringify([
        { position: 1, id: 'bepinex:a.lib', name: 'Lib', dependents: ['bepinex:b.mod'] },
        { position: 2, id: 'bepinex:b.mod', name: 'Mod', required: ['bepinex:a.lib'] },
      ]),
    )
    writeFileSync(
      log,
      [
        '[Info   :   BepInEx] Loading [Mortar BepInEx Bridge 0.1.0]',
        '[Info   :   BepInEx] Loading [Lib 1.0.0]',
        '[Info   :   BepInEx] Loading [Mod 2.1]',
        '[Info   :   BepInEx] Loading [Matrix base 1.0.0]',
        '[Info   :   BepInEx] Loading [Stray Thing 3.0.0]',
        '',
      ].join('\n'),
    )
    const shell = matrixShell(
      `echo "[$(mx_order_problems "${order}")]"; echo "[$(mx_order_unlisted "${order}" "${log}")]"`,
    )
    expect(shell.stdout.split('\n').slice(0, 2)).toEqual(['[]', '[Stray Thing]'])
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
})
