import { expect, test } from 'bun:test'
import { changesSinceCache } from './changesSinceCache.ts'

test('only the newest few answers are kept', () => {
  for (let i = 0; i < 20; i++) {
    changesSinceCache.set(`k${i}`, null)
  }
  expect(changesSinceCache.has('k0')).toBe(false)
  expect(changesSinceCache.has('k11')).toBe(false)
  expect(changesSinceCache.has('k12')).toBe(true)
  expect(changesSinceCache.has('k19')).toBe(true)
})
