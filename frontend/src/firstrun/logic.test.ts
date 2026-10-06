import { expect, test } from 'bun:test'
import { loaderChip, shouldShowFirstRun } from './logic.ts'

test('setup shows for a missing game and stays until the game has a profile', () => {
  const ready = { installed: true, profileCount: 1 }
  expect(shouldShowFirstRun(ready)).toBe(false)
  expect(shouldShowFirstRun({ ...ready, installed: false })).toBe(true)
  expect(shouldShowFirstRun({ ...ready, profileCount: 0 })).toBe(true)
})

test('the loader step is ticked only once the loader is installed', () => {
  expect(loaderChip(2, 3, false)).toBe('todo')
  expect(loaderChip(3, 3, false)).toBe('current')
  expect(loaderChip(4, 3, true)).toBe('done')
  expect(loaderChip(4, 3, false)).toBe('todo')
})
