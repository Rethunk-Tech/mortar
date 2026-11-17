import { expect, test } from 'bun:test'
import { checkLoaderNames, type Game, loaderFamilies, playableGames } from './games.ts'

test('the catalog names every loader the site lists', () => {
  expect(playableGames().some((g) => (g.loaders ?? []).length > 0)).toBe(true)
  expect(loaderFamilies()).toEqual(expect.arrayContaining(['SMAPI', 'BepInEx']))
})

test('a loader with no name fails the build naming its game and loader', () => {
  const games = [
    { id: 'a-game', name: 'A Game', enabled: true, loaders: [{ id: 'folder' }] },
  ] as unknown as Game[]
  expect(() => checkLoaderNames(games)).toThrow(/"a-game".*"folder"/)
})
