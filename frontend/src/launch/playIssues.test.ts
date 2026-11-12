import { expect, test } from 'bun:test'
import type { Group } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/playcheck/models.ts'
import { overflowIssueCount, worded } from './playIssues.ts'

const group = (over: Partial<Group>): Group => ({
  kind: 'missing',
  count: 1,
  names: ['A'],
  ...over,
})

test('a conflict lists its mods as a sentence in the language, and a bare target as it is', () => {
  const g = worded(
    group({
      kind: 'conflicts',
      count: 2,
      names: ['SVE, Other', 'maps/town'],
      nameParts: [['SVE', 'Other'], null],
    }),
  )
  expect(g.names).toEqual(['SVE and Other', 'maps/town'])
})

test('changes are worded from their kind and values', () => {
  const g = worded(
    group({
      kind: 'changes',
      count: 1,
      names: ['added Alpha'],
      changes: [{ kind: 'added', mod: 'smapi:A', name: 'Alpha', key: 'a', detail: 'added Alpha' }],
    }),
  )
  expect(g.names).toEqual(['Added Alpha'])
})

test('the last-profile group names the save by its farm, else its name', () => {
  expect(
    worded(group({ kind: 'lastProfile', names: [], saveFolder: 'Farm_1', saveFarm: 'Sunny' })).save,
  ).toBe('Sunny')
  expect(
    worded(group({ kind: 'lastProfile', names: [], saveFolder: 'Farm_1', saveName: 'World' })).save,
  ).toBe('World')
  expect(worded(group({ kind: 'missing' })).save).toBe('')
})

test('overflowIssueCount is the names past the five shown', () => {
  expect(overflowIssueCount({ count: 6, names: ['A', 'B', 'C', 'D', 'E'] })).toBe(1)
  expect(overflowIssueCount({ count: 2, names: ['A', 'B'] })).toBe(0)
})
