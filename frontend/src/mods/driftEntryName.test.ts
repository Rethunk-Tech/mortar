import { expect, test } from 'bun:test'
import { driftEntryName } from './describe.ts'

test('a drift entry reads as the names of the mods it installed, or its key when none are known', () => {
  const mods = [
    { key: 'pack', name: 'Alpha' },
    { key: 'pack', name: 'Beta' },
    { key: 'other', name: 'Gamma' },
  ]
  expect(driftEntryName(mods, 'pack')).toBe('Alpha, Beta')
  expect(driftEntryName(mods, 'gone')).toBe('gone')
})
