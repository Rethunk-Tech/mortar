import { expect, test } from 'bun:test'
import { emptyFilters, filterHistory, type HistoryEntry, historyProfiles } from './history.ts'

const row = (over: Partial<HistoryEntry>): HistoryEntry => ({
  name: 'A',
  version: '1.0',
  source: 'nexus',
  profileId: 'p1',
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
  expect(filterHistory(rows, { outcome: 'failed', profileId: '' }).map((e) => e.name)).toEqual([
    'bad',
  ])
  expect(filterHistory(rows, { outcome: '', profileId: 'p2' }).map((e) => e.name)).toEqual([
    'other',
  ])
  expect(filterHistory(rows, { outcome: 'done', profileId: 'p2' }).map((e) => e.name)).toEqual([
    'other',
  ])
  expect(filterHistory(rows, { outcome: 'failed', profileId: 'p2' })).toEqual([])
})

test('historyProfiles lists unique profile ids', () => {
  expect(historyProfiles([row({}), row({ profileId: 'p2' }), row({})])).toEqual(['p1', 'p2'])
})
