import { expect, test } from 'bun:test'
import { useNav } from './store.ts'

test('route switches between game select and a game', () => {
  expect(useNav.getState().route).toEqual({ name: 'game-select' })
  useNav.getState().openGame('stardew')
  expect(useNav.getState().route).toEqual({ name: 'game', game: 'stardew' })
  useNav.getState().openGameSelect()
  expect(useNav.getState().route).toEqual({ name: 'game-select' })
})
