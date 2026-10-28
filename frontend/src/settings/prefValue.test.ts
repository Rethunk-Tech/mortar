import { expect, test } from 'bun:test'
import type {
  PrefSpec,
  Settings,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { prefAsBool, prefAsNumber, prefAsString, prefRaw } from './prefValue.ts'

const spec = (over: Partial<PrefSpec>): PrefSpec => ({
  key: 'k',
  scope: 'game',
  type: 'int',
  default: '5',
  ...over,
})

test('prefAsString falls back to the default for missing and empty values', () => {
  expect(prefAsString(undefined, spec({}), 'g')).toBe('5')
  expect(prefAsString(null, spec({}), 'g')).toBe('5')
  expect(prefAsString('', spec({}), 'g')).toBe('5')
  expect(prefAsString('', spec({ type: 'string' }), 'g')).toBe('')
  expect(prefAsString(7, spec({}), 'g')).toBe('7')
})

test('a per-game default wins over the spec default', () => {
  expect(prefAsString(undefined, spec({ gameDefaults: { g: '9' } }), 'g')).toBe('9')
  expect(prefAsString(undefined, spec({ gameDefaults: { g: '9' } }), 'h')).toBe('5')
})

test('prefAsBool reads booleans and their strings', () => {
  const b = spec({ type: 'bool', default: 'false' })
  expect(prefAsBool(true, b, 'g')).toBe(true)
  expect(prefAsBool(false, b, 'g')).toBe(false)
  expect(prefAsBool('true', b, 'g')).toBe(true)
  expect(prefAsBool(undefined, spec({ type: 'bool', default: 'true' }), 'g')).toBe(true)
})

test('prefAsNumber returns the default when the value is not finite', () => {
  expect(prefAsNumber('12', spec({}), 'g')).toBe(12)
  expect(prefAsNumber('abc', spec({}), 'g')).toBe(5)
  expect(prefAsNumber(Number.POSITIVE_INFINITY, spec({}), 'g')).toBe(5)
  expect(prefAsNumber(Number.NaN, spec({}), 'g')).toBe(5)
})

test('prefRaw reads the game block for game scope and the top level otherwise', () => {
  const s = { k: 'top', games: { g: { k: 'game' } } } as unknown as Settings
  expect(prefRaw(s, spec({}), 'g')).toBe('game')
  expect(prefRaw(s, spec({}), 'h')).toBeUndefined()
  expect(prefRaw(s, spec({ scope: 'app' }), 'g')).toBe('top')
})
