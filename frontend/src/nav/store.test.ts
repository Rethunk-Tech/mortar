import { expect, test } from 'bun:test'
import { useNav } from './store.ts'

test('route switches between game select and a game', () => {
  // bun runs every test file in one process, so another file may have navigated already.
  useNav.setState(useNav.getInitialState(), true)
  expect(useNav.getState().route).toEqual({ name: 'game-select' })
  useNav.getState().openGame('stardew')
  expect(useNav.getState().route).toEqual({ name: 'game', game: 'stardew' })
  useNav.getState().openGameSelect()
  expect(useNav.getState().route).toEqual({ name: 'game-select' })
})

test('a setup remembers the game it was opened from', () => {
  useNav.setState(useNav.getInitialState(), true)
  useNav.getState().openGameSetup('valheim')
  expect(useNav.getState().route).toEqual({ name: 'game-setup', game: 'valheim' })
  useNav.getState().openGame('stardew')
  useNav.getState().openGameSetup('valheim')
  expect(useNav.getState().route).toEqual({ name: 'game-setup', game: 'valheim', from: 'stardew' })
})

test('opening the section already shown keeps the route, so nothing re-renders', () => {
  useNav.setState(useNav.getInitialState(), true)
  useNav.getState().openSettings('general')
  const open = useNav.getState().route
  useNav.getState().openSettings('general')
  useNav.getState().openSettings()
  expect(useNav.getState().route).toBe(open)
  useNav.getState().openSettings('about')
  expect(useNav.getState().route).toEqual({
    name: 'settings',
    section: 'about',
    back: { name: 'game-select' },
  })
})
