import { expect, test } from 'bun:test'
import { driftEntryName } from './describe.ts'

const mods = [
  { key: 'Mods/Alpha', name: 'Alpha' },
  { key: 'Mods/Pack', name: 'Pack One' },
  { key: 'Mods/Pack', name: 'Pack Two' },
]

test('a drift entry is named by the mods installed under it, else by its own key', () => {
  expect(driftEntryName(mods, 'Mods/Alpha')).toBe('Alpha')
  expect(driftEntryName(mods, 'Mods/Pack')).toBe('Pack One, Pack Two')
  expect(driftEntryName(mods, 'Mods/Unknown')).toBe('Mods/Unknown')
})
