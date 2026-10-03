import { expect, test } from 'bun:test'
import { collectionHeader } from './collectionHeader.ts'

test('no collection link', () => {
  expect(collectionHeader(null)).toEqual({ line: null, review: null })
  expect(collectionHeader({ linked: false, name: '', revision: 0, latest: 0, url: '' })).toEqual({
    line: null,
    review: null,
  })
})

test('same revision has no review button', () => {
  expect(
    collectionHeader({
      linked: true,
      name: 'Cozy Farm',
      revision: 3,
      latest: 3,
      url: 'https://www.nexusmods.com/games/stardewvalley/collections/cozy-farm',
    }),
  ).toEqual({
    line: {
      name: 'Cozy Farm',
      revision: 3,
      url: 'https://www.nexusmods.com/games/stardewvalley/collections/cozy-farm',
    },
    review: null,
  })
})

test('newer revision offers review', () => {
  expect(
    collectionHeader({
      linked: true,
      name: 'Cozy Farm',
      revision: 3,
      latest: 5,
      url: 'https://www.nexusmods.com/games/stardewvalley/collections/cozy-farm',
    }),
  ).toEqual({
    line: {
      name: 'Cozy Farm',
      revision: 3,
      url: 'https://www.nexusmods.com/games/stardewvalley/collections/cozy-farm',
    },
    review: 5,
  })
})
