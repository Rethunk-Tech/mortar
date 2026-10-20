import { expect, test } from 'bun:test'
import { groupStart, savedGroupOf } from './newProfileStart.ts'

test('a saved group start round-trips and other starts name no group', () => {
  expect(savedGroupOf(groupStart('g1'))).toBe('g1')
  expect(savedGroupOf('')).toBeNull()
  expect(savedGroupOf('My template')).toBeNull()
  expect(savedGroupOf(groupStart(''))).toBeNull()
})
