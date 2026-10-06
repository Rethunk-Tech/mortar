import { expect, test } from 'bun:test'
import type { BrowseItem } from './browseTypes.ts'
import { pickedItem } from './pickedItem.ts'

const card = {
  source: 'thunderstore',
  id: 'BepInEx-BepInExPack',
  url: 'ts',
  installed: false,
  obsolete: false,
  broken: false,
  loader: true,
  bundled: false,
  alts: [
    {
      source: 'nexus',
      id: '1',
      url: 'nx',
      installed: true,
      obsolete: true,
      broken: false,
      loader: true,
    },
  ],
} as BrowseItem

test('the picked source supplies its own id, state and flags', () => {
  expect(pickedItem(card, 'thunderstore')).toBe(card)
  const nexus = pickedItem(card, 'nexus')
  expect(nexus).toMatchObject({
    source: 'nexus',
    id: '1',
    url: 'nx',
    installed: true,
    obsolete: true,
  })
})

test('a merged loader card stays a loader card on every source', () => {
  expect(pickedItem(card, 'nexus').loader).toBe(true)
})
