import type { Details } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/models.ts'
import type {
  Mod,
  ModFact,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { cmpText } from './cmpText.ts'
import { lastRunOf } from './lastRun.ts'
import { type Figures, figureNumber, figureText } from './listFigures.ts'
import { compareManual } from './manualOrderLogic.ts'
import { isNewer } from './nexusFormat.ts'

// Columns in menu and table order, grouped: the mod itself, its versions, its health, its place in this profile, then
// what its source says. Ids are stored in settings, so a column keeps its id when its label or data changes.
const LIST_COLUMN_GROUPS = [
  ['on', 'name', 'author', 'category', 'notes', 'id'],
  ['version', 'pluginVersion', 'latest', 'released', 'updated'],
  ['status', 'needs', 'kind'],
  ['installed', 'switchedOff', 'reason', 'lastRun', 'startup', 'order', 'size'],
  ['source', 'endorsements', 'downloads'],
] as const

const LIST_COLUMN_IDS = LIST_COLUMN_GROUPS.flat()

type ListColumnId = (typeof LIST_COLUMN_GROUPS)[number][number]

const LOCKED_LIST_COLUMNS: readonly ListColumnId[] = ['on', 'name']

const DEFAULT_VISIBLE_LIST_COLUMNS: readonly ListColumnId[] = [
  'on',
  'name',
  'version',
  'category',
  'installed',
]

const NARROW_HIDE_LIST_COLUMNS: readonly ListColumnId[] = [
  'author',
  'source',
  'category',
  'latest',
  'released',
  'pluginVersion',
  'kind',
  'id',
  'endorsements',
  'downloads',
  'updated',
  'installed',
  'switchedOff',
  'reason',
  'needs',
  'notes',
  'lastRun',
  'size',
  'startup',
  'order',
]

type ListSortDir = 'asc' | 'desc'

/** The sort the player drags into: the order of the profile's `modOrder`. It has no direction. */
const MANUAL_SORT = 'manual'

interface ListColumnSort {
  column: ListColumnId | typeof MANUAL_SORT
  dir: ListSortDir
}

const DEFAULT_LIST_COLUMN_SORT: ListColumnSort = {
  column: 'name',
  dir: 'asc',
}

const known = new Set<string>(LIST_COLUMN_IDS)
const FIRST_YEAR = 1970

const LIST_COLUMN_WIDTH: Record<ListColumnId, string> = {
  on: '46px',
  name: 'minmax(0,1fr)',
  version: '88px',
  latest: '110px',
  released: '140px',
  pluginVersion: '110px',
  kind: '120px',
  id: '180px',
  author: '130px',
  source: '128px',
  category: '130px',
  endorsements: '88px',
  downloads: '88px',
  updated: '140px',
  installed: '140px',
  switchedOff: '140px',
  reason: '124px',
  needs: '140px',
  status: '100px',
  notes: '160px',
  lastRun: '88px',
  size: '88px',
  startup: '88px',
  order: '72px',
}

interface ListRow {
  mod: Mod
  added: string
  /** When the mod was last switched off; empty while it is on or when the time was not recorded. */
  switchedOff: string
  source: string
  /** The entry records no page, or no file on its page, to get it from again. */
  unsourced?: boolean
  status: string
  note: string
  tags: string[]
  size?: number
  /** Milliseconds the mod (or content pack) added to the last measured startup. */
  startupMs?: number
  /** Place in the profile's package order, 1 for the package that loses every file. */
  order?: number
  /** Files this package wins over earlier ones in the package order. */
  overrides?: number
  categoryOverride?: string
  categoryLabel: string
  siteCategory?: string | undefined
  groupName?: string
  download?: string
  pinned?: boolean
  implicit?: boolean
  /** How far an implicit mod sits beneath the mod that requires it, when the list nests them. */
  depth?: number
  /** The implicit mods a card view folds into this mod's card, in list order. */
  libraries?: readonly ListRow[]
  /** Explicit or Implicit, in the reader's language. */
  reason: string
  details?: Details
  /** What the mod's own files and its source's cached listing say (ModFacts). */
  facts?: ModFact
  /** Latest, release date, update date, rating and downloads from whichever source the mod came from. */
  figures?: Figures
}

function sanitizeListColumns(ids: readonly string[] | null | undefined): ListColumnId[] {
  if (!ids || ids.length === 0) {
    return [...DEFAULT_VISIBLE_LIST_COLUMNS]
  }
  const seen = new Set<ListColumnId>()
  const out: ListColumnId[] = []
  let knownCount = 0
  for (const id of ids) {
    if (known.has(id) && !seen.has(id as ListColumnId)) {
      knownCount += 1
      const col = id as ListColumnId
      seen.add(col)
      out.push(col)
    }
  }
  if (knownCount === 0) {
    return [...DEFAULT_VISIBLE_LIST_COLUMNS]
  }
  LOCKED_LIST_COLUMNS.forEach((id, i) => {
    if (!seen.has(id)) {
      out.splice(i, 0, id)
      seen.add(id)
    }
  })
  return out
}

function sanitizeListSort(column: string, dir: string): ListColumnSort {
  const sortCol =
    column === MANUAL_SORT || (known.has(column) && column !== 'on')
      ? (column as ListColumnSort['column'])
      : DEFAULT_LIST_COLUMN_SORT.column
  return { column: sortCol, dir: dir === 'desc' ? 'desc' : 'asc' }
}

function nextListSort(current: ListColumnSort, clicked: ListColumnId): ListColumnSort {
  if (clicked === 'on') {
    return current
  }
  if (current.column === clicked) {
    return { column: clicked, dir: current.dir === 'asc' ? 'desc' : 'asc' }
  }
  return { column: clicked, dir: 'asc' }
}

function toggleListColumn(visible: readonly ListColumnId[], id: ListColumnId): ListColumnId[] {
  if (LOCKED_LIST_COLUMNS.includes(id)) {
    return visible.includes(id) ? [...visible] : sanitizeListColumns([...visible, id])
  }
  if (visible.includes(id)) {
    return visible.filter((c) => c !== id)
  }
  return [...visible, id]
}

function visibleListColumns(
  saved: readonly string[] | null | undefined,
  narrow: boolean,
  available: (id: ListColumnId) => boolean = () => true,
): ListColumnId[] {
  return sanitizeListColumns(saved).filter(
    (id) => available(id) && !(narrow && NARROW_HIDE_LIST_COLUMNS.includes(id)),
  )
}

function moveListColumn(ids: readonly ListColumnId[], from: number, to: number): ListColumnId[] {
  if (from === to || from < 0 || to < 0 || from >= ids.length || to >= ids.length) {
    return [...ids]
  }
  const next = [...ids]
  const [item] = next.splice(from, 1)
  if (item === undefined) {
    return [...ids]
  }
  next.splice(to, 0, item)
  return next
}

// A row's fixed space: its 10px gaps between tracks and 16px padding each side.
const ROW_GAP_PX = 10
const ROW_PAD_PX = 32
const NAME_MIN_PX = 160
const ICON_TRACK_PX = 54

function fixedListWidth(cols: readonly ListColumnId[]): number {
  let px = ROW_PAD_PX
  let tracks = 0
  for (const id of cols) {
    // The name column brings the icon track before it.
    const name = id === 'name'
    px += Number.parseInt(LIST_COLUMN_WIDTH[id], 10) || 0
    px += name ? ICON_TRACK_PX : 0
    tracks += name ? 2 : 1
  }
  return px + Math.max(tracks - 1, 0) * ROW_GAP_PX
}

// fitListColumns drops the optional columns, the last of the narrow-hidden ones first, until the rest leave the name
// NAME_MIN_PX in a list width px wide, so a small window or an open details panel never scrolls the list sideways.
// A width of 0 is not measured yet and keeps every column.
function fitListColumns(cols: readonly ListColumnId[], width: number): ListColumnId[] {
  let out = [...cols]
  if (width <= 0) {
    return out
  }
  for (const id of NARROW_HIDE_LIST_COLUMNS.toReversed()) {
    if (fixedListWidth(out) + NAME_MIN_PX <= width) {
      break
    }
    out = out.filter((c) => c !== id)
  }
  return out
}

function listGridColumns(cols: readonly ListColumnId[]): string {
  const parts: string[] = []
  for (const id of cols) {
    if (id === 'name') {
      // The favourite star, then the mod's picture.
      parts.push(`${ICON_TRACK_PX}px`)
    }
    parts.push(LIST_COLUMN_WIDTH[id])
  }
  return parts.join(' ')
}

function missingLast(aMissing: boolean, bMissing: boolean, dir: ListSortDir, cmp: number): number {
  if (aMissing && bMissing) {
    return 0
  }
  if (aMissing) {
    return 1
  }
  if (bMissing) {
    return -1
  }
  if (dir === 'desc') {
    return -cmp
  }
  return cmp
}

function cmpNum(a: number, b: number): number {
  if (a === b) {
    return 0
  }
  if (a < b) {
    return -1
  }
  return 1
}

function cmpVersion(a: string, b: string): number {
  if (isNewer(a, b)) {
    return 1
  }
  if (isNewer(b, a)) {
    return -1
  }
  return cmpText(a, b)
}

function yearMissing(ms: number): boolean {
  return Number.isNaN(ms) || new Date(ms).getUTCFullYear() < FIRST_YEAR
}

function notesText(row: ListRow): string {
  const tags = row.tags.join(', ')
  return [row.note, tags].filter((part) => part !== '').join(' · ')
}

function lastRunCounts(mod: Mod): { errors: number; warnings: number } | null {
  const hit = lastRunOf(mod)
  if (!hit || (hit.errors === 0 && hit.warnings === 0)) {
    return null
  }
  return { errors: hit.errors, warnings: hit.warnings }
}

function compareLastRun(a: ListRow, b: ListRow, dir: ListSortDir): number {
  const av = lastRunCounts(a.mod)
  const bv = lastRunCounts(b.mod)
  let primary = missingLast(!av, !bv, dir, cmpNum(av?.errors ?? 0, bv?.errors ?? 0))
  if (primary === 0 && av && bv) {
    primary = missingLast(false, false, dir, cmpNum(av.warnings, bv.warnings))
  }
  return primary
}

interface SortValue {
  missing: boolean
  key: string | number
}

const textKey =
  (read: (row: ListRow) => string, emptyCounts = true) =>
  (row: ListRow): SortValue => {
    const key = read(row)
    return { missing: emptyCounts && !key, key }
  }

const numKey =
  (read: (row: ListRow) => number | undefined) =>
  (row: ListRow): SortValue => {
    const key = read(row)
    return { missing: key === undefined, key: key ?? 0 }
  }

const dateKey =
  (read: (row: ListRow) => string, miss: (ms: number) => boolean) =>
  (row: ListRow): SortValue => {
    const key = Date.parse(read(row))
    return { missing: miss(key), key }
  }

const LIST_SORT_VALUE: {
  [C in Exclude<ListColumnId, 'on' | 'lastRun'>]: (row: ListRow) => SortValue
} = {
  name: textKey((row) => row.mod.name, false),
  version: textKey((row) => row.mod.version),
  latest: textKey((row) => figureText(row, 'latest')),
  pluginVersion: textKey((row) => figureText(row, 'pluginVersion')),
  id: textKey((row) => figureText(row, 'id')),
  kind: textKey((row) => figureText(row, 'kind')),
  author: textKey((row) => row.mod.author),
  source: textKey((row) => row.source),
  category: textKey((row) => row.categoryLabel),
  endorsements: numKey((row) => figureNumber(row, 'endorsements')),
  downloads: numKey((row) => figureNumber(row, 'downloads')),
  updated: dateKey((row) => figureText(row, 'updated'), Number.isNaN),
  released: dateKey((row) => figureText(row, 'released'), Number.isNaN),
  installed: dateKey((row) => row.added, yearMissing),
  switchedOff: dateKey((row) => row.switchedOff, yearMissing),
  needs: numKey((row) => row.mod.needs?.length),
  status: textKey((row) => row.status),
  reason: textKey((row) => row.reason, false),
  notes: textKey((row) => notesText(row)),
  size: numKey((row) => row.size),
  startup: numKey((row) => row.startupMs),
  order: numKey((row) => row.order),
}

function compareSortValues(
  column: ListColumnId,
  a: SortValue,
  b: SortValue,
  dir: ListSortDir,
): number {
  let ordered: number
  if (column === 'version' || column === 'latest' || column === 'pluginVersion') {
    ordered = cmpVersion(String(a.key), String(b.key))
  } else if (typeof a.key === 'number' && typeof b.key === 'number') {
    ordered = cmpNum(a.key, b.key)
  } else {
    ordered = cmpText(String(a.key), String(b.key))
  }
  return missingLast(a.missing, b.missing, dir, ordered)
}

function compareListRows(
  a: ListRow,
  b: ListRow,
  sort: ListColumnSort,
  rank?: ReadonlyMap<string, number>,
): number {
  const { column, dir } = sort
  if (column === MANUAL_SORT) {
    return (rank ? compareManual(a, b, rank) : 0) || cmpText(a.mod.name, b.mod.name)
  }
  let primary = 0
  if (column === 'lastRun') {
    primary = compareLastRun(a, b, dir)
  } else if (column !== 'on') {
    const sortValue = LIST_SORT_VALUE[column]
    primary = compareSortValues(column, sortValue(a), sortValue(b), dir)
  }
  if (primary !== 0) {
    return primary
  }
  return cmpText(a.mod.name, b.mod.name)
}

function columnMenuFromEvent(e: {
  preventDefault: () => void
  stopPropagation: () => void
  clientY: number
  clientX: number
}) {
  e.preventDefault()
  e.stopPropagation()
  return { top: e.clientY, left: e.clientX }
}

export type { ListColumnId, ListColumnSort, ListRow }
export {
  columnMenuFromEvent,
  compareListRows,
  DEFAULT_LIST_COLUMN_SORT,
  DEFAULT_VISIBLE_LIST_COLUMNS,
  fitListColumns,
  fixedListWidth,
  LIST_COLUMN_GROUPS,
  LIST_COLUMN_IDS,
  LOCKED_LIST_COLUMNS,
  listGridColumns,
  MANUAL_SORT,
  moveListColumn,
  nextListSort,
  sanitizeListColumns,
  sanitizeListSort,
  toggleListColumn,
  visibleListColumns,
}
