import type { CDPSession, Page, Request } from '@playwright/test'

const QUIET_POLL_MS = 100
const TOP = Number(process.env.MORTAR_RENDER_TOP ?? '8')
// Library building blocks every MUI component renders through; the report names the app's own components instead.
const LIBRARY =
  /^(Insertion|Styled\(|Mui|Box$|Typography$|ButtonBase$|anonymous$|TouchRipple|Ripple$|Popper|Portal$|Grow$|Fade$|Collapse$|Transition|SvgIcon$|Tooltip$|Modal$|Dialog$|Paper$|Button$|IconButton$|InputBase$|FormControl$|Chip$|Menu$|List|TableCell$|TableRow$)/
const MIB = 1024 * 1024

export interface Sample {
  commits: number
  mounts: number
  updates: number
  top: [string, number][]
  /** The components cascades of re-renders started at, with how many re-renders each started, most first. */
  startedBy: [string, number][]
  /** What changed at each root (changed props, or the position and a preview of each changed state hook), by the re-renders it caused. */
  why: [string, number][]
  mutations: number
  longTasks: number
  longMs: number
  calls: number
  callsBy: [string, number][]
  layouts: number
  styles: number
  scriptMs: number
  heapMiB: number
  nodes: number
  listeners: number
}

interface PageCounts {
  commits: number
  mounts: number
  updates: number
  names: Record<string, number>
  roots: Record<string, number>
  why: Record<string, number>
  mutations: number
  longTasks: number
  longMs: number
}

type Metrics = Record<string, number>

export class Meter {
  calls = 0
  private inflight = 0
  private lastCallAt = 0
  private readonly named = new Map<string, number>()
  private before: Metrics = {}
  private readonly page: Page
  private readonly cdp: CDPSession

  constructor(page: Page, cdp: CDPSession) {
    this.page = page
    this.cdp = cdp
    const settled = (r: Request) => {
      if (new URL(r.url()).pathname.startsWith('/wails/')) {
        this.inflight -= 1
        this.lastCallAt = Date.now()
      }
    }
    page.on('requestfinished', settled)
    page.on('requestfailed', settled)
    page.on('request', (r) => {
      if (new URL(r.url()).pathname.startsWith('/wails/')) {
        this.inflight += 1
        this.calls += 1
        const what = (r.postData() ?? '').slice(0, 90)
        this.named.set(what, (this.named.get(what) ?? 0) + 1)
      }
    })
  }

  /** Waits until no backend call is in flight and none finished for quietMs, or maxMs passes. */
  async untilQuiet(quietMs: number, maxMs: number) {
    const end = Date.now() + maxMs
    while (Date.now() < end && (this.inflight > 0 || Date.now() - this.lastCallAt < quietMs)) {
      await this.page.waitForTimeout(QUIET_POLL_MS)
    }
  }

  private async metrics(): Promise<Metrics> {
    const { metrics } = await this.cdp.send('Performance.getMetrics')
    return Object.fromEntries(metrics.map((m) => [m.name, m.value]))
  }

  async reset() {
    await this.page.evaluate(() => (globalThis as unknown as RenderWindow).mortarRenders.reset())
    this.calls = 0
    this.named.clear()
    this.before = await this.metrics()
  }

  async read(): Promise<Sample> {
    const c = await this.page.evaluate(() =>
      (globalThis as unknown as RenderWindow).mortarRenders.read(),
    )
    const m = await this.metrics()
    const delta = (k: string) => (m[k] ?? 0) - (this.before[k] ?? 0)
    return {
      commits: c.commits,
      mounts: c.mounts,
      updates: c.updates,
      top: Object.entries(c.names)
        .filter(([n]) => !LIBRARY.test(n))
        .sort((a, b) => b[1] - a[1])
        .slice(0, TOP),
      startedBy: Object.entries(c.roots)
        .sort((a, b) => b[1] - a[1])
        .slice(0, TOP),
      why: Object.entries(c.why)
        .sort((a, b) => b[1] - a[1])
        .slice(0, TOP),
      mutations: c.mutations,
      longTasks: c.longTasks,
      longMs: Math.round(c.longMs),
      calls: this.calls,
      callsBy: [...this.named].sort((a, b) => b[1] - a[1]).slice(0, TOP),
      layouts: delta('LayoutCount'),
      styles: delta('RecalcStyleCount'),
      scriptMs: Math.round(delta('ScriptDuration') * 1000),
      heapMiB: Math.round(((m.JSHeapUsedSize ?? 0) / MIB) * 10) / 10,
      nodes: m.Nodes ?? 0,
      listeners: m.JSEventListeners ?? 0,
    }
  }
}

interface RenderWindow {
  mortarRenders: { reset: () => void; read: () => PageCounts }
}
