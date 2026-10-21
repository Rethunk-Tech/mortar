import { expect, test } from 'bun:test'
import { gameToFollow } from './followLoadedGame.ts'

test('a profile loaded for another game than the open one moves the route to it', () => {
  expect(gameToFollow({ name: 'game', game: 'gungeon' }, 'lethal-company')).toBe('lethal-company')
  expect(gameToFollow({ name: 'profiles', game: 'gungeon' }, 'lethal-company')).toBe(
    'lethal-company',
  )
})

test('the same game, or a route that is not on a game, stays put', () => {
  expect(gameToFollow({ name: 'game', game: 'gungeon' }, 'gungeon')).toBeNull()
  expect(gameToFollow({ name: 'game-select' }, 'gungeon')).toBeNull()
  expect(gameToFollow({ name: 'game-setup', game: 'valheim' }, 'gungeon')).toBeNull()
  expect(gameToFollow({ name: 'game', game: 'gungeon' }, undefined)).toBeNull()
})
