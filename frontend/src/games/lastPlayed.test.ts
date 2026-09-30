import { expect, test } from 'bun:test'
import { relativePlay } from './lastPlayed.ts'

const t0 = Date.parse('2026-09-30T16:00:00Z')

test('relativePlay rejects a bad timestamp', () => {
  expect(relativePlay('nope', t0)).toBeNull()
})

test('relativePlay uses now, minutes, hours and days', () => {
  expect(relativePlay('2026-09-30T16:00:00Z', t0)).toEqual({ kind: 'now' })
  expect(relativePlay('2026-09-30T15:50:00Z', t0)).toEqual({ kind: 'unit', n: 10, unit: 'minute' })
  expect(relativePlay('2026-09-30T13:00:00Z', t0)).toEqual({ kind: 'unit', n: 3, unit: 'hour' })
  expect(relativePlay('2026-09-28T16:00:00Z', t0)).toEqual({ kind: 'unit', n: 2, unit: 'day' })
})
