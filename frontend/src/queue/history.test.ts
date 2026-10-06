import { expect, test } from 'bun:test'
import {
  emptyFilters,
  filterHistory,
  type HistoryEntry,
  historyProfileName,
  historyProfiles,
  makeGen,
} from './history.ts'

const row = (over: Partial<HistoryEntry>): HistoryEntry => ({
  name: 'A',
  version: '1.0',
  source: 'nexus',
  profileId: 'p1',
  batchId: '',
  game: 'stardew',
  modId: 0,
  fileId: 0,
  kind: 'install',
  size: 100,
  started: 1,
  finished: 2,
  outcome: 'done',
  ...over,
})

test('filterHistory keeps every row when filters are empty', () => {
  const rows = [row({}), row({ outcome: 'failed', profileId: 'p2' })]
  expect(filterHistory(rows, emptyFilters())).toEqual(rows)
})

test('filterHistory matches outcome and profile', () => {
  const rows = [
    row({ name: 'ok' }),
    row({ name: 'bad', outcome: 'failed' }),
    row({ name: 'other', profileId: 'p2' }),
  ]
  expect(
    filterHistory(rows, { outcome: 'failed', profileId: '', batchId: '' }).map((e) => e.name),
  ).toEqual(['bad'])
  expect(
    filterHistory(rows, { outcome: '', profileId: 'p2', batchId: '' }).map((e) => e.name),
  ).toEqual(['other'])
  expect(
    filterHistory(rows, { outcome: 'done', profileId: 'p2', batchId: '' }).map((e) => e.name),
  ).toEqual(['other'])
  expect(filterHistory(rows, { outcome: 'failed', profileId: 'p2', batchId: '' })).toEqual([])
})

test('filterHistory matches an import batch', () => {
  const rows = [row({ batchId: 'batch-1' }), row({ batchId: 'batch-2' })]
  expect(
    filterHistory(rows, { outcome: '', profileId: '', batchId: 'batch-2' }).map((e) => e.batchId),
  ).toEqual(['batch-2'])
})

test('historyProfiles lists unique profile ids', () => {
  expect(historyProfiles([row({}), row({ profileId: 'p2' }), row({})])).toEqual(['p1', 'p2'])
})

test('makeGen ignores a result after clear or a newer load', () => {
  const gen = makeGen()
  const first = gen.stamp()
  const second = gen.stamp()
  expect(gen.is(first)).toBe(false)
  expect(gen.is(second)).toBe(true)
  gen.drop()
  expect(gen.is(second)).toBe(false)
})

test('a history profile is named from the open game, by its game when it is another one, or as deleted', () => {
  const entries = [
    { profileId: 'a', game: 'stardew' },
    { profileId: 'b', game: 'lethal-company' },
    { profileId: 'gone', game: 'stardew' },
  ] as HistoryEntry[]
  const profiles = [{ id: 'a', name: 'Farm' }]
  expect(historyProfileName('a', entries, 'stardew', profiles)).toBe('Farm')
  expect(historyProfileName('b', entries, 'stardew', profiles)).toEqual({ game: 'lethal-company' })
  expect(historyProfileName('gone', entries, 'stardew', profiles)).toBeNull()
})
