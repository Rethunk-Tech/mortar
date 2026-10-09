import { execFileSync } from 'node:child_process'
import { readdirSync, readFileSync, readlinkSync } from 'node:fs'
import process from 'node:process'
import { sandboxPort } from './sandbox.ts'

// Process-tree memory sampling for mem-budget.ts: PSS of the sandbox server's tree and of the browser's.

export const MIB = 1024
const SAMPLE_MS = 100

export interface Peak {
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

export function tree(root: number): number[] {
  const kids = parents()
  const out = [root]
  for (const parent of out) {
    for (const [pid, ppid] of kids) {
      if (ppid === parent) {
        out.push(pid)
      }
    }
  }
  return out
}

/** KiB of PSS (RSS where the kernel gives no PSS) of one process, 0 once it is gone. */
export function pssKiB(pid: number): number {
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
export function browserRoot(): number {
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

export function serverPid(): number {
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
    const comm = readFileSync(`/proc/${pid}/comm`, 'utf8').trim()
    // A Chromium child names its role (renderer, gpu-process, utility) only on its command line.
    const role = /--type=([\w-]+)/.exec(readFileSync(`/proc/${pid}/cmdline`, 'utf8'))
    return role?.[1] ? `${comm}:${role[1]}` : comm
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
export function sample(listener: () => number, browser: () => number, server: string): () => Peak {
  const peak: Peak = { go: 0, browser: 0, game: 0, total: 0, top: [] }
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

const CLOCK_TICKS_PER_S = 100

/** Seconds of CPU (user plus system) the process has used, 0 once it is gone. */
export function cpuSeconds(pid: number): number {
  try {
    const stat = readFileSync(`/proc/${pid}/stat`, 'utf8')
    // The command name may hold spaces, so the numeric fields start after its closing parenthesis.
    const fields = stat.slice(stat.lastIndexOf(')') + 2).split(' ')
    return (Number(fields[11]) + Number(fields[12])) / CLOCK_TICKS_PER_S
  } catch {
    return 0
  }
}
