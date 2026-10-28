import { mkdirSync, rmSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { chromium, type Page } from '@playwright/test'
import {
  dwell,
  openGameSelect,
  openProfile,
  openSeedFarm,
  openSettings,
  waitForQuietNotices,
} from './app.ts'
import { serveModTemplate } from './modTemplate.ts'
import { nestedBoxes } from './renderBoxes.ts'
import { gameSettingsCases, newCases } from './renderCases.ts'
import { dialogCases } from './renderDialogs.ts'
import { writeGallery } from './renderGallery.ts'
import { unevenSpacing } from './renderGaps.ts'
import { HOOK } from './renderHook.ts'
import { type LayoutBreak, layoutBreaks } from './renderLayout.ts'
import { printExtras, report, startupLine, watchLog } from './renderLog.ts'
import { Meter, perTen, type Sample } from './renderMeter.ts'
import { screenCases } from './renderScreens.ts'
import { PAGE } from './renderViewport.ts'
import { keepServerLogs, sandboxPort, selftest } from './sandbox.ts'
import { webkitWalk } from './webkit-walk.ts'

// Render budget: walks every screen and the common interactions of a copied (or seeded) profile in headless Chromium and
// measures, per case, what React did (commits, components mounted, components re-rendered and which), what the browser
// did (DOM mutations, layouts, style recalcs, script time, long tasks), what the page holds (JS heap, DOM nodes, event
// listeners) and how many backend calls it made: once over the action itself, and once while the screen sits idle.
// The build measured is the production bundle the sandbox serves (it keeps component names), not the dev server,
// whose StrictMode doubles effects.
// Periodic timers of 5 s or more never fire (renderHook.ts), so a poll landing in a case's window cannot skew it.
//
//   MORTAR_RENDER_DATA=<data folder>   measure a copy of that data (default: the maintainer's own, selftest --copy-data)
//   MORTAR_RENDER_SEEDED=1             the small seeded sandbox instead (a smoke run)
//   MORTAR_RENDER_URL=http://…         drive an already running server (selftest.sh start) instead of a new sandbox
//   MORTAR_RENDER_PROFILE=<name>       open this profile (default: whichever profile Mortar reopens; seeded: Seed Farm)
//   MORTAR_RENDER_ONLY=mods,config     run only the cases whose names start with these
//   MORTAR_RENDER_SETTLE_MS / _IDLE_MS how long a case's own steps wait (3000) and how long idle is watched (3000); setup
//                                      and action end once nothing has committed or called the backend for 1.5 s
//   MORTAR_RENDER_GAPS=1              print each screen's unevenly spaced stacks and cards (renderGaps.ts)
//   MORTAR_RENDER_SHOTS=<dir>          where each screen's pictures go (default <out>/shots): settled in <dir>/chromium
//                                      and <dir>/webkit, loading (just after the action) in their loading/ folders;
//                                      `bun run renders:diff OLD/chromium NEW/chromium` compares two runs' settled ones
// Every screen is checked for layout breaks (renderLayout.ts) in Chromium, and with MORTAR_RENDER_WEBKIT=1 (renders:release)
// in WebKitGTK too (webkit-walk.ts) on the same sandbox; a break fails the run like a budget.
//   MORTAR_RENDER_MOTION=1             measure with motion (by default reduced, so no ripple or animation is counted)
//   MORTAR_RENDER_TOP=<n>              how many components, roots and causes each case lists (8)
//   MORTAR_RENDER_ALL_NAMES=1          list MUI's and emotion's components among the most re-rendered too
// Writes render-budget.json and index.html (every screen loading and settled, in both engines, beside its figures and
// breaks) under MORTAR_RENDER_OUT (default /var/tmp/render-budget-<ts>) and exits 1 when a case is over its budget or fails.

const BASE = '/var/tmp'

const SETTLE_MS = Number(process.env.MORTAR_RENDER_SETTLE_MS ?? '3000')
// Most cases show nothing idle; a short watch still sees churn, and the walk spends most of its time here.
const IDLE_MS = Number(process.env.MORTAR_RENDER_IDLE_MS ?? '3000')
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
    await tabs(page)
      .waitFor({ state: 'visible', timeout: 300 })
      .catch(() => undefined)
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
  // Selecting opens the sidebar, and the narrower list drops a column: every visible row re-renders once without it.
  {
    name: 'mods-list-select',
    budget: { updates: 2500 },
    setup: (p) => view(p, 'List view'),
    run: (p) => row(p, 6).click(),
  },
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
    // Its toasts offering to switch off dependents and now-unneeded libraries each leave inside the idle watch.
    budget: { idleUpdates: 20, idleCommits: 2 },
    setup: (p) => view(p, 'List view'),
    run: async (p) => {
      const target = row(p, 8)
      switched = await target.locator('[title]').first().getAttribute('title')
      await target.locator('input[type=checkbox]').first().click()
    },
  },
  {
    name: 'mods-list-toggle-on',
    // The toast naming what came on with it goes once, inside the idle watch.
    budget: { idleUpdates: 10, idleCommits: 1 },
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
    // The search is kept; later cases expect every mod listed.
    teardown: (p) => p.getByRole('button', { name: 'Show all mods' }).click(),
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
  {
    name: 'develop',
    setup: (p) => tab(p, 'Home'),
    // Only a loader with a devTemplate has the tab.
    run: async (p) => {
      if (await tabs(p).getByRole('tab', { name: /^Develop/ }).count()) {
        await tab(p, 'Develop')
      }
    },
  },
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
  ...gameSettingsCases({ toWorkspace, tab, view, row, typeInto, tabs, settleMs: SETTLE_MS }),
  ...screenCases({ toWorkspace, tab, view, row, typeInto, tabs, settleMs: SETTLE_MS }),
  ...dialogCases({ toWorkspace, tab, view, row, typeInto, tabs, settleMs: SETTLE_MS }),
]

