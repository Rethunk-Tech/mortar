import { expect, test } from 'bun:test'
import { fitChips } from './fitChips.ts'

test('all chips show when they fit, else the last slot is left for +N', () => {
  expect(fitChips([50, 50, 50], 200, 4, 30)).toBe(3)
  // 50 + 4 + 50 = 104, + 4 + 30 = 138 <= 150; a third chip would not leave room for +N.
  expect(fitChips([50, 50, 50], 150, 4, 30)).toBe(2)
  expect(fitChips([50, 50, 50], 60, 4, 30)).toBe(0)
  expect(fitChips([], 100, 4, 30)).toBe(0)
})
