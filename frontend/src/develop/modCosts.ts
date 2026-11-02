import { useLoaded } from '@rethunk/react-load-hooks'
import { useRef } from 'react'
import type { ProjectInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  PerformanceReports,
  StartupReports,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { DEFAULT_FILTERS } from '../console/filter.ts'
import { modTotal } from '../console/startupView.ts'
import { useConsole } from '../console/store.ts'
import { useTab } from '../game/tab.ts'
import { isProjectRow } from './projectRow.ts'

interface Costs {
  startupMs?: number
  startupAt?: string
  frameMs?: number
  frameAt?: string
}

const NO_COSTS: Costs = {}

/** The project's start-up time and average cost a frame from the latest measured launch, each with when it was
 * measured, when that came after builtAt (Unix ms), the project's last build. Only a launch asked to be measured reports
 * them, so they can be much older than the last run. */
export function useModCosts(
  game: string,
  profile: string,
  info: ProjectInfo | undefined,
  builtAt: number,
): Costs {
  const latest = useRef(info)
  latest.current = info
  const known = info !== undefined && !info.missing
  // A reread that changes nothing here (a window focus) must not repeat the reports' reads.
  const key = info ? `${info.guid}|${info.name}|${builtAt}` : ''
  const current = latest.current
  const { data } = useLoaded<Costs>(
    known && current && key !== '' ? () => readCosts(game, profile, current, builtAt) : null,
    [game, profile, known, key],
    NO_COSTS,
  )
  return data
}

async function readCosts(
  game: string,
  profile: string,
  current: ProjectInfo,
  builtAt: number,
): Promise<Costs> {
  const [startups, perfs] = await Promise.all([
    StartupReports(game, profile),
    PerformanceReports(game, profile),
  ])
  const next: Costs = {}
  // A run before the last build measured the previous build, so its figures say nothing about this one.
  const startup = startups?.[0]
  const mod = measured(startup?.processStart, builtAt)
    ? startup?.mods?.find((m) => isProjectRow(current, m.name, String(m.id)))
    : undefined
  if (mod) {
    next.startupMs = modTotal(mod)
    next.startupAt = startup?.processStart ?? ''
  }
  const perf = perfs?.[0]
  const row = measured(perf?.at, builtAt)
    ? perf?.rows?.find((r) => isProjectRow(current, r.name))
    : undefined
  if (row) {
    next.frameMs = row.averageMs
    next.frameAt = perf?.at ?? ''
  }
  return next
}

/** Whether a run at the time `at` (an ISO date) ran the build written at `builtAt` (Unix ms, 0 before any build). */
export function measured(at: string | undefined, builtAt: number): boolean {
  return at !== undefined && Date.parse(at) >= builtAt
}

/** The live Console, showing only the mod's own log source. */
export function showModLog(info: ProjectInfo) {
  useTab.getState().setTab('console')
  const mods = [...new Set([info.name, info.guid].filter((s) => s !== ''))]
  useConsole.setState({ filters: { ...DEFAULT_FILTERS, mods } })
}
