import { beforeEach, expect, test } from 'bun:test'
import { State } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { applyLive, clearLiveUnlessRunning, useLive } from './live.ts'

beforeEach(() => clearLiveUnlessRunning(idle(useLive.getState().game)))

function idle(game: string) {
  return { game, state: State.Idle }
}

const run = {
  game: 'lethal-company',
  profile: 'a',
  scene: 'MainMenu',
  mods: [
    { id: 'thunderstore:Ns-Fixture', loaded: true },
    { id: 'thunderstore:Ns-Broken', loaded: false },
  ],
}

test('a live report badges each package loaded or not, and a stopped game clears it', () => {
  applyLive(run)
  expect(useLive.getState().scene).toBe('MainMenu')
  expect(useLive.getState().loaded).toEqual({
    'thunderstore:ns-fixture': true,
    'thunderstore:ns-broken': false,
  })
  clearLiveUnlessRunning(idle('stardew'))
  expect(useLive.getState().scene).toBe('MainMenu')
  clearLiveUnlessRunning(idle('lethal-company'))
  expect(useLive.getState()).toEqual({ game: '', profile: '', scene: '', loaded: {} })
})
