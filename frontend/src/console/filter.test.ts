import { expect, test } from 'bun:test'
import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import {
  countByLevel,
  DEFAULT_FILTERS,
  firstError,
  formatAll,
  isFiltered,
  modsOf,
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

test('Trace and Debug are hidden by default and counts cover every line', () => {
  expect(visible(log, DEFAULT_FILTERS)).toHaveLength(5)
  const counts = countByLevel(log)
  expect(counts.get(Level.Trace)).toBe(1)
  expect(counts.get(Level.Debug)).toBe(0)
  expect(counts.get(Level.Error)).toBe(2)
})

test('search matches message or mod, ignoring case', () => {
  expect(visible(log, { ...DEFAULT_FILTERS, search: ' spacecore ' })).toHaveLength(1)
  expect(visible(log, { ...DEFAULT_FILTERS, search: 'COOKING' })).toHaveLength(3)
})

test('mod filter keeps only the picked mods', () => {
  const rows = visible(log, { ...DEFAULT_FILTERS, mods: ['SMAPI', 'Cooking'] })
  expect(rows.map((r) => r.message)).toEqual(['Loaded 42 mods', 'obsolete API', 'hello'])
  expect(modsOf(log)).toEqual(['Cooking', 'Love of Cooking', 'SMAPI'])
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
