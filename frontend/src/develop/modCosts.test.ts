import { expect, test } from 'bun:test'
import { measured } from './modCosts.ts'

test('a run counts for the build it ran, not one written after it', () => {
  const built = Date.parse('2026-11-01T14:00:00Z')
  expect(measured('2026-11-01T14:05:00Z', built)).toBe(true)
  expect(measured('2026-11-01T13:55:00Z', built)).toBe(false)
  expect(measured('2026-11-01T13:55:00Z', 0)).toBe(true)
  expect(measured(undefined, built)).toBe(false)
})
