import { mkdirSync, rmSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { chromium, type Page } from '@playwright/test'
import {
  openGameSelect,
  openProfile,
  openSeedFarm,
  openSettings,
  waitForQuietNotices,
} from './app.ts'
import { nestedBoxes } from './renderBoxes.ts'
import { newCases } from './renderCases.ts'
import { unevenSpacing } from './renderGaps.ts'
import { HOOK } from './renderHook.ts'
import { Meter, type Sample } from './renderMeter.ts'
import { VIEWPORT } from './renderViewport.ts'
import { sandboxPort, selftest } from './sandbox.ts'

// Render budget: walks every screen and the common interactions of a copied (or seeded) profile in headless Chromium and
// measures, per case, what React did (commits, components mounted, components re-rendered and which), what the browser
// did (DOM mutations, layouts, style recalcs, script time, long tasks), what the page holds (JS heap, DOM nodes, event
// listeners) and how many backend calls it made: once over the action itself, and once while the screen sits idle.
// The build measured is the production bundle the sandbox serves (it keeps component names), not the dev server,
// whose StrictMode doubles effects.
//
//   MORTAR_RENDER_DATA=<data folder>   measure a copy of that data (default: the maintainer's own, selftest --copy-data)
//   MORTAR_RENDER_SEEDED=1             the small seeded sandbox instead (a smoke run)
//   MORTAR_RENDER_URL=http://…         drive an already running server (selftest.sh start) instead of a new sandbox
//   MORTAR_RENDER_PROFILE=<name>       open this profile (default: whichever profile Mortar reopens; seeded: Seed Farm)
//   MORTAR_RENDER_ONLY=mods,config     run only the cases whose names start with these
//   MORTAR_RENDER_SETTLE_MS / _IDLE_MS how long an action may settle (3000) and how long idle is watched (10000)
//   MORTAR_RENDER_GAPS=1              print each screen's unevenly spaced stacks and cards (renderGaps.ts)
//   MORTAR_RENDER_TOP=<n>              how many components, roots and causes each case lists (8)
// Writes render-budget.json under MORTAR_RENDER_OUT (default /var/tmp/render-budget-<ts>) and exits 1 when a case is
// over its budget or fails.

const BASE = '/var/tmp'

const SETTLE_MS = Number(process.env.MORTAR_RENDER_SETTLE_MS ?? '3000')
const IDLE_MS = Number(process.env.MORTAR_RENDER_IDLE_MS ?? '10000')
const IDLE_SCALE_MS = 10_000
const QUIET_MS = 1500
const QUIET_MAX_MS = 60_000
const TYPE_DELAY_MS = 120
const STEP_TIMEOUT_MS = 15_000

/** Limits per case. An update is a component that rendered again; a mount is new UI and costs what the screen holds. */
interface Budget {
  updates: number
  idleUpdates: number
  idleCommits: number
  idleCalls: number
}

const DEFAULT_BUDGET: Budget = { updates: 1000, idleUpdates: 0, idleCommits: 0, idleCalls: 2 }

export interface Case {
  name: string
  /** Puts the app where the case starts; not measured. */
  setup?: (page: Page) => Promise<void>
  /** The measured action. */
  run: (page: Page) => Promise<void>
  /** Undoes what setup changed for the cases after it, such as the window size. Runs even when the case fails. */
  teardown?: (page: Page) => Promise<void>
  budget?: Partial<Budget>
}

const tabs = (page: Page) => page.getByRole('tablist', { name: 'Profile sections' })

/** Back to the profile's workspace from a full page (Settings, Profiles, Game select), reloading as a last resort. */
async function toWorkspace(page: Page) {
  for (let i = 0; i < 2 && !(await tabs(page).isVisible()); i++) {
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
  }
  if (!(await tabs(page).isVisible())) {
    await land(page)
  }
}

async function tab(page: Page, name: string) {
  await toWorkspace(page)
  await tabs(page)
    .getByRole('tab', { name: new RegExp(`^${name}`) })
    .click()
}

async function view(page: Page, which: 'List view' | 'Grid view') {
  await tab(page, 'Mods')
  // A filter typed by an earlier case persists per profile and would leave a grid with fewer cards than a case clicks.
  const filter = page.getByRole('textbox', { name: /^Filter/ }).first()
  if ((await filter.isVisible()) && (await filter.inputValue()) !== '') {
    await filter.fill('')
  }
  const button = page.getByRole('button', { name: which })
  if ((await button.getAttribute('aria-pressed')) !== 'true') {
    await button.click()
  }
  // Closes the details panel, so a case that opens it starts from the same place every time.
  if (await page.getByRole('complementary', { name: 'Selected mod' }).isVisible()) {
    await page.keyboard.press('Escape')
  }
}

const row = (page: Page, n: number) => page.locator('[data-mod-id]').nth(n)

async function typeInto(page: Page, label: RegExp | string, text: string) {
  const field = page.getByRole('textbox', { name: label }).first()
  await field.fill('')
  await field.pressSequentially(text, { delay: TYPE_DELAY_MS })
}

/** Opens a profile section from Home, so the figure is that section's own and not what the last case left open. */
const open = (name: string): Case => ({
  name: name.toLowerCase().replace(/ /g, '-'),
  setup: (p) => tab(p, name === 'Home' ? 'Saves' : 'Home'),
  run: (p) => tab(p, name),
})

/** Opens the Mods tab in a view, from Home. */
const openMods = (which: 'List view' | 'Grid view'): Case => ({
  name: which === 'List view' ? 'mods-list' : 'mods-grid',
  setup: async (p) => {
    await view(p, which)
    await tab(p, 'Home')
  },
  run: (p) => tab(p, 'Mods'),
})

// The mod mods-list-toggle-off switched off, which mods-list-toggle-on puts back.
let switched: string | null = null // its name

const cases: Case[] = [
  open('Home'),
  openMods('List view'),
  { name: 'mods-list-select', setup: (p) => view(p, 'List view'), run: (p) => row(p, 6).click() },
  {
    name: 'mods-list-switch',
    setup: async (p) => {
      await view(p, 'List view')
      await row(p, 6).click()
    },
    run: (p) => row(p, 10).click(),
  },
  // One switch per case, each its own action; the second puts the mod back as the first found it.
  {
    name: 'mods-list-toggle-off',
    setup: (p) => view(p, 'List view'),
    run: async (p) => {
      const target = row(p, 8)
      switched = await target.locator('[title]').first().getAttribute('title')
      await target.locator('input[type=checkbox]').first().click()
    },
  },
  {
    name: 'mods-list-toggle-on',
    // The switched-off mod moves to the Disabled group, below the rendered window, so it is found by its name.
    setup: async (p) => {
      await view(p, 'List view')
      if (switched !== null) {
        await typeInto(p, /^Filter/, switched)
      }
    },
    run: (p) =>
      row(p, switched === null ? 8 : 0)
        .locator('input[type=checkbox]')
        .first()
        .click(),
  },
  {
    name: 'mods-list-filter',
    setup: (p) => view(p, 'List view'),
    run: (p) => typeInto(p, /^Filter/, 'farm'),
  },
  {
    name: 'mods-view-switch',
    setup: (p) => view(p, 'List view'),
    run: (p) => view(p, 'Grid view'),
  },
  openMods('Grid view'),
  { name: 'mods-grid-select', setup: (p) => view(p, 'Grid view'), run: (p) => row(p, 6).click() },
  {
    name: 'mods-grid-switch',
    setup: async (p) => {
      await view(p, 'Grid view')
      await row(p, 6).click()
    },
    run: (p) => row(p, 10).click(),
  },
  open('Saves'),
  open('Config'),
  {
    name: 'config-select',
    setup: (p) => tab(p, 'Config'),
    run: (p) =>
      p.getByRole('group', { name: 'Mods with settings' }).getByRole('button').nth(4).click(),
  },
  open('Screenshots'),
  open('Browse'),
  {
    name: 'browse-search',
    setup: (p) => tab(p, 'Browse'),
    run: (p) => typeInto(p, /^Search/, 'farm'),
  },
  open('Problems'),
  open('Load order'),
  open('Performance'),
  open('Console'),
  { name: 'settings', setup: toWorkspace, run: openSettings },
  { name: 'settings-search', setup: openSettings, run: (p) => typeInto(p, /^Search/, 'theme') },
  {
    name: 'profiles',
    setup: toWorkspace,
    run: async (p) => {
      await p.getByRole('button', { name: /^Switch profile/ }).click()
      await p.getByRole('menuitem', { name: /^Manage profiles/ }).click()
    },
  },
  { name: 'game-select', setup: toWorkspace, run: openGameSelect },
  ...newCases({ toWorkspace, tab, view, row, typeInto, tabs, settleMs: SETTLE_MS }),
]

/** Opens the profile the run measures. */
async function land(page: Page) {
  const named = process.env.MORTAR_RENDER_PROFILE
  if (named) {
    await openProfile(page, named)
    return
  }
  if (process.env.MORTAR_RENDER_SEEDED) {
    await openSeedFarm(page)
    return
  }
  await page.goto('/')
  const welcome = page.getByRole('button', { name: 'Continue' })
  await welcome.or(tabs(page)).first().waitFor({ timeout: 30_000 })
  if (await welcome.isVisible()) {
    await welcome.click()
  }
  await tabs(page).waitFor({ timeout: 30_000 })
  const skip = page.getByRole('button', { name: 'Skip tour' })
  if (await skip.isVisible({ timeout: 1500 }).catch(() => false)) {
    await skip.click()
  }
  await waitForQuietNotices(page)
}

interface Result {
  name: string
  action?: Sample
  idle?: Sample
  over: string[]
  failure?: string
}

function overBudget(c: Case, action: Sample, idle: Sample): string[] {
  const b = { ...DEFAULT_BUDGET, ...c.budget }
  const over: string[] = []
  if (action.updates > b.updates) {
    over.push(`re-rendered ${action.updates} > ${b.updates}`)
  }
  if (idle.updates > b.idleUpdates) {
    over.push(`idle re-rendered ${idle.updates} > ${b.idleUpdates}`)
  }
  if (idle.commits > b.idleCommits) {
    over.push(`idle commits ${idle.commits} > ${b.idleCommits}`)
  }
  if (idle.calls > b.idleCalls) {
    over.push(`idle backend calls ${idle.calls} > ${b.idleCalls}`)
  }
  return over
}

/** Idle figures per 10 s, whatever the watch length. */
function perIdle(s: Sample): Sample {
  const k = IDLE_SCALE_MS / IDLE_MS
  const scale = (n: number) => Math.round(n * k)
  return {
    ...s,
    commits: scale(s.commits),
    mounts: scale(s.mounts),
    updates: scale(s.updates),
    mutations: scale(s.mutations),
    calls: scale(s.calls),
    layouts: scale(s.layouts),
    styles: scale(s.styles),
    scriptMs: scale(s.scriptMs),
    top: s.top.map(([n, v]) => [n, scale(v)]),
    startedBy: s.startedBy.map(([n, v]) => [n, scale(v)]),
    why: s.why.map(([n, v]) => [n, scale(v)]),
  }
}

async function measure(page: Page, meter: Meter, c: Case): Promise<Result> {
  try {
    await c.setup?.(page)
    await page.waitForTimeout(SETTLE_MS)
    await meter.reset()
    await c.run(page)
    await page.waitForTimeout(SETTLE_MS)
    // Background checks (a profile's badges) finish after the settle; what they cause belongs to the action, not idle.
    await meter.untilQuiet(QUIET_MS, QUIET_MAX_MS)
    const action = await meter.read()
    await meter.reset()
    await page.waitForTimeout(IDLE_MS)
    const idle = perIdle(await meter.read())
    return { name: c.name, action, idle, over: overBudget(c, action, idle) }
  } catch (e) {
    return { name: c.name, over: [], failure: String(e).split('\n')[0] }
  } finally {
    await c.teardown?.(page)
  }
}

const pad = (v: string | number, n: number) => String(v).padStart(n)

function report(results: Result[]): string {
  const head = `${'case'.padEnd(18)}${pad('commits', 8)}${pad('mounted', 9)}${pad('re-rend', 9)}${pad('mutate', 8)}${pad('layout', 7)}${pad('script', 8)}${pad('settled', 9)}${pad('long', 6)}${pad('calls', 6)}${pad('heap', 7)}${pad('nodes', 7)} | idle/10s: ${pad('commits', 7)}${pad('re-rend', 8)}${pad('calls', 6)}`
  const lines = [
    head,
    '  action: commits, components mounted and re-rendered, DOM mutations, layouts, script ms, ms until the last commit, long tasks, backend calls, then JS heap MiB and DOM nodes after it',
  ]
  for (const r of results) {
    if (!(r.action && r.idle)) {
      lines.push(`${r.name.padEnd(18)}FAILED: ${r.failure}`)
      continue
    }
    const a = r.action
    const i = r.idle
    lines.push(
      `${r.name.padEnd(18)}${pad(a.commits, 8)}${pad(a.mounts, 9)}${pad(a.updates, 9)}${pad(a.mutations, 8)}${pad(a.layouts, 7)}${pad(a.scriptMs, 8)}${pad(a.settledMs, 9)}${pad(a.longTasks, 6)}${pad(a.calls, 6)}${pad(a.heapMiB, 7)}${pad(a.nodes, 7)} | ${' '.repeat(10)}${pad(i.commits, 7)}${pad(i.updates, 8)}${pad(i.calls, 6)}${r.over.length > 0 ? `  OVER: ${r.over.join('; ')}` : ''}`,
    )
    if (a.lastWhy.length > 0) {
      lines.push(`${' '.repeat(18)}last commit at ${a.settledMs} ms: ${a.lastWhy.join(' | ')}`)
    }
    const top = [...a.top, ...i.top.map(([n, v]) => [`idle ${n}`, v] as [string, number])]
    if (r.over.length > 0 && top.length > 0) {
      lines.push(
        `${' '.repeat(18)}most re-rendered: ${top.map(([n, v]) => `${n} ${v}`).join(', ')}`,
      )
    }
    const started = [
      ...a.startedBy,
      ...i.startedBy.map(([n, v]) => [`idle ${n}`, v] as [string, number]),
    ]
    if (r.over.length > 0 && started.length > 0) {
      lines.push(`${' '.repeat(18)}started by: ${started.map(([n, v]) => `${n} ${v}`).join(', ')}`)
    }
    const extra = (label: string, rows: [string, number][]) =>
      r.over.length > 0 && rows.length > 0
        ? lines.push(`${' '.repeat(18)}${label}: ${rows.map(([n, v]) => `${n} ×${v}`).join(' | ')}`)
        : 0
    const idle = (label: string, rows: [string, number][]) =>
      rows.map(([n, v]): [string, number] => [`idle ${label}${n}`, v])
    extra('because', [...a.why, ...idle('', i.why), ...idle('call ', i.callsBy)])
  }
  return lines.join('\n')
}

function chosenCases(): Case[] {
  const only = (process.env.MORTAR_RENDER_ONLY ?? '').split(',').filter(Boolean)
  const chosen = cases.filter((c) => only.length === 0 || only.some((o) => c.name.startsWith(o)))
  if (chosen.length === 0) {
    throw new Error(`MORTAR_RENDER_ONLY names no case (${cases.map((c) => c.name).join(', ')})`)
  }
  return chosen
}

async function run(baseURL: string, chosen: Case[]): Promise<Result[]> {
  const browser = await chromium.launch()
  try {
    const page = await browser.newPage({ baseURL, viewport: VIEWPORT })
    page.setDefaultTimeout(STEP_TIMEOUT_MS)
    await page.addInitScript(HOOK)
    const cdp = await page.context().newCDPSession(page)
    await cdp.send('Performance.enable')
    const meter = new Meter(page, cdp)
    await land(page)
    const results: Result[] = []
    for (const c of chosen) {
      results.push(await measure(page, meter, c))
      console.log(`${c.name} measured`)
      if (process.env.MORTAR_RENDER_BOXES) {
        for (const b of await nestedBoxes(page)) {
          console.log(`BOX ${c.name} | ${b.inner} in ${b.outer} | ${b.text}`)
        }
      }
      if (process.env.MORTAR_RENDER_GAPS) {
        for (const g of await unevenSpacing(page)) {
          console.log(`GAP ${c.name} | ${g.kind} ${g.sizes.join('/')} | ${g.where} | ${g.text}`)
        }
      }
    }
    return results
  } finally {
    await browser.close()
  }
}

async function main(): Promise<number> {
  const chosen = chosenCases()
  const external = process.env.MORTAR_RENDER_URL
  const dir = `${BASE}/mortar-render-${process.pid}`
  let results: Result[]
  if (external) {
    results = await run(external, chosen)
  } else {
    process.env.MORTAR_E2E_DIR = dir
    mkdirSync(dir, { recursive: true })
    try {
      const seeded = Boolean(process.env.MORTAR_RENDER_SEEDED)
      if (process.env.MORTAR_RENDER_DATA) {
        process.env.MORTAR_SELFTEST_DATA = process.env.MORTAR_RENDER_DATA
      }
      selftest(dir, 'start', ...(seeded ? [] : ['--copy-data']))
      if (seeded) {
        selftest(dir, 'seed')
      }
      results = await run(`http://127.0.0.1:${sandboxPort()}`, chosen)
    } finally {
      try {
        selftest(dir, 'destroy')
      } catch {
        // A start that failed before the marker leaves destroy nothing to delete.
      }
      rmSync(dir, { recursive: true, force: true })
    }
  }
  const out = process.env.MORTAR_RENDER_OUT ?? `${BASE}/render-budget-${Date.now()}`
  mkdirSync(out, { recursive: true })
  writeFileSync(`${out}/render-budget.json`, `${JSON.stringify(results, null, 2)}\n`)
  console.log(report(results))
  console.log(`JSON: ${out}/render-budget.json`)
  return results.some((r) => r.failure || r.over.length > 0) ? 1 : 0
}

main().then(
  (code) => process.exit(code),
  (e) => {
    console.error(e)
    process.exit(1)
  },
)
