import { expect, test } from 'bun:test'
import { applyCPSchema, fieldLabel, parseCPSchema, setByPath } from './configFields.ts'
import { isKeybind, parseConfig, stringifyConfig } from './configForm.ts'

test('labels split camel, pascal, snake and keep acronyms', () => {
  expect(fieldLabel('enableDebug')).toBe('Enable Debug')
  expect(fieldLabel('HTMLParser')).toBe('HTML Parser')
  expect(fieldLabel('use_smapi')).toBe('Use Smapi')
  expect(fieldLabel('SMAPI')).toBe('SMAPI')
})

test('detects SMAPI keybinds and rejects ordinary strings', () => {
  expect(isKeybind('F5')).toBe(true)
  expect(isKeybind('LeftShift + Q')).toBe(true)
  expect(isKeybind('F5, LeftShift + Q')).toBe(true)
  expect(isKeybind('Spring')).toBe(false)
  expect(isKeybind('')).toBe(false)
})

test('maps Content Patcher ConfigSchema to selects', () => {
  const schema = parseCPSchema({
    ConfigSchema: {
      Season: {
        AllowValues: 'Spring, Summer, Fall, Winter',
        AllowMultiple: true,
        AllowBlank: true,
        Default: 'Spring',
        Description: 'Which seasons',
        Section: 'World',
      },
    },
  })
  const season = schema.Season
  if (!season) {
    throw new Error('Season')
  }
  expect(season.allowValues).toEqual(['Spring', 'Summer', 'Fall', 'Winter'])
  const tree = applyCPSchema(parseConfig('{"Season":"Spring, Summer"}'), schema)
  expect(tree).toEqual({
    kind: 'object',
    entries: [
      {
        key: 'Season',
        node: {
          kind: 'choice',
          value: 'Spring, Summer',
          options: ['Spring', 'Summer', 'Fall', 'Winter'],
          multiple: true,
          allowBlank: true,
          defaultValue: 'Spring',
          description: 'Which seasons',
          section: 'World',
        },
      },
    ],
  })
})

test('parse keeps key order and number text; set leaves untouched keys', () => {
  const raw = '{"z":1,"n":1.50,"s":"keep"}'
  const tree = parseConfig(raw)
  expect(stringifyConfig(tree)).toBe('{"z":1,"n":1.50,"s":"keep"}\n')
  expect(stringifyConfig(setByPath(tree, 's', 'changed'))).toBe('{"z":1,"n":1.50,"s":"changed"}\n')
})

test('F5 becomes a keybind and scalar arrays become lists', () => {
  expect(parseConfig('"F5"')).toEqual({ kind: 'keybind', value: 'F5' })
  expect(parseConfig('[1,2]')).toEqual({
    kind: 'list',
    items: [
      { kind: 'int', value: '1' },
      { kind: 'int', value: '2' },
    ],
  })
  expect(parseConfig('[{"a":1}]').kind).toBe('readonly')
})
