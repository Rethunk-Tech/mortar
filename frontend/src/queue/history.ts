import { History } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'

export interface HistoryEntry {
  name: string
  version: string
  source: string
  profileId: string
  batchId: string
  game: string
  modId: number
  fileId: number
  kind: string
  repo?: string
  tag?: string
  asset?: string
  latest?: boolean
  size: number
  started: number
  finished: number
  outcome: string
  error?: string
}

export interface HistoryFilters {
  outcome: string
  profileId: string
  batchId: string
}

export const emptyFilters = (): HistoryFilters => ({ outcome: '', profileId: '', batchId: '' })

export function filterHistory(entries: HistoryEntry[], filters: HistoryFilters): HistoryEntry[] {
  return entries.filter((e) => {
    if (filters.outcome && e.outcome !== filters.outcome) {
      return false
    }
    if (filters.profileId && e.profileId !== filters.profileId) {
      return false
    }
    if (filters.batchId && e.batchId !== filters.batchId) {
      return false
    }
    return true
  })
}

export function historyProfiles(entries: HistoryEntry[]): string[] {
  return [...new Set(entries.map((e) => e.profileId).filter(Boolean))]
}

export function makeGen() {
  let n = 0
  return {
    stamp: () => {
      n += 1
      return n
    },
    is: (id: number) => id === n,
    drop: () => {
      n += 1
    },
  }
}

export function loadHistory(): Promise<HistoryEntry[]> {
  return History().then((rows) =>
    (rows ?? []).map((e) => ({
      name: e.name ?? '',
      version: e.version ?? '',
      source: e.source ?? '',
      profileId: e.profileId ?? '',
      batchId: e.batchId ?? '',
      game: e.game ?? '',
      modId: e.modId ?? 0,
      fileId: e.fileId ?? 0,
      kind: e.kind ?? 'install',
      repo: e.repo ?? '',
      tag: e.tag ?? '',
      asset: e.asset ?? '',
      latest: e.latest ?? false,
      size: e.size ?? 0,
      started: e.started ?? 0,
      finished: e.finished ?? 0,
      outcome: e.outcome ?? '',
      ...(e.error ? { error: e.error } : {}),
    })),
  )
}
