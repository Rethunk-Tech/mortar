import { expect, test } from 'bun:test'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { useNav } from '../nav/store.ts'
import { openGame } from './useOpenGame.ts'

const game = (id: string) => ({ id, installed: true }) as GameInfo
const messages = { last: 'last', read: 'read' }
const deps = (needed: boolean, remembered: string[]) => ({
  remember: (id: string) => {
    remembered.push(id)
    return Promise.resolve()
  },
  setupNeeded: () => Promise.resolve(needed),
  onError: () => () => undefined,
})

test('choosing another game remembers it and opens it, or its setup when it has no profile', async () => {
  useNav.setState({ route: { name: 'game', game: 'stardew' } })
  const remembered: string[] = []
  await openGame(game('valheim'), messages, deps(false, remembered))
  expect(useNav.getState().route).toEqual({ name: 'game', game: 'valheim' })
  await openGame(game('lethal-company'), messages, deps(true, remembered))
  expect(useNav.getState().route).toEqual({
    name: 'game-setup',
    game: 'lethal-company',
    from: 'valheim',
  })
  expect(remembered).toEqual(['valheim', 'lethal-company'])
})

test('an id that is not a game slug opens nothing', async () => {
  useNav.setState({ route: { name: 'game-select' } })
  await openGame(game('../x'), messages, deps(false, []))
  expect(useNav.getState().route.name).toBe('game-select')
})
