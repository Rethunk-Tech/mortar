import { execFileSync } from 'node:child_process'
import {
  browserRoot,
  byRole,
  cpuSeconds,
  exeOf,
  MIB,
  mappings,
  pssKiB,
  roleOf,
  tree,
} from '@rethunk/react-render-audit/proc'
import { sandboxPort } from './sandbox.ts'

export { browserRoot, byRole, cpuSeconds, MIB, mappings, pssKiB, tree }

export interface Peak {
  go: number
  browser: number
  game: number
  total: number
  /** The five biggest processes at the total's peak. */
  top: { pid: number; comm: string; mib: number }[]
}

export function serverPid(): number {
  const out = execFileSync('ss', ['-ltnp', `sport = :${sandboxPort()}`], { encoding: 'utf8' })
  const m = /pid=(\d+)/.exec(out)
  if (!m) {
    throw new Error('the sandbox server is not listening')
  }
  return Number(m[1])
}

interface Proc {
  pid: number
  comm: string
  mib: number
}

const SAMPLE_MS = 100

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
      [...pids].map((pid) => ({ pid, comm: roleOf(pid), mib: pssKiB(pid) / MIB }))
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
