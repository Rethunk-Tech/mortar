import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'

const LEVEL_WIDTH = 5
const INCOMPATIBLE_GAME = 'this version of SMAPI is only compatible up to Stardew Valley'

export const LEVELS = [Level.Trace, Level.Debug, Level.Info, Level.Warn, Level.Error, Level.Alert]

export interface Filters {
  search: string
  levels: Level[]
  mods: string[]
  excludeMods: string[]
}

export const DEFAULT_FILTERS: Filters = {
  search: '',
  levels: [Level.Info, Level.Warn, Level.Error, Level.Alert],
  mods: [],
  excludeMods: [],
}

export function levelsFromFloor(floor: string): Level[] {
  const order = [Level.Trace, Level.Debug, Level.Info, Level.Warn, Level.Error]
  const start = order.indexOf(
    (
      {
        trace: Level.Trace,
        debug: Level.Debug,
        info: Level.Info,
        warn: Level.Warn,
        error: Level.Error,
      } as Record<string, Level>
    )[floor] ?? Level.Info,
  )
  return LEVELS.filter((l) => l === Level.Alert || order.indexOf(l) >= start)
}

export function shownLog(entries: Entry[], mine: boolean): Entry[] {
  return mine ? entries : []
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
    f.excludeMods.length > 0 ||
    f.levels.length !== DEFAULT_FILTERS.levels.length ||
    DEFAULT_FILTERS.levels.some((l) => !f.levels.includes(l))
  )
}

export function visible(entries: Entry[], f: Filters): Entry[] {
  const needle = f.search.trim().toLowerCase()
  const groups: Entry[][] = []
  for (const entry of entries) {
    if (!entry.cont || groups.length === 0) {
      groups.push([])
    }
    groups.at(-1)?.push(entry)
  }
  return groups.flatMap((group) => {
    const [header] = group
    if (!header) {
      return []
    }
    const matches =
      group.every((e) => f.levels.includes(e.level)) &&
      (f.mods.length === 0 || f.mods.includes(header.mod)) &&
      !f.excludeMods.includes(header.mod) &&
      (needle === '' ||
        group.some(
          (e) => e.message.toLowerCase().includes(needle) || e.mod.toLowerCase().includes(needle),
        ))
    return matches ? group : []
  })
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

export function incompatibleSMAPI(message: string): boolean {
  return message.includes(INCOMPATIBLE_GAME)
}
