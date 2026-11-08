import { expect, test } from 'bun:test'
import { listNames } from './list.ts'

test('names the caller left out count once in the "and N more" tail', () => {
  expect(listNames(['A', 'B', 'C', 'D', 'E'], 3, 93)).toBe('A, B, C, and 90 more')
  expect(listNames(['A', 'B'])).toBe('A and B')
})

test('Infinity keeps every name with a locale conjunction, not a comma join', () => {
  const names = ['Ana', 'Bo', 'Cy', 'Di']
  expect(listNames(names)).toBe('Ana, Bo, Cy, and 1 more')
  expect(names.join(', ')).toBe('Ana, Bo, Cy, Di')
  expect(listNames(names, Number.POSITIVE_INFINITY)).toBe('Ana, Bo, Cy, and Di')
})
