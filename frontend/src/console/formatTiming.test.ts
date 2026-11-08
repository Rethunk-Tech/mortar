import { expect, test } from 'bun:test'
import { percentLabel } from './formatTiming.ts'

test('percentLabel follows the locale decimal, unlike toFixed', () => {
  const share = 0.123
  expect(`${(share * 100).toFixed(1)}%`).toBe('12.3%')
  expect(percentLabel(share, 'en')).toBe('12.3%')
  expect(percentLabel(share, 'de')).toBe('12,3%')
})
