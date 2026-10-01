import { expect, test } from 'bun:test'
import type {
  Missing,
  Result,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import {
  andList,
  missingRequired,
  offersFor,
  requiredUniqueIds,
  stillMissing,
  wantOf,
  wantsOf,
} from './missingDeps.ts'

const profile = (...ids: string[]) => ({
  entries: [{ mods: ids.map((uniqueId) => ({ uniqueId })) }],
})

const ref = (over: Partial<NonNullable<Missing['where']>> = {}): NonNullable<Missing['where']> => ({
  site: 'Nexus',
  github: '',
  pageId: 541,
  pageName: 'Content Patcher',
  url: 'https://nexusmods.com/stardewvalley/mods/1915',
  fileId: 1,
  fileName: 'CP.zip',
  version: '2.0.0',
  ...over,
})

const absent = (over: Partial<Missing> = {}): Missing => ({
  dependentId: 'A.Mod',
  dependentName: 'A Mod',
  uniqueId: 'Pathoschild.ContentPatcher',
  minimumVersion: '',
  reason: 'absent',
  installedVersion: '',
  where: ref(),
  ...over,
})

const result = (missing: Missing[]): Result => ({
  missing,
  duplicates: null,
  broken: null,
  assetConflicts: null,
  runErrors: null,
  unknown: false,
})

test('required deps skip optional entries and include ContentPackFor', () => {
  expect(
    requiredUniqueIds({
      UniqueID: 'A.Pack',
      Dependencies: [
        { UniqueID: 'Need.This', IsRequired: true },
        { UniqueID: 'Nice.ToHave', IsRequired: false },
        { UniqueID: 'Also.Default' },
      ],
      ContentPackFor: { UniqueID: 'Pathoschild.ContentPatcher' },
    }),
  ).toEqual(['Need.This', 'Also.Default', 'Pathoschild.ContentPatcher'])
})

test('missing required deps are those the profile does not already have', () => {
  const manifest = {
    UniqueID: 'A.Pack',
    Dependencies: [
      { UniqueID: 'Need.This', IsRequired: true },
      { UniqueID: 'Nice.ToHave', IsRequired: false },
    ],
    ContentPackFor: { UniqueID: 'Pathoschild.ContentPatcher' },
  }
  expect(missingRequired(manifest, profile('A.Pack'))).toEqual([
    'Need.This',
    'Pathoschild.ContentPatcher',
  ])
  expect(
    missingRequired(manifest, profile('a.pack', 'need.this', 'Pathoschild.ContentPatcher')),
  ).toEqual([])
})

test('optional-only manifests are not offered', () => {
  expect(
    missingRequired(
      { UniqueID: 'A.Mod', Dependencies: [{ UniqueID: 'Maybe', IsRequired: false }] },
      profile('A.Mod'),
    ),
  ).toEqual([])
})

test('offersFor uses Problems absent rows for the installed dependents', () => {
  expect(
    offersFor(
      ['A.Mod'],
      result([
        absent(),
        absent({
          dependentId: 'A.Mod',
          uniqueId: 'Off.Mod',
          reason: 'disabled',
          where: null,
        }),
        absent({ dependentId: 'Other.Mod', dependentName: 'Other', uniqueId: 'Skip.Me' }),
      ]),
    ),
  ).toEqual([{ dependentName: 'A Mod', missing: [absent()] }])
})

test('Add them queues through the same Want the Problems bar uses', () => {
  expect(wantOf(absent({ where: null }))).toBeNull()
  expect(wantsOf([absent(), absent({ uniqueId: 'Other.CP' })])).toEqual([
    {
      kind: 'dependency',
      modId: 541,
      fileId: 1,
      name: 'Content Patcher',
      fileName: 'CP.zip',
      version: '2.0.0',
    },
  ])
  expect(andList(['Content Patcher', 'Lookup Anything'])).toBe(
    'Content Patcher and Lookup Anything',
  )
})

test('an offer drops deps the live problems no longer list', () => {
  const meep = absent({ uniqueId: 'Spiderbuttons.MEEP' })
  const core = absent({ uniqueId: 'spacechase0.SpaceCore' })
  const offer = { dependentName: 'A Mod', missing: [meep, core] }
  expect(stillMissing(offer, null)).toEqual([meep, core])
  expect(stillMissing(offer, result([core]))).toEqual([core])
  expect(stillMissing(offer, result([]))).toEqual([])
})