/** Opens the profile the run measures, and returns how long a fresh load took to show the workspace's tabs. */
async function land(page: Page): Promise<number | null> {
  const named = process.env.MORTAR_RENDER_PROFILE
  if (named) {
    await openProfile(page, named)
    return null
  }
  if (process.env.MORTAR_RENDER_SEEDED) {
    await openSeedFarm(page)
    return null
  }
  const start = Date.now()
  await page.goto('/')
  const welcome = page.getByRole('button', { name: 'Continue' })
  await welcome.or(tabs(page)).first().waitFor({ timeout: 30_000 })
  if (await welcome.isVisible()) {
    await welcome.click()
  }
  await tabs(page).waitFor({ timeout: 30_000 })
  const shownMs = Date.now() - start
  const skip = page.getByRole('button', { name: 'Skip tour' })
  if (await skip.isVisible({ timeout: 1500 }).catch(() => false)) {
    await skip.click()
  }
  await waitForQuietNotices(page)
  return shownMs
}

export interface Result {
  name: string
  action?: Sample
  idle?: Sample
  over: string[]
  failure?: string
  // Wall time of the case's phases, and what the page logged or the backend refused while it ran.
  ms?: { setup: number; action: number; idle: number; check?: number }
  errors?: string[]
}

/** idle is the watch's own count: the idle budgets are per 10 s, so they shrink with a shorter watch instead of the
 * count growing, and one poll caught in a short watch is not read as several. */
function overBudget(c: Case, action: Sample, idle: Sample): string[] {
  const b = { ...DEFAULT_BUDGET, ...c.budget }
  const forWatch = (tenSecond: number) => Math.ceil((tenSecond * IDLE_MS) / IDLE_SCALE_MS)
  const over: string[] = []
  if (action.updates > b.updates) {
    over.push(`re-rendered ${action.updates} > ${b.updates}`)
  }
  if (idle.updates > forWatch(b.idleUpdates)) {
    over.push(
      `idle re-rendered ${idle.updates} > ${forWatch(b.idleUpdates)} in ${IDLE_MS / 1000} s`,
    )
  }
  if (idle.commits > forWatch(b.idleCommits)) {
    over.push(`idle commits ${idle.commits} > ${forWatch(b.idleCommits)} in ${IDLE_MS / 1000} s`)
  }
  if (idle.calls > forWatch(b.idleCalls)) {
    over.push(`idle backend calls ${idle.calls} > ${forWatch(b.idleCalls)} in ${IDLE_MS / 1000} s`)
  }
  return over
}

async function measure(
  page: Page,
  meter: Meter,
  c: Case,
  loading: () => Promise<unknown>,
): Promise<Result> {
  try {
    const t0 = Date.now()
    await c.setup?.(page)
    await meter.untilStill(QUIET_MS, QUIET_MAX_MS)
    await meter.quietToasts(c.name)
    await meter.reset()
    const t1 = Date.now()
    await c.run(page)
    await loading()
    // Background checks (a profile's badges) finish after the action; what they cause belongs to it, not to idle.
    await meter.untilStill(QUIET_MS, QUIET_MAX_MS)
    const action = await meter.read()
    await meter.reset()
    const t2 = Date.now()
    await dwell(page, IDLE_MS)
    const watched = await meter.read()
    const ms = { setup: t1 - t0, action: t2 - t1, idle: Date.now() - t2 }
    return {
      name: c.name,
      action,
      idle: perTen(watched, IDLE_MS),
      ms,
      over: overBudget(c, action, watched),
    }
  } catch (e) {
    return { name: c.name, over: [], failure: String(e).split('\n')[0] }
  }
}

function chosenCases(): Case[] {
  const only = (process.env.MORTAR_RENDER_ONLY ?? '').split(',').filter(Boolean)
  const chosen = cases.filter((c) => only.length === 0 || only.some((o) => c.name.startsWith(o)))
  if (chosen.length === 0) {
    throw new Error(`MORTAR_RENDER_ONLY names no case (${cases.map((c) => c.name).join(', ')})`)
  }
  return chosen
}

