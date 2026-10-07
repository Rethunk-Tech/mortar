import { expect, test } from 'bun:test'
import { includedKeys, leftOutCounts, pickMethod, shareMethods } from './methods.ts'

const on = { disabledMods: false, fomodChoices: true, notes: true, configFiles: true }

test('thunderstore is listed only for games that have it', () => {
  expect(shareMethods({ thunderstore: false, count: 3 }).map((m) => m.id)).toEqual([
    'link',
    'file',
    'list',
    'nexus',
    'nearby',
  ])
  expect(shareMethods({ thunderstore: true, count: 3 }).map((m) => m.id)).toContain('thunderstore')
})

test('an empty share disables every method', () => {
  expect(shareMethods({ thunderstore: true, count: 0 }).every((m) => m.disabled)).toBe(true)
})

test('pickMethod prefers the remembered method, then the suggestion, then the link', () => {
  const ms = shareMethods({ thunderstore: false, count: 5 })
  expect(pickMethod('list', 'file', ms)).toBe('list')
  expect(pickMethod(null, 'file', ms)).toBe('file')
  expect(pickMethod('thunderstore', 'link', ms)).toBe('link')
  expect(pickMethod('list', 'file', shareMethods({ thunderstore: false, count: 0 }))).toBe('link')
})

test('includedKeys follows the method and the game', () => {
  expect(includedKeys(on, { file: false, fomod: false })).toEqual(['notes'])
  expect(includedKeys(on, { file: true, fomod: true })).toEqual([
    'notes',
    'fomodChoices',
    'configFiles',
  ])
  expect(includedKeys({ ...on, notes: false }, { file: false, fomod: false })).toEqual([])
})

test('leftOutCounts separates archive mods', () => {
  expect(leftOutCounts([{ reason: 'local' }, { reason: 'off' }, { reason: 'local' }])).toEqual({
    local: 2,
    other: 1,
  })
})
