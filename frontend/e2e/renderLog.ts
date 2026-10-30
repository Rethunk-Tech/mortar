import type { Page } from '@playwright/test'
import type { Sample } from '@rethunk/react-render-audit'
import type { Result } from './render-budget.ts'
import { methodOf } from './renderMeter.ts'

// A backend call this slow is felt as a stalled screen.
const SLOW_CALL_MS = 300
const oneLine = (text: string) => text.replace(/\s+/g, ' ').slice(0, 240)

/** What the page logs as an error or warning, throws, or gets refused by the backend; the caller empties it per case. */
export function watchLog(page: Page): string[] {
  const logged: string[] = []
  page.on('console', (m) => {
    // Wails warns on every load in a plain browser, which is where the harness runs.
    if (
      (m.type() === 'error' || m.type() === 'warning') &&
      !m.text().includes('Browser Environment Detected')
    ) {
      logged.push(`${m.type()}: ${oneLine(m.text())}`)
    }
  })
  page.on('pageerror', (e) => logged.push(`thrown: ${oneLine(String(e))}`))
  page.on('response', (r) => {
    if (r.status() >= 400) {
      const path = new URL(r.url()).pathname
      // A backend call is named by its bound method, which the request body carries.
      const what = path.startsWith('/wails/') ? methodOf(r.request().postData() ?? '') : path
      logged.push(`HTTP ${r.status()}: ${what}`)
    }
  })
  return logged
}

/** A fresh load: how long until the workspace showed, and what the load and its settling did. */
export function startupLine(shownMs: number, s: Sample): string {
  const list = (rows: [string, number][], unit = '') =>
    rows.map(([n, v]) => `${n} ${v}${unit}`).join(', ')
  return [
    `startup: tabs shown at ${shownMs} ms, ${s.commits} commits, ${s.mounts} mounted, ${s.updates} re-rendered, script ${s.scriptMs} ms, ${s.longTasks} long tasks (${s.longMs} ms), ${s.calls} backend calls`,
    `  most called: ${list(s.callsBy)}`,
    `  slowest: ${list(s.slowest, ' ms')}`,
    `  most re-rendered: ${list(s.top)}`,
    `  because: ${list(s.why)}`,
  ].join('\n')
}

/** Each case's logged errors, once each, and where the run's time went. */
export function printExtras(results: Result[]) {
  for (const r of results) {
    for (const e of new Set(r.errors ?? [])) {
      console.log(`LOGGED ${r.name} | ${e}`)
    }
    for (const [method, ms] of [...(r.action?.slowest ?? []), ...(r.idle?.slowest ?? [])]) {
      if (ms >= SLOW_CALL_MS) {
        console.log(`SLOW ${r.name} | ${method} ${ms} ms`)
      }
    }
  }
  const secs = (pick: (m: NonNullable<Result['ms']>) => number) =>
    Math.round(results.reduce((sum, r) => sum + (r.ms ? pick(r.ms) : 0), 0) / 1000)
  console.log(
    `time: setup ${secs((m) => m.setup)} s, action ${secs((m) => m.action)} s, idle ${secs((m) => m.idle)} s, checks and pictures ${secs((m) => m.check ?? 0)} s`,
  )
}

const pad = (v: string | number, n: number) => String(v).padStart(n)

export function report(results: Result[]): string {
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
