import { expect, test } from 'bun:test'
import { tipVisible } from './visible.ts'

test('a tip shows until its id is in the seen list', () => {
  expect(tipVisible(undefined, 'mods')).toBe(true)
  expect(tipVisible(null, 'console')).toBe(true)
  expect(tipVisible([], 'console')).toBe(true)
  expect(tipVisible(['mods'], 'mods')).toBe(false)
  expect(tipVisible(['mods'], 'share')).toBe(true)
})
