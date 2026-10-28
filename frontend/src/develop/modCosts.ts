import { useEffect, useRef, useState } from 'react'
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

/** The mod's start-up time and average cost a frame in the development profile's latest measured run. */
export function useModCosts(game: string, profile: string, info: ProjectInfo | undefined): Costs {
  const [costs, setCosts] = useState<Costs>({})
  const latest = useRef(info)
  latest.current = info
  const known = info !== undefined && !info.missing
  // A reread that changes nothing here (a window focus) must not repeat the reports' reads.
  const key = info ? `${info.guid}|${info.name}` : ''
  useEffect(() => {
    const current = latest.current
    if (!(known && current) || key === '') {
      return
    }
    let stale = false
    Promise.all([StartupReports(game, profile), PerformanceReports(game, profile)])
      .then(([startups, perfs]) => {
        if (stale) {
          return
        }
        const next: Costs = {}
        const mod = startups?.[0]?.mods?.find((m) => isProjectRow(current, m.name, String(m.id)))
        if (mod) {
          next.startupMs = modTotal(mod)
        }
        const row = perfs?.[0]?.rows?.find((r) => isProjectRow(current, r.name))
        if (row) {
          next.frameMs = row.averageMs
        }
        setCosts(next)
      })
      .catch(() => !stale && setCosts({}))
    return () => {
      stale = true
    }
  }, [game, profile, known, key])
  return costs
}

/** The live Console, showing only the mod's own log source. */
export function showModLog(info: ProjectInfo) {
  useTab.getState().setTab('console')
  const mods = [...new Set([info.name, info.guid].filter((s) => s !== ''))]
  useConsole.setState({ filters: { ...DEFAULT_FILTERS, mods } })
}
