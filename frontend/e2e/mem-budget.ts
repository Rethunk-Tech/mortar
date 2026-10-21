import { execFileSync, spawnSync } from 'node:child_process'
import { mkdirSync, rmSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { chromium, type Page } from '@playwright/test'
import { openProfile } from './app.ts'
import { browserBreakdown, goBreakdown } from './mem-breakdown.ts'
import { browserRoot, cpuSeconds, MIB, pssKiB, sample, serverPid, tree } from './mem-proc.ts'
import { sandboxPort, selftest, serverEnv } from './sandbox.ts'

// Measures the memory Mortar holds while it does its heaviest work: peak PSS of the Go server's process tree and of the
// browser's, per scenario. The WebView exists only in the desktop build, so in server mode headless Chromium (Playwright)
// stands in for it. PSS divides shared pages between the processes sharing them, so the sum counts each page once.
//
//   MORTAR_MEM_DATA=<a Mortar data folder>   copy of the data to measure on (else the seeded sandbox, a small smoke run)
//   MORTAR_MEM_ONLY=idle,browse              run only those scenarios (idle, browse, after-browse, install, problems,
//                                            launch, games); after-browse means something only after browse
//   MORTAR_MEM_PROFILE=<id or name>          the Stardew profile (else the one with the most mods)
//   MORTAR_MEM_BUDGET_MIB=1152               the limit for Go and browser together, per scenario
// Exits 1 when a scenario's total is over the budget. Each scenario's breakdown (Go memory classes, GOGC and the memory
// limit in effect, the top of its heap profile by package, the browser's JS heap, DOM and per-process split) goes to
// $MORTAR_MEM_OUT/report.md; the Go parts need MORTAR_PPROF.

const BASE = '/var/tmp'
const BUDGET_MIB = Number(process.env.MORTAR_MEM_BUDGET_MIB ?? '1152')
const ZIP_MIB = Number(process.env.MORTAR_MEM_ZIP_MIB ?? '500')
// archive.DefaultMaxEntries: the largest archive in a real 811-mod store holds 2,438 files.
const MAX_ZIP_ENTRIES = 20_000
const ZIP_FILE_KIB = Number(process.env.MORTAR_MEM_ZIP_FILE_KIB ?? '100')
const BROWSE_QUERY = process.env.MORTAR_MEM_QUERY ?? 'content'
const SCROLLS = 5
const SCROLL_STEP_PX = 4000
const SCROLL_WAIT_MS = 1200
const SETTLE_MS = 3000
// Enough wheel steps to pass every Game Selection row of a 280-game catalog (168 px each) quickly.
const GAME_SCROLLS = 70
const GAME_SCROLL_PX = 800
const GAME_SCROLL_WAIT_MS = 150
const LAUNCH_TIMEOUT_MS = 240_000
// Idle means the server has spent under QUIET_CPU_S of CPU in each of QUIET_SECONDS straight seconds: the catalog, index
// loads and update checks that follow a start or a screen change are done. QUIET_MIN_S covers work that waits on the
// network at near-zero CPU before it decodes; MORTAR_MEM_SETTLE_S raises it past the idle releases (Content Patcher
// packs after 2 minutes, a Thunderstore listing after 10) to see what stays for good. A server that never gets there is sampled after QUIET_MAX_S anyway, and
// the report says so.
const QUIET_CPU_S = 0.05
const QUIET_SECONDS = 5
const QUIET_MIN_S = Number(process.env.MORTAR_MEM_SETTLE_S ?? '30')
const QUIET_MAX_S = QUIET_MIN_S + 150
const IDLE_SAMPLE_MS = 5000

const dir = `${BASE}/mortar-mem-${process.pid}`

function verdict(failure: string | undefined, over: boolean): string {
  if (failure) {
    return `FAILED: ${failure}`
  }
  return over ? 'OVER BUDGET' : 'ok'
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

/** What the renderer holds outside the JS heap: DOM nodes, mounted images and the pixels they decode to. */
async function domStats(page: Page): Promise<string> {
  const d = await page.evaluate(() => {
    const imgs = [...document.images]
    const px = imgs.reduce((t, i) => t + i.naturalWidth * i.naturalHeight, 0)
    const top = imgs.reduce<HTMLImageElement | undefined>(
      (m, i) => (!m || i.naturalWidth * i.naturalHeight > m.naturalWidth * m.naturalHeight ? i : m),
      undefined,
    )
    const largest = top
      ? `${top.naturalWidth}x${top.naturalHeight} ${new URL(top.src).pathname}`
      : 'none'
    return { nodes: document.getElementsByTagName('*').length, imgs: imgs.length, px, largest }
  })
  const mib = (d.px * 4) / MIB / MIB
  return `DOM ${d.nodes} nodes, ${d.imgs} images (${mib.toFixed(0)} MiB decoded, largest ${d.largest}).`
}

/** The page's JS heap after a collection, and with MORTAR_PPROF set its heap snapshot beside the Go profiles, so the
 * renderer's share of the browser figure can be told from the GPU's and traced to the objects that hold it. */
async function browserHeap(page: Page, scenario: string, browserPid: number): Promise<string> {
  const web = () => tree(browserPid).reduce((t, pid) => t + pssKiB(pid) / MIB, 0)
  const held = web()
  const cdp = await page.context().newCDPSession(page)
  try {
    await cdp.send('HeapProfiler.collectGarbage')
    const { usedSize, totalSize } = await cdp.send('Runtime.getHeapUsage')
    let saved = ''
    if (process.env.MORTAR_PPROF) {
      mkdirSync(outDir, { recursive: true })
      const file = `${outDir}/${scenario}.browser.heapsnapshot`
      const chunks: string[] = []
      cdp.on('HeapProfiler.addHeapSnapshotChunk', (e) => chunks.push(e.chunk))
      await cdp.send('HeapProfiler.takeHeapSnapshot', { reportProgress: false })
      writeFileSync(file, chunks.join(''))
      saved = ` snapshot: ${file}`
    }
    // A critical pressure notice makes Chromium drop its reclaimable caches (decoded images, resource cache), so what
    // remains is memory the page itself holds.
    await cdp.send('Memory.simulatePressureNotification', { level: 'critical' })
    await sleep(SETTLE_MS)
    const purged = web()
    return `Browser PSS ${held.toFixed(0)} MiB, ${purged.toFixed(0)} MiB after a pressure purge. ${await domStats(page)} JS heap after GC: ${(usedSize / MIB / 1024).toFixed(0)} MiB used of ${(totalSize / MIB / 1024).toFixed(0)} MiB.${saved}`
  } finally {
    await cdp.detach()
  }
}

const PEAK_POLL_MS = 250

/** With MORTAR_PPROF set, polls the server's heap while a scenario runs and keeps the profile taken when the heap in use
 * was highest, so the code holding memory at the peak shows rather than what is left after the scenario. The returned
 * function stops polling and says where the profile is. */
function pollPeakHeap(addr: string, scenario: string): () => Promise<string> {
  let best = 0
  let running = true
  const loop = (async () => {
    mkdirSync(outDir, { recursive: true })
    while (running) {
      try {
        const text = await (await fetch(`http://${addr}/debug/pprof/heap?debug=1`)).text()
        const inuse = Number(/# HeapInuse = (\d+)/.exec(text)?.[1] ?? 0)
        if (inuse > best) {
          best = inuse
          const body = Buffer.from(
            await (await fetch(`http://${addr}/debug/pprof/heap`)).arrayBuffer(),
          )
          writeFileSync(`${outDir}/${scenario}.peak.heap.pb.gz`, body)
        }
      } catch {
        // The server is busy or gone; the next poll tries again.
      }
      await sleep(PEAK_POLL_MS)
    }
  })()
  return async () => {
    running = false
    await loop
    return `peak heap in use ${(best / MIB / MIB).toFixed(0)} MiB: ${outDir}/${scenario}.peak.heap.pb.gz`
  }
}

/** The server's completed GC cycles and its share of CPU spent collecting since start, from its pprof heap summary. */
async function gcStats(addr: string): Promise<{ cycles: number; fraction: number }> {
  const text = await (await fetch(`http://${addr}/debug/pprof/heap?debug=1`)).text()
  return {
    cycles: Number(/# NumGC = (\d+)/.exec(text)?.[1] ?? 0),
    fraction: Number(/# GCCPUFraction = ([\d.e+-]+)/.exec(text)?.[1] ?? 0),
  }
}

const sleep = (ms: number) => new Promise((done) => setTimeout(done, ms))

let env: Record<string, string> = {}
const cli = (...args: string[]) =>
  spawnSync(`${dir}/mortar-server`, args, {
    env,
    encoding: 'utf8',
    maxBuffer: 2 ** 28,
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

/** A zip of many files holding one SMAPI mod, in the sandbox's own scratch folder. */
function bigZip(): string {
  const path = `${dir}/tmp/MemBigMod.zip`
  mkdirSync(`${dir}/tmp`, { recursive: true })
  if ((ZIP_MIB * 1024) / ZIP_FILE_KIB > MAX_ZIP_ENTRIES) {
    throw new Error(
      `MORTAR_MEM_ZIP_MIB and MORTAR_MEM_ZIP_FILE_KIB make ${(ZIP_MIB * 1024) / ZIP_FILE_KIB} files; Mortar unpacks at most ${MAX_ZIP_ENTRIES}`,
    )
  }
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
  /** Runs before sampling starts, so the figure is what is held afterwards rather than the peak of getting there. */
  settle?: (page: Page) => Promise<string>
  run: (page: Page, profile: Profile) => Promise<void>
}

/** Waits until the server's CPU goes flat (see QUIET_CPU_S) and says how long that took. */
async function waitQuiet(): Promise<string> {
  const start = Date.now()
  let last = cpuSeconds(serverPid())
  let quiet = 0
  const elapsed = () => Date.now() - start
  while (
    (quiet < QUIET_SECONDS || elapsed() < QUIET_MIN_S * 1000) &&
    elapsed() < QUIET_MAX_S * 1000
  ) {
    await sleep(1000)
    const now = cpuSeconds(serverPid())
    quiet = now - last < QUIET_CPU_S ? quiet + 1 : 0
    last = now
  }
  const s = ((Date.now() - start) / 1000).toFixed(0)
  return quiet < QUIET_SECONDS
    ? `server CPU still busy after ${s} s`
    : `server CPU quiet after ${s} s`
}

/** The open profile's Home, left until the server is idle. */
const idleScenario = (name: string): Scenario => ({
  name,
  async settle(page) {
    await page.keyboard.press('Escape')
    await page.getByRole('tab', { name: 'Home' }).click()
    return waitQuiet()
  },
  async run() {
    await sleep(IDLE_SAMPLE_MS)
  },
})

const scenarios: Scenario[] = [
  idleScenario('idle'),
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
  // Whether browsing's memory is given back. browse's pressure purge has already dropped the browser's caches by then.
  idleScenario('after-browse'),
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
  {
    // Last: it leaves the page on Game Selection. Every catalog row passes by, as a player flicking through the list.
    name: 'games',
    async run(page) {
      // A launch before this leaves its run dialog open over the title bar.
      await page.keyboard.press('Escape')
      await page.getByRole('button', { name: /^(Switch game|Choose a game)/ }).click()
      await page.getByRole('menuitem', { name: 'All games…' }).click()
      await sleep(SETTLE_MS)
      for (let n = 0; n < GAME_SCROLLS; n++) {
        await page.mouse.move(700, 500)
        await page.mouse.wheel(0, GAME_SCROLL_PX)
        await sleep(GAME_SCROLL_WAIT_MS)
      }
      await sleep(SETTLE_MS)
    },
  },
]

/** Runs one scenario while sampling memory, and answers its report rows and whether it failed or went over budget. */
async function measureScenario(
  page: Page,
  profile: Profile,
  scenario: Scenario,
  browserPid: number,
): Promise<{ rows: string[]; report: string; failed: boolean }> {
  const rows: string[] = []
  let failed = false
  const settled = scenario.settle ? await scenario.settle(page) : ''
  const before = await domStats(page)
  const wallStart = Date.now()
  const cpuStart = cpuSeconds(serverPid())
  const gcStart = process.env.MORTAR_PPROF ? await gcStats(process.env.MORTAR_PPROF) : null
  const stop = sample(serverPid, () => browserPid, `${dir}/mortar-server`)
  const stopPeak = process.env.MORTAR_PPROF
    ? pollPeakHeap(process.env.MORTAR_PPROF, scenario.name)
    : null
  let failure = ''
  try {
    await scenario.run(page, profile)
  } catch (e) {
    failure = e instanceof Error ? e.message : String(e)
  }
  const p = stop()
  const wall = (Date.now() - wallStart) / 1000
  const cpu = cpuSeconds(serverPid()) - cpuStart
  let gcNote = ''
  if (gcStart && process.env.MORTAR_PPROF) {
    const end = await gcStats(process.env.MORTAR_PPROF)
    gcNote = `, ${end.cycles - gcStart.cycles} GCs, GC share of CPU since start ${(end.fraction * 100).toFixed(1)}%`
  }
  const peakNote = stopPeak ? await stopPeak() : ''
  const over = p.total > BUDGET_MIB
  failed = over || failure !== ''
  rows.push(
    `${scenario.name.padEnd(9)} ${p.go.toFixed(0).padStart(8)} ${p.browser.toFixed(0).padStart(8)} ${p.total.toFixed(0).padStart(8)} ${p.game.toFixed(0).padStart(8)}  ${verdict(failure, over)}`,
  )
  rows.push(
    `          top at peak: ${p.top.map((t) => `${t.comm}[${t.pid}] ${t.mib.toFixed(0)}`).join(', ')}`,
  )
  rows.push(`          Wall ${wall.toFixed(1)} s, Go CPU ${cpu.toFixed(1)} s${gcNote}`)
  if (settled) {
    rows.push(`          Settle: ${settled}`)
  }
  rows.push(`          Before: ${before}`)
  const [webReport, webLine] = await browserBreakdown(page, browserPid)
  rows.push(`          ${await browserHeap(page, scenario.name, browserPid)}`)
  if (peakNote) {
    rows.push(`          ${peakNote}`)
  }
  if (process.env.MORTAR_PPROF) {
    rows.push(`          ${await saveProfiles(process.env.MORTAR_PPROF, scenario.name)}`)
  }
  const [goReport, goLine] = await goBreakdown(
    process.env.MORTAR_PPROF,
    scenario.name,
    `${dir}/mortar-server`,
    outDir,
  )
  rows.push(`          ${goLine}`, `          ${webLine}`)
  const report = [
    `## ${scenario.name}`,
    '',
    `Peak PSS: Go ${p.go.toFixed(0)} MiB, browser ${p.browser.toFixed(0)} MiB, total ${p.total.toFixed(0)} MiB (${verdict(failure, over)}).${settled ? ` Sampled for ${IDLE_SAMPLE_MS / 1000} s once the ${settled}.` : ''}`,
    '',
    goReport,
    '',
    webReport,
    '',
  ].join('\n')
  return { rows, report, failed }
}

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
  const reports: string[] = []
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
      const result = await measureScenario(page, profile, scenario, browserPid)
      rows.push(...result.rows)
      reports.push(result.report)
      if (result.failed) {
        code = 1
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
  mkdirSync(outDir, { recursive: true })
  writeFileSync(`${outDir}/report.md`, `# Mortar memory breakdown\n\n${reports.join('\n')}`)
  console.log(`Breakdown: ${outDir}/report.md`)
  return code
}

main().then(
  (code) => process.exit(code),
  (e) => {
    console.error(e)
    process.exit(1)
  },
)
