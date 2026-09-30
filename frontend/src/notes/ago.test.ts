import { expect, test } from 'bun:test'
import { ago } from './ago.ts'

test('ago picks the largest whole unit', () => {
  expect(ago(-5)).toEqual({ unit: 'now', n: 0 })
  expect(ago(59_999)).toEqual({ unit: 'now', n: 0 })
  expect(ago(60_000)).toEqual({ unit: 'minute', n: 1 })
  expect(ago(2 * 3_600_000 - 1)).toEqual({ unit: 'hour', n: 1 })
  expect(ago(3 * 86_400_000)).toEqual({ unit: 'day', n: 3 })
})
