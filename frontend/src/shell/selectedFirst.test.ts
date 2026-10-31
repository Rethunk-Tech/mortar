import { expect, test } from 'bun:test'
import { selectedFirst } from './selectedFirst.ts'

test('picked options come first, each group keeping its order', () => {
  const picked = new Set(['Misc', 'BepInEx', 'Equipment'])
  expect(
    selectedFirst(['Mods', 'BepInEx', 'Items', 'Equipment', 'Misc'], (o) => picked.has(o)),
  ).toEqual(['BepInEx', 'Equipment', 'Misc', 'Mods', 'Items'])
})
