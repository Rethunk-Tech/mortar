import { expect, test } from 'bun:test'
import type { Settings } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { gamePrefs } from './gamePrefs.ts'

test('SMAPI builds and console come from the loader scope, with or without a game block', () => {
  const loader = { smapiBuilds: 'include', showSmapiConsole: false }
  expect(gamePrefs(loader as unknown as Settings, 'stardew')).toMatchObject(loader)
  const withGame = { ...loader, games: { stardew: { runsKept: 3 } } }
  expect(gamePrefs(withGame as unknown as Settings, 'stardew')).toMatchObject({
    ...loader,
    runsKept: 3,
  })
})
