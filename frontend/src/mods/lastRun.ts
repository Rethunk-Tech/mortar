import { plural } from '@lingui/core/macro'
import { create } from 'zustand'
import type { ModRunIssues } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import { LastRunIssues } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { DEFAULT_FILTERS } from '../console/filter.ts'
import { useConsole } from '../console/store.ts'
import { useTab } from '../game/tab.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { idKey } from './dependents.ts'

let loadSeq = 0

export const useLastRun = create<{
  runId: string
  byId: Record<string, ModRunIssues>
  load: (game: string, profileId: string) => Promise<void>
}>((set) => ({
  runId: '',
  byId: {},
  load: async (game, profileId) => {
    loadSeq += 1
    const n = loadSeq
    try {
      const got = await LastRunIssues(game, profileId)
      if (n !== loadSeq) {
        return
      }
      const byId: Record<string, ModRunIssues> = {}
      for (const row of got.mods ?? []) {
        if (row.uniqueId !== '') {
          byId[idKey(row.uniqueId)] = row
        }
      }
      set({ runId: got.runId ?? '', byId })
    } catch (e) {
      if (n === loadSeq) {
        reportUnexpected(e)
      }
    }
  },
}))

export function lastRunOf(mod: Mod): ModRunIssues | undefined {
  return useLastRun.getState().byId[idKey(mod.uniqueId)]
}

export function showLastRunInConsole(game: string, profileId: string, mod: Mod) {
  const { runId } = useLastRun.getState()
  const mods = [...new Set([mod.name, mod.uniqueId].filter((s) => s !== ''))]
  useTab.getState().setTab('console')
  if (runId !== '') {
    useConsole.getState().viewRun(game, profileId, runId)
  }
  useConsole.setState({ filters: { ...DEFAULT_FILTERS, mods } })
}

export function lastRunSummary(hit: ModRunIssues): string {
  const errors = hit.errors ? plural(hit.errors, { one: '# error', other: '# errors' }) : ''
  const warnings = hit.warnings
    ? plural(hit.warnings, { one: '# warning', other: '# warnings' })
    : ''
  return [errors, warnings].filter((p) => p !== '').join(', ')
}
