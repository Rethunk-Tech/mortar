import type { Details } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { SetListColumns } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { lastRunOf } from './lastRun.ts'
import { isNewer } from './nexusFormat.ts'

const LIST_COLUMN_IDS = [
  'on',
  'name',
  'version',
  'latest',
  'uniqueId',
  'author',
  'source',
  'category',
  'endorsements',
  'downloads',
  'updated',
  'installed',
  'needs',
  'status',
  'notes',
  'lastRun',
] as const

type ListColumnId = (typeof LIST_COLUMN_IDS)[number]

const LOCKED_LIST_COLUMNS: readonly ListColumnId[] = ['on', 'name']

const DEFAULT_VISIBLE_LIST_COLUMNS: readonly ListColumnId[] = [
  'on',
  'name',
  'version',
  'author',
  'source',
  'category',
  'status',
]

const NARROW_HIDE_LIST_COLUMNS: readonly ListColumnId[] = [
  'author',
  'source',
  'category',
  'latest',
  'uniqueId',
  'endorsements',
  'downloads',
  'updated',
  'installed',
  'needs',
  'notes',
  'lastRun',
]

type ListSortDir = 'asc' | 'desc'

interface ListColumnSort {
  column: ListColumnId
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
  uniqueId: '140px',
  author: '130px',
  source: '100px',
  category: '130px',
  endorsements: '88px',
  downloads: '88px',
  updated: '140px',
  installed: '140px',
  needs: '140px',
  status: '100px',
  notes: '160px',
  lastRun: '88px',
}

interface ListRow {
  mod: Mod
  added: string
  source: string
  status: string
  note: string
  tags: string[]
  categoryOverride?: string
  categoryLabel: string
  pinned?: boolean
  details?: Details
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
    known.has(column) && column !== 'on'
      ? (column as ListColumnId)
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
): ListColumnId[] {
  return sanitizeListColumns(saved).filter(
    (id) => !(narrow && NARROW_HIDE_LIST_COLUMNS.includes(id)),
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

function listGridColumns(cols: readonly ListColumnId[]): string {
  const parts: string[] = []
  for (const id of cols) {
    if (id === 'name') {
      parts.push('26px')
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

function cmpText(a: string, b: string): number {
  return a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' })
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

function pageOf(row: ListRow) {
  return row.details?.page
}

function isMissing(value: unknown): boolean {
  return value === undefined || value === null
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

function compareListRows(a: ListRow, b: ListRow, sort: ListColumnSort): number {
  const { column, dir } = sort
  let primary = 0
  switch (column) {
    case 'on':
      primary = 0
      break
    case 'name':
      primary = missingLast(false, false, dir, cmpText(a.mod.name, b.mod.name))
      break
    case 'version':
      primary = missingLast(
        !a.mod.version,
        !b.mod.version,
        dir,
        cmpVersion(a.mod.version, b.mod.version),
      )
      break
    case 'latest': {
      const av = pageOf(a)?.version ?? ''
      const bv = pageOf(b)?.version ?? ''
      primary = missingLast(!av, !bv, dir, cmpVersion(av, bv))
      break
    }
    case 'uniqueId':
      primary = missingLast(
        !a.mod.uniqueId,
        !b.mod.uniqueId,
        dir,
        cmpText(a.mod.uniqueId, b.mod.uniqueId),
      )
      break
    case 'author':
      primary = missingLast(!a.mod.author, !b.mod.author, dir, cmpText(a.mod.author, b.mod.author))
      break
    case 'source':
      primary = missingLast(!a.source, !b.source, dir, cmpText(a.source, b.source))
      break
    case 'category': {
      const av = a.categoryLabel
      const bv = b.categoryLabel
      primary = missingLast(!av, !bv, dir, cmpText(av, bv))
      break
    }
    case 'endorsements': {
      const av = pageOf(a)?.endorsements ?? a.mod.endorsements
      const bv = pageOf(b)?.endorsements ?? b.mod.endorsements
      const aM = isMissing(pageOf(a)) && a.mod.endorsements === 0
      const bM = isMissing(pageOf(b)) && b.mod.endorsements === 0
      primary = missingLast(aM, bM, dir, cmpNum(av, bv))
      break
    }
    case 'downloads': {
      const av = pageOf(a)?.downloads
      const bv = pageOf(b)?.downloads
      primary = missingLast(isMissing(av), isMissing(bv), dir, cmpNum(av ?? 0, bv ?? 0))
      break
    }
    case 'updated': {
      const av = Date.parse(pageOf(a)?.updated ?? '')
      const bv = Date.parse(pageOf(b)?.updated ?? '')
      primary = missingLast(Number.isNaN(av), Number.isNaN(bv), dir, cmpNum(av, bv))
      break
    }
    case 'installed': {
      const av = Date.parse(a.added)
      const bv = Date.parse(b.added)
      primary = missingLast(yearMissing(av), yearMissing(bv), dir, cmpNum(av, bv))
      break
    }
    case 'needs': {
      const an = a.mod.needs
      const bn = b.mod.needs
      primary = missingLast(
        isMissing(an),
        isMissing(bn),
        dir,
        cmpNum(an?.length ?? 0, bn?.length ?? 0),
      )
      break
    }
    case 'status':
      primary = missingLast(!a.status, !b.status, dir, cmpText(a.status, b.status))
      break
    case 'notes': {
      const av = notesText(a)
      const bv = notesText(b)
      primary = missingLast(!av, !bv, dir, cmpText(av, bv))
      break
    }
    case 'lastRun': {
      const av = lastRunCounts(a.mod)
      const bv = lastRunCounts(b.mod)
      primary = missingLast(!av, !bv, dir, cmpNum(av?.errors ?? 0, bv?.errors ?? 0))
      if (primary === 0 && av && bv) {
        primary = missingLast(false, false, dir, cmpNum(av.warnings, bv.warnings))
      }
      break
    }
    default:
      primary = 0
  }
  if (primary !== 0) {
    return primary
  }
  return cmpText(a.mod.name, b.mod.name)
}

function sortListRows(rows: readonly ListRow[], sort: ListColumnSort): ListRow[] {
  return rows
    .map((row, i) => ({ row, i }))
    .sort((x, y) => {
      const c = compareListRows(x.row, y.row, sort)
      return c === 0 ? x.i - y.i : c
    })
    .map((x) => x.row)
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

function persistColumns(ids: ListColumnId[]) {
  useSettings.setState({ listColumns: ids })
  SetListColumns(ids).catch(reportUnexpected)
}

export type { ListColumnId, ListColumnSort, ListRow, ListSortDir }
export {
  columnMenuFromEvent,
  compareListRows,
  DEFAULT_LIST_COLUMN_SORT,
  DEFAULT_VISIBLE_LIST_COLUMNS,
  LIST_COLUMN_IDS,
  LIST_COLUMN_WIDTH,
  LOCKED_LIST_COLUMNS,
  listGridColumns,
  moveListColumn,
  NARROW_HIDE_LIST_COLUMNS,
  nextListSort,
  persistColumns,
  sanitizeListColumns,
  sanitizeListSort,
  sortListRows,
  toggleListColumn,
  visibleListColumns,
}
