import { expect, test } from 'bun:test'
import type { State } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/netstate/models.ts'
import { savedAt, unreachable } from './offline.ts'

const down = (id: string, lastOK: string): State => ({
  id,
  unreachable: true,
  lastOK,
  lastFail: '2026-10-05T10:00:00Z',
  lastError: '',
  lastReason: '',
})

test('savedAt is the clock time of the last success, empty when there was none', () => {
  expect(savedAt(down('nexus', '2026-10-05T09:30:00Z'), 'en')).toMatch(/\d:\d\d/)
  expect(savedAt(down('nexus', '0001-01-01T00:00:00Z'), 'en')).toBe('')
})

test('unreachable filters to the named sources', () => {
  const states = [down('nexus', ''), { ...down('github', ''), unreachable: false }]
  expect(unreachable(states).map((s) => s.id)).toEqual(['nexus'])
  expect(unreachable(states, ['github'])).toEqual([])
})
