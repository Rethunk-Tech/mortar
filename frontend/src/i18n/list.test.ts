import { expect, test } from 'bun:test'
import { listNames } from './list.ts'

test('names the caller left out count once in the "and N more" tail', () => {
  expect(listNames(['A', 'B', 'C', 'D', 'E'], 3, 93)).toBe('A, B, C, and 90 more')
  expect(listNames(['A', 'B'])).toBe('A and B')
})
