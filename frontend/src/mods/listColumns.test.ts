import { expect, test } from 'bun:test'
import type { Details } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useSettings } from '../settings/store.ts'
import {
  compareListRows,
  DEFAULT_LIST_COLUMN_SORT,
  DEFAULT_VISIBLE_LIST_COLUMNS,
  type ListRow,
  moveListColumn,
  nextListSort,
  sanitizeListColumns,
  sanitizeListSort,
  sortListRows,
  toggleListColumn,
  visibleListColumns,
} from './listColumns.ts'

const mod = (over: Partial<Mod> & Pick<Mod, 'name'>): Mod => ({
  key: over.key ?? over.name,
  uniqueId: over.uniqueId ?? over.name,
  name: over.name,
  author: over.author ?? '',
  version: over.version ?? '',
  enabled: over.enabled ?? true,
  siblings: over.siblings ?? [],
  picture: over.picture ?? '',
  endorsements: over.endorsements ?? 0,
  needs: over.needs ?? [],
})

const row = (over: Partial<ListRow> & { name: string }): ListRow => {
  const out: ListRow = {
    mod: mod({ name: over.name, ...over.mod }),
    added: over.added ?? '',
    source: over.source ?? '',
    status: over.status ?? '',
    note: over.note ?? '',
    tags: over.tags ?? [],
  }
  if (over.details) {
    out.details = over.details
  }
  return out
}

const details = (over: {
  version?: string
  category?: string
  endorsements?: number
  downloads?: number
  updated?: string
}): Details =>
  ({
    category: over.category ?? '',
    files: null,
    changelogs: null,
    page: {
      modId: 1,
      name: '',
      summary: '',
      description: '',
      pictureUrl: '',
      version: over.version ?? '',
      author: '',
      uploadedBy: '',
      uploaderUrl: '',
      categoryId: 0,
      endorsements: over.endorsements ?? 0,
      downloads: over.downloads ?? 0,
      uniqueDownloads: 0,
      created: '',
      updated: over.updated ?? '',
      adult: false,
      status: '',
      available: true,
    },
  }) as Details

test('resets settings list columns from getInitialState', () => {
  useSettings.setState({
    listColumns: ['on', 'name'],
    listSortColumn: 'author',
    listSortDir: 'desc',
  })
  useSettings.setState(useSettings.getInitialState(), true)
  expect(useSettings.getState().listColumns).toEqual([...DEFAULT_VISIBLE_LIST_COLUMNS])
  expect(useSettings.getState().listSortColumn).toBe(DEFAULT_LIST_COLUMN_SORT.column)
  expect(useSettings.getState().listSortDir).toBe(DEFAULT_LIST_COLUMN_SORT.dir)
})

test('sanitizes column ids and keeps On and Name', () => {
  expect(sanitizeListColumns([])).toEqual([...DEFAULT_VISIBLE_LIST_COLUMNS])
  expect(sanitizeListColumns(['nope'])).toEqual([...DEFAULT_VISIBLE_LIST_COLUMNS])
  expect(sanitizeListColumns(['version', 'name'])).toEqual(['on', 'version', 'name'])
  expect(toggleListColumn(['on', 'name', 'version'], 'on')).toEqual(['on', 'name', 'version'])
  expect(toggleListColumn(['on', 'name', 'version'], 'version')).toEqual(['on', 'name'])
  expect(toggleListColumn(['on', 'name'], 'author')).toEqual(['on', 'name', 'author'])
})

test('clicking a header sorts ascending, then reverses, then switches column', () => {
  const nameAsc = DEFAULT_LIST_COLUMN_SORT
  expect(nextListSort(nameAsc, 'on')).toEqual(nameAsc)
  const nameDesc = nextListSort(nameAsc, 'name')
  expect(nameDesc).toEqual({ column: 'name', dir: 'desc' })
  expect(nextListSort(nameDesc, 'name')).toEqual({ column: 'name', dir: 'asc' })
  expect(nextListSort(nameDesc, 'author')).toEqual({ column: 'author', dir: 'asc' })
  expect(sanitizeListSort('on', 'up')).toEqual(DEFAULT_LIST_COLUMN_SORT)
  expect(sanitizeListSort('version', 'desc')).toEqual({ column: 'version', dir: 'desc' })
})

test('hides low-priority columns below 960px even when shown', () => {
  expect(visibleListColumns(['on', 'name', 'author', 'status'], true)).toEqual([
    'on',
    'name',
    'status',
  ])
  expect(visibleListColumns(['on', 'name', 'author', 'status'], false)).toEqual([
    'on',
    'name',
    'author',
    'status',
  ])
})

test('compares numbers, versions, dates and text, with missing last in both directions', () => {
  const a = row({
    name: 'Beta',
    mod: mod({ name: 'Beta', version: '2.0.0', uniqueId: 'B', author: 'Ann' }),
    source: 'Nexus Mods',
    status: 'Enabled',
    added: '2026-01-02T00:00:00Z',
    details: details({
      version: '2.1.0',
      category: 'UI',
      endorsements: 10,
      downloads: 100,
      updated: '2026-02-01T00:00:00Z',
    }),
  })
  const b = row({
    name: 'Alpha',
    mod: mod({ name: 'Alpha', version: '10.0.0', uniqueId: 'A', author: 'Bob' }),
    source: 'Archive',
    status: 'Off',
    added: '2026-01-01T00:00:00Z',
    details: details({
      version: '10.0.0',
      category: 'Crops',
      endorsements: 2,
      downloads: 50,
      updated: '2026-01-01T00:00:00Z',
    }),
  })
  const missing = row({ name: 'Zed', mod: mod({ name: 'Zed', version: '', uniqueId: 'Z' }) })
  expect(compareListRows(a, b, { column: 'version', dir: 'asc' })).toBeLessThan(0)
  expect(compareListRows(a, b, { column: 'name', dir: 'asc' })).toBeGreaterThan(0)
  expect(compareListRows(a, b, { column: 'endorsements', dir: 'desc' })).toBeLessThan(0)
  expect(compareListRows(missing, a, { column: 'version', dir: 'asc' })).toBeGreaterThan(0)
  expect(compareListRows(missing, a, { column: 'version', dir: 'desc' })).toBeGreaterThan(0)
  expect(compareListRows(a, b, { column: 'installed', dir: 'asc' })).toBeGreaterThan(0)
  const sorted = sortListRows([a, missing, b], { column: 'name', dir: 'asc' })
  expect(sorted.map((r) => r.mod.name)).toEqual(['Alpha', 'Beta', 'Zed'])
})

test('keeps saved column order and moves a header', () => {
  const saved = ['on', 'author', 'name', 'status'] as const
  expect(visibleListColumns(saved, false)).toEqual(['on', 'author', 'name', 'status'])
  expect(moveListColumn(['on', 'name', 'author', 'status'], 3, 1)).toEqual([
    'on',
    'status',
    'name',
    'author',
  ])
  expect(moveListColumn(['on', 'name'], 0, 0)).toEqual(['on', 'name'])
})