async function run(
  baseURL: string,
  chosen: Case[],
  shotsDir: string,
): Promise<{ results: Result[]; breaks: string[] }> {
  const browser = await chromium.launch()
  try {
    const page = await browser.newPage({ baseURL, ...PAGE })
    page.setDefaultTimeout(STEP_TIMEOUT_MS)
    await page.addInitScript(HOOK)
    const cdp = await page.context().newCDPSession(page)
    await cdp.send('Performance.enable')
    const meter = new Meter(page, cdp)
    const shownMs = await land(page)
    if (shownMs !== null) {
      await meter.untilStill(QUIET_MS, QUIET_MAX_MS)
      console.log(startupLine(shownMs, await meter.read()))
    }
    const results: Result[] = []
    const breaks: string[] = []
    const shots = `${shotsDir}/chromium`
    mkdirSync(`${shots}/loading`, { recursive: true })
    const logged = watchLog(page)
    for (const c of chosen) {
      logged.length = 0
      const result = await measure(page, meter, c, () =>
        page.screenshot({ path: `${shots}/loading/${c.name}.png` }),
      )
      console.log(`${c.name} measured`)
      const t = Date.now()
      // A page that reloads under the check is itself a break; the walk goes on with the next case.
      try {
        breaks.push(
          ...(await layoutBreaks(page)).map(
            (b: LayoutBreak) => `LAYOUT ${c.name} | ${b.kind} | ${b.where} | ${b.text}`,
          ),
        )
        await page.screenshot({ path: `${shots}/${c.name}.png` })
      } catch (e) {
        breaks.push(`LAYOUT ${c.name} | unchecked | ${String(e).split('\n')[0]}`)
      }
      // After the check and the picture, which show the state the case left. A teardown that fails leaves that state for
      // the next cases, so it is reported.
      await c.teardown?.(page).catch((e: unknown) => {
        breaks.push(`TEARDOWN ${c.name} | ${String(e).split('\n')[0]}`)
      })
      results.push({
        ...result,
        ...(result.ms ? { ms: { ...result.ms, check: Date.now() - t } } : {}),
        errors: [...logged],
      })
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
    return { results, breaks }
  } finally {
    await browser.close()
  }
}

// The Chromium walk, then, before a release, the WebKitGTK one on the same sandbox while it still runs (webkit-walk.ts).
async function walk(url: string, sandboxDir: string, chosen: Case[], shots: string) {
  const { results, breaks } = await run(url, chosen, shots)
  if (process.env.MORTAR_RENDER_WEBKIT !== '1') {
    return { results, breaks }
  }
  const only = (process.env.MORTAR_RENDER_ONLY ?? '').split(',').filter(Boolean)
  const webkit = await webkitWalk({ sandbox: sandboxDir, url, shots: `${shots}/webkit`, only })
  if (webkit === null) {
    console.log('WebKitGTK walk skipped: no WebKitWebDriver or no hidden display')
  }
  return { results, breaks: [...breaks, ...(webkit ?? [])] }
}

async function main(): Promise<number> {
  const chosen = chosenCases()
  const external = process.env.MORTAR_RENDER_URL
  const dir = `${BASE}/mortar-render-${process.pid}`
  const out = process.env.MORTAR_RENDER_OUT ?? `${BASE}/render-budget-${Date.now()}`
  const shots = process.env.MORTAR_RENDER_SHOTS ?? `${out}/shots`
  let results: Result[]
  let breaks: string[]
  if (external) {
    const sandboxDir = process.env.MORTAR_SELFTEST_DIR ?? `${BASE}/mortar-selftest`
    ;({ results, breaks } = await walk(external, sandboxDir, chosen, shots))
  } else {
    process.env.MORTAR_E2E_DIR = dir
    mkdirSync(dir, { recursive: true })
    const stopTemplate = serveModTemplate(dir)
    try {
      const seeded = Boolean(process.env.MORTAR_RENDER_SEEDED)
      if (process.env.MORTAR_RENDER_DATA) {
        process.env.MORTAR_SELFTEST_DATA = process.env.MORTAR_RENDER_DATA
      }
      selftest(dir, 'start', ...(seeded ? [] : ['--copy-data']))
      if (seeded) {
        selftest(dir, 'seed')
      }
      ;({ results, breaks } = await walk(`http://127.0.0.1:${sandboxPort()}`, dir, chosen, shots))
    } finally {
      keepServerLogs(dir, `${out}/sandbox-server`)
      stopTemplate()
      try {
        selftest(dir, 'destroy')
      } catch {
        // A start that failed before the marker leaves destroy nothing to delete.
      }
      rmSync(dir, { recursive: true, force: true })
    }
  }
  mkdirSync(out, { recursive: true })
  writeFileSync(`${out}/render-budget.json`, `${JSON.stringify(results, null, 2)}\n`)
  console.log(`gallery: xdg-open ${writeGallery(out, shots, results, breaks)}`)
  console.log(report(results))
  console.log(`JSON: ${out}/render-budget.json`)
  console.log(breaks.length === 0 ? 'layout: no breaks' : breaks.join('\n'))
  printExtras(results)
  return results.some((r) => r.failure || r.over.length > 0) || breaks.length > 0 ? 1 : 0
}

main().then(
  (code) => process.exit(code),
  (e) => {
    console.error(e)
    process.exit(1)
  },
)
