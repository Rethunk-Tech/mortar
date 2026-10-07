import { expect, test } from 'bun:test'
import {
  gridColumns,
  includeAvailability,
  includedKeys,
  lastUsedDestination,
  leftOutCounts,
  shareDestinations,
} from './methods.ts'

const on = {
  disabledMods: false,
  fomodChoices: true,
  notes: true,
  configFiles: true,
  problemChoices: true,
}

test('thunderstore is listed only for games that have it', () => {
  expect(shareDestinations({ thunderstore: false, count: 3, leftOut: 0 }).map((d) => d.id)).toEqual(
    ['mortar', 'nexus', 'nearby', 'list'],
  )
  expect(
    shareDestinations({ thunderstore: true, count: 3, leftOut: 0 }).map((d) => d.id),
  ).toContain('thunderstore')
})

test('an empty share disables every destination', () => {
  expect(
    shareDestinations({ thunderstore: true, count: 0, leftOut: 0 }).every((d) => d.disabled),
  ).toBe(true)
})

const disabledIds = (count: number, leftOut: number) =>
  shareDestinations({ thunderstore: true, count, leftOut })
    .filter((d) => d.disabled)
    .map((d) => d.id)

test('a profile of local archives keeps every destination that can carry them', () => {
  expect(disabledIds(0, 2)).toEqual(['nexus', 'thunderstore'])
  expect(shareDestinations({ thunderstore: false, count: 0, leftOut: 2 })[1]?.reason).toBe(
    'local-only',
  )
})

test('a profile of site mods and a mixed one disable nothing', () => {
  expect(disabledIds(3, 0)).toEqual([])
  expect(disabledIds(1, 2)).toEqual([])
})

test('lastUsedDestination needs a usable tile', () => {
  const ds = shareDestinations({ thunderstore: false, count: 5, leftOut: 0 })
  expect(lastUsedDestination('list', ds)).toBe('list')
  expect(lastUsedDestination(null, ds)).toBeNull()
  expect(lastUsedDestination('thunderstore', ds)).toBeNull()
  expect(
    lastUsedDestination('list', shareDestinations({ thunderstore: false, count: 0, leftOut: 0 })),
  ).toBeNull()
})

test('includedKeys follows the target and the game', () => {
  expect(includedKeys(on, { target: 'link', fomod: false })).toEqual(['notes'])
  expect(includedKeys(on, { target: 'file', fomod: true })).toEqual([
    'notes',
    'fomodChoices',
    'configFiles',
    'problemChoices',
  ])
  expect(includedKeys({ ...on, notes: false }, { target: 'link', fomod: false })).toEqual([])
})

test('a paired computer gets every setting, an unpaired one only what is safe to hand over', () => {
  expect(Object.values(includeAvailability('paired')).every((v) => v === null)).toBe(true)
  expect(includeAvailability('unpaired')).toEqual({
    disabledMods: null,
    fomodChoices: null,
    notes: null,
    configFiles: 'paired-only',
    problemChoices: null,
  })
  expect(includedKeys(on, { target: 'unpaired', fomod: false })).toEqual([
    'notes',
    'problemChoices',
  ])
  expect(includeAvailability('link').problemChoices).toBe('file-only')
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
