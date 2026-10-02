import { expect, test } from 'bun:test'
import { addedWithin, WEEK_MS } from './recent.ts'

test('addedWithin keeps the last week and drops older or missing dates', () => {
  const now = Date.parse('2026-10-02T12:00:00Z')
  expect(addedWithin('2026-09-30T12:00:00Z', WEEK_MS, now)).toBe(true)
  expect(addedWithin('2026-09-20T12:00:00Z', WEEK_MS, now)).toBe(false)
  expect(addedWithin('', WEEK_MS, now)).toBe(false)
  expect(addedWithin(undefined, WEEK_MS, now)).toBe(false)
})
