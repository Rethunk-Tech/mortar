import { expect, test } from 'bun:test'
import type { StartupMod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { foldMods, modTotal, phaseSegments, slowestEvent } from './startupView.ts'

const mod = (
  id: string,
  eventMs: Record<string, number>,
  rest: Partial<StartupMod> = {},
): StartupMod =>
  ({ id, name: id, entryMs: 0, assetMs: 0, loadMs: 0, eventMs, packs: [], ...rest }) as StartupMod

test('phases run back to back and skip one the bridge did not see', () => {
  const segs = phaseSegments({
    bridgeEntry: 10_452,
    entryDone: 0,
    gameLaunched: 20_910,
    titleMenu: 57_108,
    titleScreen: 57_108,
  })
  expect(segs).toEqual([
    { id: 'smapi', ms: 10_452 },
    { id: 'content', ms: 10_458 },
    { id: 'firstTicks', ms: 36_198 },
  ])
  expect(segs.reduce((n, s) => n + s.ms, 0)).toBe(57_108)
})

test('mods under the fold line collapse into one row; the rest sort slowest first', () => {
  const { shown, folded } = foldMods([
    mod('quiet', { Rendered: 3 }),
    mod('fs', { GameLaunched: 3546 }, { entryMs: 207 }),
    mod('cp', { UpdateTicked: 14_158 }, { loadMs: 11_217, assetMs: 566 }),
    mod('small', { GameLaunched: 20 }),
  ])
  expect(shown.map((m) => m.id)).toEqual(['cp', 'fs'])
  expect(folded).toEqual({ count: 2, ms: 23 })
  expect(modTotal(shown[0] as StartupMod)).toBe(25_941)
  expect(slowestEvent(shown[1] as StartupMod)).toEqual(['GameLaunched', 3546])
})
