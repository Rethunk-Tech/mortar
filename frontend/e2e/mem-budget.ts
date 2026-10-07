import { execFileSync, spawnSync } from 'node:child_process'
import { mkdirSync, readdirSync, readFileSync, readlinkSync, rmSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { chromium, type Page } from '@playwright/test'
import { sandboxPort, selftest, serverEnv } from './sandbox.ts'

// Measures the memory Mortar holds while it does its heaviest work: peak PSS of the Go server's process tree and of the
// browser's, per scenario. The WebView exists only in the desktop build, so in server mode headless Chromium (Playwright)
// stands in for it. PSS divides shared pages between the processes sharing them, so the sum counts each page once.
//
//   MORTAR_MEM_DATA=<a Mortar data folder>   copy of the data to measure on (else the seeded sandbox, a small smoke run)
//   MORTAR_MEM_ONLY=browse,problems          run only those scenarios (browse, install, problems, launch)
//   MORTAR_MEM_PROFILE=<id or name>          the Stardew profile (else the one with the most mods)
//   MORTAR_MEM_BUDGET_MIB=1024               the limit for Go and browser together, per scenario
// Exits 1 when a scenario's total is over the budget.

const BASE = '/var/tmp'
const MIB = 1024
const SAMPLE_MS = 100
const BUDGET_MIB = Number(process.env.MORTAR_MEM_BUDGET_MIB ?? '1024')
const ZIP_MIB = Number(process.env.MORTAR_MEM_ZIP_MIB ?? '500')
const ZIP_FILE_KIB = 100
const BROWSE_QUERY = process.env.MORTAR_MEM_QUERY ?? 'content'
const SCROLLS = 5
const SCROLL_STEP_PX = 4000
const SCROLL_WAIT_MS = 1200
const SETTLE_MS = 3000
const LAUNCH_TIMEOUT_MS = 240_000

const dir = `${BASE}/mortar-mem-${process.pid}`

interface Peak {
  go: number
  browser: number
  game: number
  total: number
  /** The five biggest processes at the total's peak. */
  top: { pid: number; comm: string; mib: number }[]
}

/** pid -> parent pid of every process, from /proc. */
function parents(): Map<number, number> {
  const map = new Map<number, number>()
  for (const name of readdirSync('/proc')) {
    try {
      const stat = readFileSync(`/proc/${name}/stat`, 'utf8')
      // The command name may hold spaces and parentheses, so the fields start after the last one.
      const rest = stat.slice(stat.lastIndexOf(')') + 2).split(' ')
      map.set(Number(name), Number(rest[1]))
    } catch {
      // Not a process, or it ended while the walk ran.
    }
  }
  return map
}

function tree(root: number): number[] {
  const kids = parents()
  const out = [root]
  for (let i = 0; i < out.length; i++) {
    for (const [pid, ppid] of kids) {
      if (ppid === out[i]) {
        out.push(pid)
      }
    }
  }
  return out
}

/** KiB of PSS (RSS where the kernel gives no PSS) of one process, 0 once it is gone. */
function pssKiB(pid: number): number {
  try {
    const text = readFileSync(`/proc/${pid}/smaps_rollup`, 'utf8')
    const pss = /^Pss:\s+(\d+) kB/m.exec(text)
    return pss ? Number(pss[1]) : 0
  } catch {
    return 0
  }
}

/** The browser's main process: the one of our descendants that is Chromium and has no --type (its helpers do). Playwright
 * no longer hands out the pid. */
function browserRoot(): number {
  for (const pid of tree(process.pid)) {
    try {
      const args = readFileSync(`/proc/${pid}/cmdline`, 'utf8').split('\0')
      if (
        /chrom|headless_shell/i.test(args[0] ?? '') &&
        !args.some((a) => a.startsWith('--type='))
      ) {
        return pid
      }
    } catch {
      // Gone already.
    }
  }
  throw new Error('no Chromium process under this one')
}

function serverPid(): number {
  const out = execFileSync('ss', ['-ltnp', `sport = :${sandboxPort()}`], { encoding: 'utf8' })
  const m = /pid=(\d+)/.exec(out)
  if (!m) {
    throw new Error('the sandbox server is not listening')
  }
  return Number(m[1])
}

const exeOf = (pid: number) => {
  try {
    return readlinkSync(`/proc/${pid}/exe`).replace(/ \(deleted\)$/, '')
  } catch {
    return ''
  }
}

const commOf = (pid: number) => {
  try {
    return readFileSync(`/proc/${pid}/comm`, 'utf8').trim()
  } catch {
    return '?'
  }
}

interface Proc {
  pid: number
  comm: string
  mib: number
}

/** Samples until the returned function is called, which answers the peaks. Go is the sandbox's mortar-server process
 * alone: the tree around it holds the sandbox's display, D-Bus and reaper, which are not Mortar. A game it launches is
 * reported apart and left out of the total. */
function sample(listener: () => number, browser: () => number): () => Peak {
  const peak: Peak = { go: 0, browser: 0, game: 0, total: 0, top: [] }
  const server = `${dir}/mortar-server`
  const timer = setInterval(() => {
    const all = tree(listener())
    const goPids = all.filter((pid) => exeOf(pid) === server)
    const gamePids = new Set(
      goPids.flatMap((root) => tree(root)).filter((pid) => !goPids.includes(pid)),
    )
    const mem = (pids: Iterable<number>): Proc[] =>
      [...pids].map((pid) => ({ pid, comm: commOf(pid), mib: pssKiB(pid) / MIB }))
    const goProcs = mem(goPids)
    const browserProcs = mem(tree(browser()))
    const gameProcs = mem(gamePids)
    const sum = (procs: Proc[]) => procs.reduce((t, p) => t + p.mib, 0)
    const go = sum(goProcs)
    const web = sum(browserProcs)
    peak.go = Math.max(peak.go, go)
    peak.browser = Math.max(peak.browser, web)
    peak.game = Math.max(peak.game, sum(gameProcs))
    if (go + web > peak.total) {
      peak.total = go + web
      peak.top = [...goProcs, ...browserProcs, ...gameProcs]
        .sort((x, y) => y.mib - x.mib)
        .slice(0, 5)
    }
  }, SAMPLE_MS)
  return () => {
    clearInterval(timer)
    return peak
  }
}

const outDir = process.env.MORTAR_MEM_OUT ?? `${BASE}/mem-budget-${Date.now()}`

/** With MORTAR_PPROF=<addr> (the server serves its pprof endpoints there), saves the server's heap and allocation profiles
 * after a scenario and says where, so a figure over budget can be traced to the code that holds it. */
async function saveProfiles(addr: string, scenario: string): Promise<string> {
  mkdirSync(outDir, { recursive: true })
  const files: string[] = []
  for (const kind of ['heap', 'allocs']) {
    const res = await fetch(`http://${addr}/debug/pprof/${kind}`)
    const file = `${outDir}/${scenario}.${kind}.pb.gz`
    writeFileSync(file, Buffer.from(await res.arrayBuffer()))
    files.push(file)
  }
  return `profiles: ${files.join(' ')}`
}

const sleep = (ms: number) => new Promise((done) => setTimeout(done, ms))

let env: Record<string, string> = {}
const cli = (...args: string[]) =>
  spawnSync(`${dir}/mortar-server`, args, {
    env,
    encoding: 'utf8',
    maxBuffer: 1 << 28,
    timeout: LAUNCH_TIMEOUT_MS,
  })

function pickProfile(): Profile {
  const rows = cli('profiles', 'stardew')
    .stdout.split('\n')
    .filter(Boolean)
    .map((line) => {
      // "<id>  <name>  <enabled>/<mods>  <updated>": the name ends where the counts start.
      const [id = '', ...rest] = line.split(/\s+/)
      return {
        id,
        name: rest
          .join(' ')
          .replace(/\s+\d+\/\d+.*$/, '')
          .trim(),
      }
    })
  const want = process.env.MORTAR_MEM_PROFILE
  const named = rows.find((r) => r.id === want || r.name === want)
  if (named) {
    return named
  }
  const mods = (r: { id: string }) => cli('mods', 'stardew', r.id).stdout.split('\n').length
  const best = rows.reduce((a, b) => (mods(b) > mods(a) ? b : a), rows[0] ?? { id: '', name: '' })
  if (!best.id) {
    throw new Error('the sandbox has no Stardew Valley profile')
  }
  return best
}

async function openProfile(page: Page, name: string) {
  await page.goto('/')
  const welcome = page.getByRole('button', { name: 'Continue' })
  const profile = page
    .getByRole('button', { name: new RegExp(`^(Open )?${escapeRegExp(name)}`) })
    .first()
  const tabs = page.getByRole('tablist', { name: 'Profile sections' })
  await welcome.or(profile).or(tabs).first().waitFor({ timeout: 30_000 })
  if (await welcome.isVisible()) {
    await welcome.click()
  }
  if (!(await profile.isVisible())) {
    await page.getByRole('button', { name: /^(Switch game|Choose a game)/ }).click()
    await page.getByRole('menuitem', { name: /^All games/ }).click()
  }
  const mods = page.getByRole('tab', { name: 'Mods' })
  // The tiles animate under the pointer, so a click can land between them; clicking again settles it.
  for (let attempt = 0; attempt < 3 && !(await mods.isVisible()); attempt++) {
    if (await profile.isVisible({ timeout: 5000 }).catch(() => false)) {
      await profile.click()
    }
    await mods.waitFor({ timeout: 8000 }).catch(() => undefined)
  }
  await mods.waitFor({ timeout: 1000 })
  const skip = page.getByRole('button', { name: 'Skip tour' })
  if (await skip.isVisible({ timeout: 1500 }).catch(() => false)) {
    await skip.click()
  }
}

const escapeRegExp = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

/** A zip of many files holding one SMAPI mod, in the sandbox's own scratch folder. */
function bigZip(): string {
  const path = `${dir}/tmp/MemBigMod.zip`
  mkdirSync(`${dir}/tmp`, { recursive: true })
  const py = `
import os, sys, zipfile
path, mib, kib = sys.argv[1], int(sys.argv[2]), int(sys.argv[3])
count = mib * 1024 // kib
with zipfile.ZipFile(path, "w", zipfile.ZIP_STORED) as z:
    z.writestr("MemBigMod/manifest.json", '{"Name":"Mem Big Mod","Author":"mem","Version":"1.0.0","Description":"memory budget fixture","UniqueID":"mem.BigMod","EntryDll":"MemBigMod.dll","MinimumApiVersion":"4.0.0"}')
    z.writestr("MemBigMod/MemBigMod.dll", b"MZ")
    for i in range(count):
        z.writestr("MemBigMod/assets/f%05d.bin" % i, os.urandom(kib * 1024))
`
  execFileSync('python3', ['-I', '-c', py, path, String(ZIP_MIB), String(ZIP_FILE_KIB)])
  return path
}

interface Profile {
  id: string
  name: string
}

interface Scenario {
  name: string
  run: (page: Page, profile: Profile) => Promise<void>
}

const scenarios: Scenario[] = [
  {
    name: 'browse',
    async run(page) {
      await page.getByRole('tab', { name: 'Browse' }).click()
      await page.getByRole('textbox').first().fill(BROWSE_QUERY)
      await sleep(SETTLE_MS)
      for (let n = 1; n <= SCROLLS; n++) {
        cli('browse', 'stardew', BROWSE_QUERY, '--source', 'all', '--page', String(n))
        await page.mouse.move(700, 500)
        await page.mouse.wheel(0, SCROLL_STEP_PX)
        await sleep(SCROLL_WAIT_MS)
      }
      await page.getByRole('tab', { name: 'Mods' }).click()
    },
  },
  {
    name: 'install',
    async run(page, profile) {
      const zip = bigZip()
      await page.getByRole('tab', { name: 'Mods' }).click()
      const r = cli('install', 'stardew', profile.id, zip)
      if (r.status !== 0) {
        throw new Error(`install failed: ${r.stderr || r.stdout}`)
      }
      await sleep(SETTLE_MS)
    },
  },
  {
    name: 'problems',
    async run(page, profile) {
      await page.getByRole('tab', { name: 'Problems' }).click()
      const r = cli('problems', 'stardew', profile.id, '--format', 'text')
      if (r.status !== 0 && r.status !== 3) {
        throw new Error(`problems failed: ${r.stderr || r.stdout}`)
      }
      await sleep(SETTLE_MS)
    },
  },
  {
    name: 'launch',
    async run(page, profile) {
      await page.getByRole('tab', { name: 'Mods' }).click()
      const r = cli('play', 'stardew', profile.id, '--test')
      if (r.status !== 0) {
        throw new Error(`launch failed: ${r.stderr || r.stdout}`)
      }
    },
  },
]

async function main(): Promise<number> {
  const only = (process.env.MORTAR_MEM_ONLY ?? '').split(',').filter(Boolean)
  const chosen = scenarios.filter((s) => only.length === 0 || only.includes(s.name))
  if (chosen.length === 0) {
    throw new Error(
      `MORTAR_MEM_ONLY names no scenario (${scenarios.map((s) => s.name).join(', ')})`,
    )
  }
  process.env.MORTAR_E2E_DIR = dir
  process.env.MORTAR_LAUNCH_SESSION = `mem-${process.pid}`
  mkdirSync(dir, { recursive: true })
  let code = 0
  const rows: string[] = []
  try {
    const data = process.env.MORTAR_MEM_DATA
    if (data) {
      process.env.MORTAR_SELFTEST_DATA = data
    }
    selftest(dir, 'start', ...(data ? ['--copy-data'] : []))
    if (!data) {
      selftest(dir, 'seed')
    }
    env = serverEnv(dir)
    const profile = pickProfile()
    const browser = await chromium.launch()
    const page = await browser.newPage({
      baseURL: `http://127.0.0.1:${sandboxPort()}`,
      viewport: { width: 1400, height: 840 },
    })
    const browserPid = browserRoot()
    await openProfile(page, profile.name).catch(async (e: unknown) => {
      await page.screenshot({ path: `${BASE}/mortar-mem-failure.png` })
      throw new Error(
        `could not open ${profile.name}; screenshot in ${BASE}/mortar-mem-failure.png: ${e}`,
      )
    })
    console.log(
      `Profile ${profile.name} (${profile.id}). Browser: headless Chromium via Playwright, a stand-in for the desktop WebView. PSS in MiB, budget ${BUDGET_MIB} MiB.`,
    )
    for (const scenario of chosen) {
      const stop = sample(serverPid, () => browserPid)
      let failure = ''
      try {
        await scenario.run(page, profile)
      } catch (e) {
        failure = e instanceof Error ? e.message : String(e)
      }
      const p = stop()
      const over = p.total > BUDGET_MIB
      if (over || failure) {
        code = 1
      }
      rows.push(
        `${scenario.name.padEnd(9)} ${p.go.toFixed(0).padStart(8)} ${p.browser.toFixed(0).padStart(8)} ${p.total.toFixed(0).padStart(8)} ${p.game.toFixed(0).padStart(8)}  ${failure ? `FAILED: ${failure}` : over ? 'OVER BUDGET' : 'ok'}`,
      )
      rows.push(
        `          top at peak: ${p.top.map((t) => `${t.comm}[${t.pid}] ${t.mib.toFixed(0)}`).join(', ')}`,
      )
      if (process.env.MORTAR_PPROF) {
        rows.push(`          ${await saveProfiles(process.env.MORTAR_PPROF, scenario.name)}`)
      }
    }
    await browser.close()
  } finally {
    try {
      selftest(dir, 'destroy')
    } catch {
      // A start that failed before the marker leaves destroy nothing to delete.
    }
    if (dir === `${BASE}/mortar-mem-${process.pid}`) {
      rmSync(dir, { recursive: true, force: true })
    }
  }
  console.log(
    `${'scenario'.padEnd(9)} ${'Go'.padStart(8)} ${'browser'.padStart(8)} ${'total'.padStart(8)} ${'game'.padStart(8)}   (game: a launched game, not in the total)`,
  )
  console.log(rows.join('\n'))
  return code
}

main().then(
  (code) => process.exit(code),
  (e) => {
    console.error(e)
    process.exit(1)
  },
)
