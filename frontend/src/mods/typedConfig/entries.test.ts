import { expect, test } from 'bun:test'
import { filterFile, humanizeKey, isModified, parseNumber, wantsSlider } from './entries.ts'
import type { ConfigEntry, ConfigFile } from './types.ts'

const entry = (over: Partial<ConfigEntry>): ConfigEntry => ({
  key: 'k',
  type: 'int',
  default: 1,
  value: 1,
  description: '',
  ...over,
})

test('an entry is modified only when it differs from its default, lists by content', () => {
  expect(isModified(entry({}))).toBe(false)
  expect(isModified(entry({ value: 2 }))).toBe(true)
  expect(isModified(entry({ type: 'list', default: ['a'], value: ['a'] }))).toBe(false)
  expect(isModified(entry({ type: 'list', default: ['a'], value: ['a', 'b'] }))).toBe(true)
})

test('small ranges get a slider and numbers clamp into range', () => {
  expect(wantsSlider(entry({ min: 0, max: 10 }))).toBe(true)
  expect(wantsSlider(entry({ min: 0, max: 5000 }))).toBe(false)
  expect(wantsSlider(entry({ min: 0 }))).toBe(false)
  expect(parseNumber('99', entry({ min: 0, max: 10 }))).toBe(10)
  expect(parseNumber('abc', entry({}))).toBeNull()
})

test('search matches keys and descriptions and a section name keeps its entries', () => {
  const file: ConfigFile = {
    name: 'a.cfg',
    label: 'a',
    format: 'bepinex',
    sections: [
      {
        name: 'General',
        entries: [entry({ key: 'Speed' }), entry({ key: 'Gravity', description: 'fall speed' })],
      },
      { name: 'Audio', entries: [entry({ key: 'Volume' })] },
    ],
  }
  expect(filterFile(file, 'speed').sections[0]?.entries).toHaveLength(2)
  expect(filterFile(file, 'audio').sections[0]?.entries).toHaveLength(1)
  expect(filterFile(file, 'zzz').sections).toHaveLength(0)
})

test('a key is split into words at camelCase and number boundaries', () => {
  expect(humanizeKey('SpawnFreqCoal0To2')).toBe('Spawn Freq Coal 0 To 2')
  expect(humanizeKey('RemoveTreeHidingBackyardMouseStatue')).toBe(
    'Remove Tree Hiding Backyard Mouse Statue',
  )
  expect(humanizeKey('enableHTTPServer')).toBe('enable HTTP Server')
  expect(humanizeKey('max_item-count.x')).toBe('max item count x')
  expect(humanizeKey('Speed')).toBe('Speed')
  expect(humanizeKey('')).toBe('')
})
