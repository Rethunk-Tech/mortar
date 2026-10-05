import { expect, test } from 'bun:test'
import { filterFile, isModified, parseNumber, wantsSlider } from './entries.ts'
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
