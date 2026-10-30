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
  frameMs?: number
}

const NO_COSTS: Costs = {}

/** The mod's start-up time and average cost a frame in the development profile's latest measured run. */
export function useModCosts(game: string, profile: string, info: ProjectInfo | undefined): Costs {
  const latest = useRef(info)
  latest.current = info
  const known = info !== undefined && !info.missing
  // A reread that changes nothing here (a window focus) must not repeat the reports' reads.
  const key = info ? `${info.guid}|${info.name}` : ''
  const current = latest.current
  const { data } = useLoaded<Costs>(
    known && current && key !== '' ? () => readCosts(game, profile, current) : null,
    [game, profile, known, key],
    NO_COSTS,
  )
  return data
}

async function readCosts(game: string, profile: string, current: ProjectInfo): Promise<Costs> {
  const [startups, perfs] = await Promise.all([
    StartupReports(game, profile),
    PerformanceReports(game, profile),
  ])
  const next: Costs = {}
  const mod = startups?.[0]?.mods?.find((m) => isProjectRow(current, m.name, String(m.id)))
  if (mod) {
    next.startupMs = modTotal(mod)
  }
  const row = perfs?.[0]?.rows?.find((r) => isProjectRow(current, r.name))
  if (row) {
    next.frameMs = row.averageMs
  }
  return next
}

/** The live Console, showing only the mod's own log source. */
export function showModLog(info: ProjectInfo) {
  useTab.getState().setTab('console')
  const mods = [...new Set([info.name, info.guid].filter((s) => s !== ''))]
  useConsole.setState({ filters: { ...DEFAULT_FILTERS, mods } })
}
