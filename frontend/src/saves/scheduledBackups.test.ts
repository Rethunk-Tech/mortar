import { expect, test } from 'bun:test'
import { hoursUntilNext } from './scheduledBackups.ts'

test('next scheduled backup is counted in whole hours, never 0 while ahead', () => {
  const hour = 3_600_000
  expect(hoursUntilNext(0, 24, 20 * hour)).toBe(4)
  expect(hoursUntilNext(0, 24, 24 * hour - 1)).toBe(1)
  expect(hoursUntilNext(0, 24, 24 * hour)).toBe(0)
})
