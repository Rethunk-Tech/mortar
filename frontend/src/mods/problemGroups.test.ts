import { expect, test } from 'bun:test'
import type { Result } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { problemCount } from './lookup.ts'
import { assetRows, problemSections } from './problemGroups.ts'

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
        uniqueId: 'a',
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
        uniqueId: 'u',
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
        uniqueId: 'Pack.Compat',
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
