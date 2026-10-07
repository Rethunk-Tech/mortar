import { expect, test } from 'bun:test'

import { skeletonCount } from './skeletonCount.ts'

test('the grid skeleton fills columns by rows and the list one column', () => {
  expect(skeletonCount({ width: 1400, height: 600, grid: true, rowPx: 96 })).toBe(4 * 6)
  expect(skeletonCount({ width: 1400, height: 600, grid: false, rowPx: 64 })).toBe(9)
  expect(skeletonCount({ width: 0, height: 0, grid: true, rowPx: 96 })).toBe(1)
})
