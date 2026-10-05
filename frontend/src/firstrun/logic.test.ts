import { expect, test } from 'bun:test'
import { shouldShowFirstRun } from './logic.ts'

test('first run shows only for a missing game or a first launch', () => {
  const ready = { installed: true, smapiInstalled: true, profileCount: 1 }
  expect(shouldShowFirstRun(ready)).toBe(false)
  expect(shouldShowFirstRun({ ...ready, installed: false })).toBe(true)
  expect(shouldShowFirstRun({ ...ready, smapiInstalled: false })).toBe(false)
  expect(shouldShowFirstRun({ ...ready, profileCount: 0 })).toBe(false)
  expect(shouldShowFirstRun({ ...ready, smapiInstalled: false, profileCount: 0 })).toBe(true)
})
