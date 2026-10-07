import { expect, test } from 'bun:test'
import {
  gridColumns,
  includedKeys,
  lastUsedDestination,
  leftOutCounts,
  shareDestinations,
} from './methods.ts'

const on = { disabledMods: false, fomodChoices: true, notes: true, configFiles: true }

test('thunderstore is listed only for games that have it', () => {
  expect(shareDestinations({ thunderstore: false, count: 3 }).map((d) => d.id)).toEqual([
    'mortar',
    'nexus',
    'nearby',
    'list',
  ])
  expect(shareDestinations({ thunderstore: true, count: 3 }).map((d) => d.id)).toContain(
    'thunderstore',
  )
})

test('an empty share disables every destination', () => {
  expect(shareDestinations({ thunderstore: true, count: 0 }).every((d) => d.disabled)).toBe(true)
})

test('lastUsedDestination needs a usable tile', () => {
  const ds = shareDestinations({ thunderstore: false, count: 5 })
  expect(lastUsedDestination('list', ds)).toBe('list')
  expect(lastUsedDestination(null, ds)).toBeNull()
  expect(lastUsedDestination('thunderstore', ds)).toBeNull()
  expect(
    lastUsedDestination('list', shareDestinations({ thunderstore: false, count: 0 })),
  ).toBeNull()
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

test('gridColumns never leaves an orphan tile', () => {
  expect([1, 2, 3, 4, 5].map(gridColumns)).toEqual([1, 2, 3, 2, 3])
})
