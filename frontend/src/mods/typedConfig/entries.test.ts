import { expect, test } from 'bun:test'
import {
  commaListKind,
  filterFile,
  formatLike,
  humanizeKey,
  isModified,
  listSeparator,
  looksLikeColour,
  parseNumber,
  suggestedOptions,
  wantsSlider,
} from './entries.ts'
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

test('a description that lists values suggests them; one without a list suggests none', () => {
  expect(
    suggestedOptions(
      'What language should it use? Some Languages may also need FontPatcher Languages Available: en, es, ru, kr, zh',
    ),
  ).toEqual(['en', 'es', 'ru', 'kr', 'zh'])
  expect(suggestedOptions('Options: Low/Medium/High.')).toEqual(['Low', 'Medium', 'High'])
  expect(suggestedOptions('The name shown above the player.')).toEqual([])
})

test('a picked colour is written in the form the mod wrote', () => {
  expect(formatLike('#00ff00', '#FF2222')).toBe('#00FF00')
  expect(formatLike('#00ff00', '00000040')).toBe('00FF0040')
  expect(formatLike('#00ff00', '#aa00eeff')).toBe('#00ff00ff')
})

test('a hex string is a colour under a colour key or with a hash', () => {
  const e = (key: string, value: string) => entry({ key, type: 'string', value, default: value })
  expect(looksLikeColour(e('Emote Name Color Rare', '#2828FF'))).toBe(true)
  expect(looksLikeColour(e('OverlayBackgroundColor', '00000040'))).toBe(true)
  expect(looksLikeColour(e('Seed', '123456'))).toBe(false)
})

test('a comma list is numbers, chips or neither, and keeps the separator it was written with', () => {
  const e = (value: string, over: Partial<ConfigEntry> = {}) =>
    entry({ key: 'List', type: 'string', value, default: value, ...over })
  expect(commaListKind(e('200, 250, 350'))).toBe('numbers')
  expect(commaListKind(e('0, 0.8'))).toBe('numbers')
  expect(commaListKind(e('750,600, 800, 1000'))).toBe('numbers')
  expect(commaListKind(e('', { default: '1,2' }))).toBe('numbers')
  expect(commaListKind(e('zombie, bat, 7'))).toBe('chips')
  expect(commaListKind(e('only one'))).toBeNull()
  expect(commaListKind(e('a, b', { description: 'Options: a, b' }))).toBeNull()
  expect(commaListKind(e(`${'x'.repeat(65)}, b`))).toBeNull()
  expect(commaListKind(e('LeftShift, RightTrigger', { key: 'MenuKey' }))).toBeNull()
  expect(commaListKind(e('#FF0000, #00FF00'))).toBe('chips')
  expect(listSeparator('a,b, c', '')).toBe(', ')
  expect(listSeparator('a,b', 'x, y')).toBe(',')
  expect(listSeparator('a', 'x, y')).toBe(', ')
  expect(listSeparator('', 'x,y')).toBe(',')
})

test('a colour or number list keeps its editor while its value is half typed', () => {
  expect(looksLikeColour(entry({ type: 'string', default: '#ff00aa', value: '#ff' }))).toBe(true)
  expect(commaListKind(entry({ type: 'string', default: '200, 250', value: '200, x' }))).toBe(
    'numbers',
  )
})
