import { spawnSync } from 'node:child_process'
import type { Page } from '@playwright/test'
import { byRole, MIB, mappings, serverPid } from './mem-proc.ts'

// The per-scenario breakdown mem-budget.ts writes to report.md: what the Go server's and the browser's memory is made of.

const PPROF_TOP = 25

const mib = (bytes: number) => `${(bytes / MIB / MIB).toFixed(1)} MiB`

/** Bytes from a pprof size such as 12.50MB or 980kB. */
function pprofBytes(size: string): number {
  const m = /^([\d.]+)([kMGT]?B)$/.exec(size)
  const unit = { B: 1, kB: 1024, MB: MIB * MIB, GB: MIB * MIB * MIB, TB: MIB ** 4 }
  return m ? Number(m[1]) * unit[m[2] as keyof typeof unit] : 0
}

/** The package of a symbol such as github.com/a/b.(*T).M or encoding/json.Unmarshal: the import path before the first dot
 * of its last element. */
function packageOf(symbol: string): string {
  const name = symbol.split('[')[0] ?? symbol
  const slash = name.lastIndexOf('/')
  const dot = name.indexOf('.', slash + 1)
  return dot < 0 ? name : name.slice(0, dot)
}

/** What the server's memory is made of: its mappings by kind, the runtime's memory classes, the GOGC and limit in effect,
 * and the top of its in-use heap profile by function and by package. Answers the report section and one summary line. */
export async function goBreakdown(
  addr: string | undefined,
  scenario: string,
  exe: string,
  outDir: string,
): Promise<[string, string]> {
  const maps = Object.entries(mappings(serverPid(), exe))
    .map(([k, v]) => `${k} ${v.toFixed(1)} MiB`)
    .join(', ')
  const md = [`Go PSS by mapping: ${maps}.`]
  if (!addr) {
    md.push('', 'Set MORTAR_PPROF for the runtime memory classes and the heap profile.')
    return [md.join('\n'), `Go mappings: ${maps}`]
  }
  const metrics = new Map(
    (await (await fetch(`http://${addr}/debug/metrics`)).text())
      .split('\n')
      .filter(Boolean)
      .map((line) => {
        const [name = '', value = '0'] = line.split(' ')
        return [name, Number(value)] as const
      }),
  )
  const m = (name: string) => metrics.get(name) ?? 0
  const c = (cls: string) => m(`/memory/classes/${cls}:bytes`)
  const classes: [string, number][] = [
    ['heap objects (HeapInuse, live and not yet swept)', c('heap/objects')],
    ['heap unused (HeapInuse, free slots in in-use spans)', c('heap/unused')],
    ['heap free, not released (HeapIdle - HeapReleased)', c('heap/free')],
    ['heap released to the OS (HeapReleased)', c('heap/released')],
    ['goroutine stacks (StackInuse)', c('heap/stacks')],
    ['OS thread stacks', c('os-stacks')],
    ['MSpan', c('metadata/mspan/inuse') + c('metadata/mspan/free')],
    ['MCache', c('metadata/mcache/inuse') + c('metadata/mcache/free')],
    ['other metadata (GCSys)', c('metadata/other')],
    ['profiling buckets (BuckHashSys)', c('profiling/buckets')],
    ['other (OtherSys)', c('other')],
    ['total mapped by the runtime (Sys)', c('total')],
  ]
  const limit = m('/gc/gomemlimit:bytes')
  const gc = `GOGC ${m('/gc/gogc:percent')}, memory limit ${limit >= 2 ** 62 ? 'none' : mib(limit)}, heap goal ${mib(m('/gc/heap/goal:bytes'))}`
  md.push(
    '',
    '| Go memory class | Size |',
    '| --- | ---: |',
    ...classes.map(([k, v]) => `| ${k} | ${mib(v)} |`),
    '',
    `${gc}.`,
  )
  const resident = c('heap/objects') + c('heap/unused') + c('heap/free') + c('heap/stacks')
  let summary = `Go: heap objects ${mib(c('heap/objects'))}, heap held but free ${mib(c('heap/unused') + c('heap/free'))}, released ${mib(c('heap/released'))}, runtime-resident ${mib(resident)}; ${maps}; ${gc}`
  const top = spawnSync(
    'go',
    [
      'tool',
      'pprof',
      '-top',
      '-sample_index=inuse_space',
      '-nodefraction=0',
      exe,
      `${outDir}/${scenario}.heap.pb.gz`,
    ],
    { encoding: 'utf8', maxBuffer: 2 ** 26 },
  )
  const rows = top.stdout
    .split('\n')
    .map((line) => /^\s*([\d.]+[kMGT]?B)\s+\S+\s+\S+\s+([\d.]+[kMGT]?B)\s+\S+\s+(.+)$/.exec(line))
    .filter((r) => r !== null)
  if (rows.length === 0) {
    md.push('', `go tool pprof gave no rows: ${top.stderr.trim()}`)
    return [md.join('\n'), summary]
  }
  const packages = new Map<string, number>()
  for (const r of rows) {
    const pkg = packageOf(r[3] ?? '')
    packages.set(pkg, (packages.get(pkg) ?? 0) + pprofBytes(r[1] ?? ''))
  }
  const byPackage = [...packages].sort((a, b) => b[1] - a[1]).slice(0, PPROF_TOP)
  md.push(
    '',
    `In-use heap by package (flat, from \`go tool pprof -top -sample_index=inuse_space ${scenario}.heap.pb.gz\`):`,
    '',
    '| Package | In use |',
    '| --- | ---: |',
    ...byPackage.map(([k, v]) => `| \`${k}\` | ${mib(v)} |`),
    '',
    `Top ${PPROF_TOP} functions:`,
    '',
    '```',
    ...top.stdout
      .split('\n')
      .slice(0, top.stdout.split('\n').findIndex((l) => /flat%/.test(l)) + 1 + PPROF_TOP),
    '```',
  )
  summary += `\n          heap by package: ${byPackage
    .slice(0, 5)
    .map(([k, v]) => `${k.split('/').pop()} ${mib(v)}`)
    .join(', ')}`
  return [md.join('\n'), summary]
}

/** The page's JS heap, DOM and listeners from Chromium's own counters, and the browser's PSS per process role. Taken
 * before any GC or pressure purge, so it is what the page holds as left. */
export async function browserBreakdown(page: Page, browserPid: number): Promise<[string, string]> {
  const cdp = await page.context().newCDPSession(page)
  try {
    await cdp.send('Performance.enable')
    const { metrics } = await cdp.send('Performance.getMetrics')
    const v = (name: string) => metrics.find((x) => x.name === name)?.value ?? 0
    const roles = byRole(browserPid)
      .map(([role, size]) => `${role} ${size.toFixed(1)} MiB`)
      .join(', ')
    const line = `JS heap ${mib(v('JSHeapUsedSize'))} used of ${mib(v('JSHeapTotalSize'))}, ${v('Nodes')} DOM nodes, ${v('Documents')} documents, ${v('JSEventListeners')} listeners, ${v('LayoutObjects')} layout objects`
    return [`Browser: ${line}.\n\nBrowser PSS by process: ${roles}.`, `Browser: ${line}; ${roles}`]
  } finally {
    await cdp.detach()
  }
}
