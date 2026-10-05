import { expect, test } from 'bun:test'
import type {
  StartupMod,
  StartupReport,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import {
  foldMods,
  modTotal,
  phaseSegments,
  slowestEvent,
  slowStartups,
  startupRegressions,
} from './startupView.ts'

const mod = (
  id: string,
  eventMs: Record<string, number>,
  rest: Partial<StartupMod> = {},
): StartupMod =>
  ({
    id,
    name: id,
    entryMs: 0,
    assetMs: 0,
    loadMs: 0,
    sampleMs: 0,
    eventMs,
    packs: [],
    ...rest,
  }) as StartupMod

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

test('a mod with little bridge time but a large sampled time is not folded', () => {
  const { shown } = foldMods([
    mod('patcher', { GameLaunched: 4 }, { sampleMs: 900 }),
    mod('small', { GameLaunched: 20 }),
  ])
  expect(shown.map((m) => m.id)).toEqual(['patcher'])
})

test('regressions name updated or new mods that added a second or more', () => {
  const report = (mods: StartupMod[]) => ({ mods }) as StartupReport
  const previous = report([
    mod('fs', { GameLaunched: 2900 }, { version: '7.0.0' }),
    mod('cp', { UpdateTicked: 14_000 }, { version: '2.9.1' }),
    mod('wol', { GameLaunched: 100 }, { version: '1.0.0' }),
  ])
  const latest = report([
    mod('fs', { GameLaunched: 8900 }, { version: '7.1.0' }),
    mod('cp', { UpdateTicked: 20_000 }, { version: '2.9.1' }),
    mod('wol', { GameLaunched: 600 }, { version: '1.1.0' }),
    mod('new', { GameLaunched: 1500 }, { version: '1.0.0' }),
  ])
  expect(startupRegressions(latest, previous)).toEqual([
    { name: 'fs', version: '7.1.0', addedMs: 6000 },
    { name: 'new', version: '1.0.0', addedMs: 1500 },
  ])
  expect(startupRegressions(latest, undefined)).toEqual([])
})

test('slow startup names plain mods and slow packs, never the framework that loads them', () => {
  const report = {
    mods: [
      mod('cp', { UpdateTicked: 14_000 }, {
        loadMs: 11_000,
        packs: [
          { id: 'rsv', name: 'RSV', assetMs: 0, loadMs: 1200, ms: 1200 },
          { id: 'small', name: 'Small', assetMs: 0, loadMs: 300, ms: 300 },
        ],
      } as Partial<StartupMod>),
      mod('fs', { GameLaunched: 3100 }),
      mod('quick', { GameLaunched: 900 }),
    ],
  } as StartupReport
  expect(slowStartups(report).map((s) => [s.kind, s.id])).toEqual([
    ['mod', 'fs'],
    ['pack', 'rsv'],
  ])
})
