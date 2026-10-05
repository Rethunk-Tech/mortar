import { expect, test } from 'bun:test'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { directProfile } from './route.ts'

const open = { id: 'p1', name: 'Main' } as Profile

test('a link goes straight into the open profile on the game screen', () => {
  expect(
    directProfile({ name: 'game', game: 'stardew' }, 'stardew', {
      game: 'stardew',
      openId: 'p1',
      profiles: [open],
    }),
  ).toBe(open)
})

test('other screens ask the user', () => {
  const route = { name: 'profiles', game: 'stardew' } as const
  expect(
    directProfile(route, 'stardew', { game: 'stardew', openId: 'p1', profiles: [open] }),
  ).toBeNull()
  expect(
    directProfile({ name: 'game-select' }, 'stardew', {
      game: undefined,
      openId: '',
      profiles: [],
    }),
  ).toBeNull()
})

test('a game screen without an open profile asks the user', () => {
  expect(
    directProfile({ name: 'game', game: 'stardew' }, 'stardew', {
      game: 'stardew',
      openId: 'gone',
      profiles: [open],
    }),
  ).toBeNull()
})
