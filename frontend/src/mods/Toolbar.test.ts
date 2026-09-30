import { expect, test } from 'bun:test'
import { searchFieldOpen } from '../game/compact.ts'

test('compact search stays an icon until expanded or the query is set', () => {
  expect(searchFieldOpen(false, false, '')).toBe(true)
  expect(searchFieldOpen(true, false, '')).toBe(false)
  expect(searchFieldOpen(true, true, '')).toBe(true)
  expect(searchFieldOpen(true, false, 'smapi')).toBe(true)
})
