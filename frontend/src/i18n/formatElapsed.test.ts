import { expect, test } from 'bun:test'
import { formatElapsed } from './formatElapsed.ts'

test('formatElapsed adds hours only past the hour', () => {
  expect(formatElapsed(59_000)).toBe('0:59')
  expect(formatElapsed(75 * 60_000)).toBe('1:15:00')
  expect(formatElapsed(3 * 3_600_000)).toBe('3:00:00')
  expect(formatElapsed(-5)).toBe('0:00')
})
