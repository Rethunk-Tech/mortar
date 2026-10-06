import { expect, test } from 'bun:test'
import type { Result } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { problemCount } from './lookup.ts'
import { assetRows, problemSections, rowKeys } from './problemGroups.ts'

const emptyResult = (): Result => ({
  duplicates: [],
  broken: [],
  missing: [],
  assetConflicts: [],
  settings: [],
  runErrors: [],
  drift: [],
  dismissed: [],
  unknown: false,
})

test('problemSections omits empty groups and keeps order', () => {
  const result: Result = {
    ...emptyResult(),
    missing: [
      {
        dependentId: 'd',
        dependentName: 'Dep',
        id: 'a',
        minimumVersion: '',
        installedVersion: '',
        reason: 'absent',
        listed: false,
        note: '',
        optional: false,
        where: null,
      },
    ],
    duplicates: [
      {
        id: 'u',
        name: 'Dup',
        copies: null,
      },
    ],
  }
  const sections = problemSections(result)
  expect(sections.map((s) => s.id)).toEqual(['missing', 'duplicates'])
  expect(problemCount(result)).toBe(2)
})

test('problemCount is zero while result is null', () => {
  expect(problemCount(null)).toBe(0)
})

test('settings are an info-level group and count as problems', () => {
  const result: Result = {
    ...emptyResult(),
    settings: [
      {
        key: 'pack',
        id: 'Pack.Compat',
        name: 'Pack',
        field: 'Enabled',
        current: 'false',
        suggested: ['true'],
        for: ['Other.Mod'],
        forNames: ['Other'],
        description: '',
        variant: false,
        currentFor: '',
      },
    ],
  }
  expect(problemSections(result).map((s) => s.id)).toEqual(['settings'])
  expect(problemCount(result)).toBe(1)
})

test('cosmetic conflicts get their own section and are not counted', () => {
  const conflict = (target: string, cosmetic: boolean) => ({
    kind: 'edit',
    target,
    packIds: ['A', 'B'],
    names: ['A', 'B'],
    keys: ['a', 'b'],
    winnerId: '',
    winnerName: 'unclear',
    overridden: null,
    cosmetic,
    fixes: [],
    evidence: [],
  })
  const result: Result = {
    ...emptyResult(),
    assetConflicts: [conflict('maps/forest', false), conflict('loosesprites/cursors', true)],
  }
  const sections = problemSections(result)
  expect(sections.map((s) => [s.id, s.rows.length])).toEqual([
    ['conflicts', 1],
    ['cosmetic', 1],
  ])
  expect(problemCount(result)).toBe(1)
})

test('dismissed problems are listed but not counted', () => {
  const result: Result = {
    ...emptyResult(),
    dismissed: [
      {
        token: 'load\tmaps/greenhouse',
        assetConflict: {
          kind: 'load',
          target: 'maps/greenhouse',
          packIds: [],
          names: [],
          keys: [],
          winnerId: '',
          winnerName: '',
          overridden: [],
          cosmetic: false,
          fixes: [],
          evidence: [],
        },
      },
    ],
  }
  expect(problemSections(result).map((s) => s.id)).toEqual(['dismissed'])
  expect(problemCount(result)).toBe(0)
})

test('conflicts between the same mods with one outcome share a row', () => {
  const conflict = (target: string, packIds: string[]) => ({
    kind: 'edit',
    target,
    packIds,
    names: packIds,
    keys: packIds,
    winnerId: '',
    winnerName: 'unclear',
    overridden: null,
    cosmetic: true,
    fixes: [],
    evidence: [],
  })
  const seasonal = (target: string) => conflict(target, ['SVE', 'WorldMapGF'])
  const rows = assetRows([
    seasonal('loosesprites/map'),
    seasonal('loosesprites/map_fall'),
    conflict('maps/forest', ['SVE', 'Toothless']),
  ])
  expect(rows.length).toBe(2)
  const [first] = rows
  expect(first?.kind === 'asset' && first.siblings?.map((s) => s.target)).toEqual([
    'loosesprites/map_fall',
  ])
})

test('dismissed notes sharing one requirement token get distinct row keys', () => {
  const missing = (dependentId: string) => ({
    dependentId,
    dependentName: dependentId,
    id: 'outside:Immersive Farm 2 Remastered',
    minimumVersion: '',
    installedVersion: '',
    reason: 'absent',
    listed: true,
    note: '',
    external: true,
    optional: false,
    where: null,
  })
  const result: Result = {
    ...emptyResult(),
    dismissed: ['a', 'b', 'c'].map((d) => ({
      token: 'listed\toutside:immersive',
      missing: missing(d),
    })),
  }
  const dismissed = problemSections(result).find((s) => s.id === 'dismissed')
  const keys = rowKeys(dismissed?.rows ?? [])
  expect(keys).toHaveLength(3)
  expect(new Set(keys).size).toBe(3)
})
