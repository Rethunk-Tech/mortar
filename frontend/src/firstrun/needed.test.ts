import { afterEach, expect, test } from 'bun:test'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { gameSetupNeeded } from './needed.ts'

const realFetch = globalThis.fetch
afterEach(() => {
  globalThis.fetch = realFetch
})

test('a game opens its setup until its folder is known and it has a profile', async () => {
  let asked = 0
  globalThis.fetch = (() => {
    asked += 1
    return Promise.resolve(Response.json([{ id: 'p' }]))
  }) as unknown as typeof fetch
  expect(await gameSetupNeeded({ id: 'a', installed: false } as GameInfo)).toBe(true)
  expect(asked).toBe(0)
  expect(await gameSetupNeeded({ id: 'b', installed: true } as GameInfo)).toBe(false)
  globalThis.fetch = (() => Promise.resolve(Response.json(null))) as unknown as typeof fetch
  expect(await gameSetupNeeded({ id: 'c', installed: true } as GameInfo)).toBe(true)
})
