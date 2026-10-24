import { afterEach, expect, test } from 'bun:test'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { gameName, reloadGames } from './info.ts'

const realFetch = globalThis.fetch
afterEach(() => {
  globalThis.fetch = realFetch
})

test('the catalog loads once, and names read "the game" until it answers', async () => {
  let asked = 0
  let answer = Promise.withResolvers<Response>()
  globalThis.fetch = (() => {
    asked += 1
    return answer.promise
  }) as unknown as typeof fetch
  useProfiles.setState({ game: { id: 'sims4', name: 'The Sims 4' } as GameInfo })
  reloadGames()
  expect(gameName('stardew')).toBe('the game')
  expect(gameName('sims4')).toBe('The Sims 4')
  expect(asked).toBe(1)
  answer.resolve(Response.json([{ id: 'stardew', name: 'Stardew Valley' }]))
  await answer.promise
  await new Promise((resolve) => setTimeout(resolve, 0))
  expect(gameName('stardew')).toBe('Stardew Valley')
  answer = Promise.withResolvers<Response>()
  reloadGames()
  expect(asked).toBe(2)
  answer.resolve(Response.json([]))
  await answer.promise
})
