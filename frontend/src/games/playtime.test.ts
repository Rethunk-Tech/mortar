import { expect, test } from 'bun:test'
import { formatPlaytime } from './playtime.ts'

test('formatPlaytime', () => {
  expect(formatPlaytime(0)).toBe('')
  expect(formatPlaytime(59_000)).toBe('')
  expect(formatPlaytime(45 * 60_000)).toBe('45 min')
  expect(formatPlaytime(150 * 60_000)).toBe('2.5 hr')
  expect(formatPlaytime(12.4 * 3_600_000)).toBe('12 hr')
})
