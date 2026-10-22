import { expect, test } from 'bun:test'

import { clampPage, pageKeeping, screenSize, screenSpan } from './browseState.ts'

test('a page clamps to the first and last page', () => {
  expect(clampPage({ page: 9, total: 27 })).toBe(2)
  expect(clampPage({ page: 0, total: 27 })).toBe(1)
  expect(clampPage({ page: 2, total: 0 })).toBe(1)
  expect(clampPage({ page: 9, total: 100, size: 28 })).toBe(4)
})

test('a screen is whole rows of what fits, and at least one source page', () => {
  expect(screenSize(4, 6)).toBe(24)
  expect(screenSize(7, 7)).toBe(49)
  expect(screenSize(7, 2)).toBe(21)
  expect(screenSize(1, 3)).toBe(20)
})

test('screens are cut from source pages without repeating or skipping a hit', () => {
  expect(screenSpan(1, 24)).toEqual({ first: 1, last: 2, skip: 0 })
  expect(screenSpan(2, 24)).toEqual({ first: 2, last: 3, skip: 4 })
  expect(screenSpan(3, 49)).toEqual({ first: 5, last: 8, skip: 18 })
  expect(screenSpan(2, 20)).toEqual({ first: 2, last: 2, skip: 0 })
  expect(pageKeeping(3, 20, 24)).toBe(2)
  expect(pageKeeping(2, 49, 20)).toBe(3)
})

test('a hit that must stay in view picks the screen holding it at the new size', () => {
  // The 20th hit of screen 2 (size 24) is hit 44, which a screen of 18 puts on screen 3, not on the one starting at hit 24.
  expect(pageKeeping(2, 24, 18, 19)).toBe(3)
  expect(pageKeeping(2, 24, 18)).toBe(2)
  expect(pageKeeping(1, 24, 30, 5)).toBe(1)
  expect(pageKeeping(3, 20, 40, 15)).toBe(2)
})
