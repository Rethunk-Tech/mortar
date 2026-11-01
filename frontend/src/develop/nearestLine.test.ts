import { expect, test } from 'bun:test'
import { nearestLine } from './nearestLine.ts'

test('nearestLine prefers the line itself, then below, then above, within reach', () => {
  const linked = new Set([5, 9])
  const has = (n: number) => linked.has(n)
  expect(nearestLine(5, 20, has)).toBe(5)
  // A signature on line 3: its first statement is below.
  expect(nearestLine(3, 20, has)).toBe(5)
  // Equal distance: below wins.
  expect(nearestLine(7, 20, has)).toBe(9)
  expect(nearestLine(12, 20, has)).toBe(9)
  expect(nearestLine(5, 20, () => false)).toBe(-1)
})
