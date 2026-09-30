import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'

const LEVEL_WIDTH = 5

export const LEVELS = [Level.Trace, Level.Debug, Level.Info, Level.Warn, Level.Error, Level.Alert]

export interface Filters {
  search: string
  levels: Level[]
  mods: string[]
}

export const DEFAULT_FILTERS: Filters = {
  search: '',
  levels: [Level.Info, Level.Warn, Level.Error, Level.Alert],
  mods: [],
}

export function countByLevel(entries: Entry[]): Map<Level, number> {
  const counts = new Map<Level, number>(LEVELS.map((l) => [l, 0]))
  for (const e of entries) {
    counts.set(e.level, (counts.get(e.level) ?? 0) + 1)
  }
  return counts
}

export function modsOf(entries: Entry[]): string[] {
  return [...new Set(entries.map((e) => e.mod).filter((m) => m !== ''))].sort((a, b) =>
    a.localeCompare(b),
  )
}

export function isFiltered(f: Filters): boolean {
  return (
    f.search.trim() !== '' ||
    f.mods.length > 0 ||
    f.levels.length !== DEFAULT_FILTERS.levels.length ||
    DEFAULT_FILTERS.levels.some((l) => !f.levels.includes(l))
  )
}

export function visible(entries: Entry[], f: Filters): Entry[] {
  const needle = f.search.trim().toLowerCase()
  return entries.filter(
    (e) =>
      f.levels.includes(e.level) &&
      (f.mods.length === 0 || f.mods.includes(e.mod)) &&
      (needle === '' ||
        e.message.toLowerCase().includes(needle) ||
        e.mod.toLowerCase().includes(needle)),
  )
}

export function firstError(rows: Entry[]): number {
  return rows.findIndex((e) => e.level === Level.Error && !e.cont)
}

// SMAPI pads the level to five characters; a continuation line, or the log's headerless first line, is written bare.
export function format(e: Entry): string {
  if (e.cont || e.time === '') {
    return e.message
  }
  return `[${e.time} ${e.level.padEnd(LEVEL_WIDTH)} ${e.mod}] ${e.message}`
}

export function formatAll(rows: Entry[]): string {
  return rows.map(format).join('\n')
}
