import { expect, test } from 'bun:test'

import { clampPage } from './browseState.ts'

test('a page clamps to the first and last page', () => {
  expect(clampPage({ page: 9, total: 27 })).toBe(2)
  expect(clampPage({ page: 0, total: 27 })).toBe(1)
  expect(clampPage({ page: 2, total: 0 })).toBe(1)
})
