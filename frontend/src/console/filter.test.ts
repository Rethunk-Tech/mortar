import { expect, test } from 'bun:test'
import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import {
  countByLevel,
  DEFAULT_FILTERS,
  firstError,
  formatAll,
  incompatibleSMAPI,
  isFiltered,
  levelsFromFloor,
  modsOf,
  shownLog,
  visible,
} from './filter.ts'

const e = (level: Level, mod: string, message: string, cont = false): Entry => ({
  seq: 0,
  time: '19:43:50',
  level,
  mod,
  message,
  cont,
})

const log = [
  e(Level.Trace, 'SMAPI', 'loading Love of Cooking'),
  e(Level.Info, 'SMAPI', 'Loaded 42 mods'),
  e(Level.Warn, 'SMAPI', 'obsolete API'),
  e(Level.Error, 'Love of Cooking', 'Failed to load:'),
  e(Level.Error, 'Love of Cooking', '  needs SpaceCore', true),
  e(Level.Alert, 'Cooking', 'hello'),
]

// The searches and mod filters below run at the Info floor, so the Info lines in the log count.
const INFO = { ...DEFAULT_FILTERS, levels: levelsFromFloor('info') }

test('levelsFromFloor warn matches the default filter', () => {
  expect(levelsFromFloor('warn')).toEqual(DEFAULT_FILTERS.levels)
  expect(levelsFromFloor('error')).toEqual([Level.Error, Level.Alert])
})

test('Trace and Debug are hidden at the Info floor and counts cover every line', () => {
  expect(visible(log, INFO)).toHaveLength(5)
  const counts = countByLevel(log)
  expect(counts.get(Level.Trace)).toBe(1)
  expect(counts.get(Level.Debug)).toBe(0)
  expect(counts.get(Level.Error)).toBe(2)
})

test('search matches message or mod, ignoring case', () => {
  expect(visible(log, { ...INFO, search: ' spacecore ' })).toHaveLength(2)
  expect(visible(log, { ...INFO, search: 'COOKING' })).toHaveLength(3)
})

test('mod filter keeps only the picked mods', () => {
  const rows = visible(log, { ...INFO, mods: ['SMAPI', 'Cooking'] })
  expect(rows.map((r) => r.message)).toEqual(['Loaded 42 mods', 'obsolete API', 'hello'])
  expect(modsOf(log)).toEqual(['Cooking', 'Love of Cooking', 'SMAPI'])
})

test('filters complete stack-trace entries and can exclude a mod', () => {
  const rows = visible(log, { ...INFO, excludeMods: ['Love of Cooking'] })
  expect(rows.map((r) => r.message)).toEqual(['Loaded 42 mods', 'obsolete API', 'hello'])
  expect(visible(log, { ...INFO, search: 'spacecore' }).map((r) => r.message)).toEqual([
    'Failed to load:',
    '  needs SpaceCore',
  ])
})

test('isFiltered is false only for the defaults', () => {
  expect(isFiltered(DEFAULT_FILTERS)).toBe(false)
  expect(isFiltered({ ...DEFAULT_FILTERS, levels: [...DEFAULT_FILTERS.levels, Level.Trace] })).toBe(
    true,
  )
  expect(isFiltered({ ...DEFAULT_FILTERS, search: 'x' })).toBe(true)
})

test('firstError skips continuation lines and reports -1 when none', () => {
  expect(firstError(log)).toBe(3)
  expect(firstError(log.filter((l) => l.level === Level.Info))).toBe(-1)
})

test('formatAll writes SMAPI lines, continuations bare', () => {
  expect(formatAll(log.slice(2, 5))).toBe(
    '[19:43:50 WARN  SMAPI] obsolete API\n[19:43:50 ERROR Love of Cooking] Failed to load:\n  needs SpaceCore',
  )
})

test('formatAll writes BepInEx lines with level and source, headerless lines bare', () => {
  const rows = [
    { ...e(Level.Info, 'BepInEx', 'Loading [CullFactory 2.0.11]'), time: '' },
    { ...e(Level.Info, '', 'no header yet'), time: '' },
  ]
  expect(formatAll(rows)).toBe('[INFO  BepInEx] Loading [CullFactory 2.0.11]\nno header yet')
})

test('shownLog is empty when the open profile does not own the log', () => {
  expect(shownLog(log, true)).toBe(log)
  expect(shownLog(log, false)).toEqual([])
})

test('incompatible SMAPI is the max-version Oops line from SMAPI itself', () => {
  expect(
    incompatibleSMAPI(
      "Oops! You're running Stardew Valley 1.6.16, but this version of SMAPI is only compatible up to Stardew Valley 1.6.15. Please check for a newer version of SMAPI: https://smapi.io.",
    ),
  ).toBe(true)
  expect(incompatibleSMAPI('Failed to load a mod')).toBe(false)
})

test('an unknown or unloaded floor shows Warn and above, never Info', () => {
  expect(levelsFromFloor('')).toEqual(DEFAULT_FILTERS.levels)
  expect(levelsFromFloor('bogus')).toEqual(DEFAULT_FILTERS.levels)
})
