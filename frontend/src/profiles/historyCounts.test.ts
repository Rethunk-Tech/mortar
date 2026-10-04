import { expect, test } from 'bun:test'
import type { HistoryEvent } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { historyChangeSummary, historyEventKind } from './historyCounts.ts'

const ev = (added = 0, removed = 0, updated = 0): HistoryEvent => ({
  id: 'e',
  at: '',
  kind: 'bulk',
  label: 'Changed mods',
  count: 0,
  added,
  removed,
  updated,
  from: '',
  to: '',
  snapshotId: '',
})

test('historyChangeSummary formats added, removed, and updated counts', () => {
  expect(historyChangeSummary(ev())).toBe('')
  expect(historyChangeSummary(ev(3, 1, 2))).toBe('+3 −1 ~2')
  expect(historyChangeSummary(ev(1, 0, 0))).toBe('+1')
})

test('historyEventKind recognizes a backend kind and ignores a sentence label', () => {
  expect(historyEventKind({ kind: 'added', label: 'added' })).toBe('added')
  expect(historyEventKind({ kind: 'added', label: 'Added SpaceCore' })).toBe('added')
})
