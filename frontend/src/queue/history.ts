import { History } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'

export interface HistoryEntry {
  name: string
  version: string
  source: string
  profileId: string
  size: number
  started: number
  finished: number
  outcome: string
}

export interface HistoryFilters {
  outcome: string
  profileId: string
}

export const emptyFilters = (): HistoryFilters => ({ outcome: '', profileId: '' })

export function filterHistory(entries: HistoryEntry[], filters: HistoryFilters): HistoryEntry[] {
  return entries.filter((e) => {
    if (filters.outcome && e.outcome !== filters.outcome) {
      return false
    }
    if (filters.profileId && e.profileId !== filters.profileId) {
      return false
    }
    return true
  })
}

export function historyProfiles(entries: HistoryEntry[]): string[] {
  return [...new Set(entries.map((e) => e.profileId).filter(Boolean))]
}

export function loadHistory(): Promise<HistoryEntry[]> {
  return History().then((rows) =>
    (rows ?? []).map((e) => ({
      name: e.name ?? '',
      version: e.version ?? '',
      source: e.source ?? '',
      profileId: e.profileId ?? '',
      size: e.size ?? 0,
      started: e.started ?? 0,
      finished: e.finished ?? 0,
      outcome: e.outcome ?? '',
    })),
  )
}
