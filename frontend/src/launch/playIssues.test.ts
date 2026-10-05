import { expect, test } from 'bun:test'
import type { AssetConflict } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/framework/models.ts'
import type {
  Broken,
  Missing,
  Update,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { overflowIssueCount, playIssueSummary } from './playIssues.ts'

const missing = (over: Partial<Missing> = {}): Missing => ({
  dependentId: 'A.Mod',
  dependentName: 'A',
  id: 'Need.This',
  minimumVersion: '',
  reason: 'absent',
  installedVersion: '',
  listed: false,
  note: '',
  optional: false,
  where: null,
  ...over,
})

const conflict = (over: Partial<AssetConflict> = {}): AssetConflict => ({
  kind: 'load',
  target: 'maps/town',
  packIds: [],
  names: ['One', 'Two'],
  keys: [],
  winnerId: '',
  winnerName: '',
  overridden: [],
  cosmetic: false,
  fixes: [],
  info: '',
  evidence: [],
  ...over,
})

const broken = (over: Partial<Broken> = {}): Broken => ({
  key: 'k',
  id: 'B.Mod',
  name: 'Broke',
  status: 'broken',
  brokeIn: '',
  summary: '',
  ...over,
})

const update = (over: Partial<Update> = {}): Update => ({
  key: 'u',
  id: '',
  name: 'Newer',
  installed: '',
  version: '2.0.0',
  url: '',
  nexusId: 0,
  githubRepo: '',
  unofficial: false,
  source: '',
  ...over,
})

test('playIssueSummary is empty when nothing is wrong', () => {
  expect(playIssueSummary({})).toEqual([])
  expect(
    playIssueSummary({
      missing: [missing({ optional: true })],
      assetConflicts: [conflict({ cosmetic: true })],
      broken: [broken({ status: 'abandoned' })],
      updates: [],
      currentProfileId: 'p1',
      lastPlayed: { folder: 'Farm_1', farm: 'Sunny', profileId: 'p1', profileName: 'Main' },
    }),
  ).toEqual([])
})

test('playIssueSummary adds a last-profile group when the newest save used another profile', () => {
  expect(
    playIssueSummary({
      currentProfileId: 'p1',
      lastPlayed: { folder: 'Farm_1', farm: 'Sunny', profileId: 'p2', profileName: 'Co-op' },
    }),
  ).toEqual([
    {
      kind: 'lastProfile',
      count: 1,
      names: [],
      save: 'Sunny',
      profileName: 'Co-op',
      switchProfileId: 'p2',
    },
  ])
})

test('playIssueSummary names a missing requirement Unknown mod when the page has no name', () => {
  expect(
    playIssueSummary({
      missing: [missing({ id: 'Need.This', where: null })],
    }),
  ).toEqual([
    {
      kind: 'missing',
      count: 1,
      names: ['Unknown mod'],
      nameTitles: ['Need.This'],
    },
  ])
})

test('playIssueSummary groups required missing, non-cosmetic conflicts, updates, and broken or obsolete', () => {
  const names = ['A', 'B', 'C', 'D', 'E', 'F']
  expect(
    playIssueSummary({
      missing: [
        missing({
          id: 'SpaceCore',
          where: {
            site: 'Nexus',
            github: '',
            pageId: 1,
            pageName: 'SpaceCore',
            url: '',
            fileId: 0,
            fileName: '',
            version: '',
          },
        }),
      ],
      assetConflicts: [conflict({ names: ['SVE', 'Other'] })],
      broken: [
        broken({ name: 'Old', status: 'obsolete' }),
        broken({ name: 'Dead', status: 'abandoned' }),
      ],
      updates: names.map((name, i) => update({ name, key: String(i) })),
    }),
  ).toEqual([
    { kind: 'missing', count: 1, names: ['SpaceCore'] },
    { kind: 'conflicts', count: 1, names: ['SVE, Other'] },
    { kind: 'updates', count: 6, names: ['A', 'B', 'C', 'D', 'E'] },
    { kind: 'broken', count: 1, names: ['Old'] },
  ])
})

test('overflowIssueCount is the names past the five shown', () => {
  expect(overflowIssueCount({ count: 6, names: ['A', 'B', 'C', 'D', 'E'] })).toBe(1)
  expect(overflowIssueCount({ count: 2, names: ['A', 'B'] })).toBe(0)
})
