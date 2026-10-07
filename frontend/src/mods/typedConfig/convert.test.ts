import { expect, test } from 'bun:test'
import { formatValue, toSections } from './convert.ts'

test('wire text becomes typed values, with the default standing in when none is recorded', () => {
  const [section] = toSections([
    {
      name: 'General',
      entries: [
        { key: 'On', type: 'bool', value: 'true', default: 'false', hasDefault: true },
        { key: 'Speed', type: 'float', value: '1.5', hasDefault: false, min: 0, max: 3 },
        { key: 'Names', type: 'list', value: '["a","b"]', hasDefault: false },
        {
          key: 'Mode',
          type: 'enum',
          value: 'A',
          hasDefault: false,
          values: ['A', 'B'],
          flags: true,
        },
      ],
    },
  ])
  const [on, speed, names, mode] = section?.entries ?? []
  expect([on?.value, on?.default]).toEqual([true, false])
  expect([speed?.value, speed?.default, speed?.max]).toEqual([1.5, 1.5, 3])
  expect(names?.value).toEqual(['a', 'b'])
  expect(mode?.type).toBe('multi')
  expect(formatValue(['a'])).toBe('["a"]')
  expect(formatValue(false)).toBe('false')
})

test('a value keeps its JSON type through a read and the text written back', () => {
  const [section] = toSections([
    {
      name: '',
      entries: [
        { key: 'On', type: 'bool', value: 'false', hasDefault: false },
        { key: 'Count', type: 'int', value: '7', hasDefault: false },
        { key: 'Ratio', type: 'float', value: '0.25', hasDefault: false },
        { key: 'Name', type: 'string', value: 'true', hasDefault: false },
      ],
    },
  ])
  const values = section?.entries.map((e) => e.value)
  expect(values).toEqual([false, 7, 0.25, 'true'])
  expect(values?.map(formatValue)).toEqual(['false', '7', '0.25', 'true'])
})

test('a Content Patcher setting picks its widget from the schema the backend typed', () => {
  const [section] = toSections([
    {
      name: '',
      entries: [
        { key: 'Mist', type: 'bool', value: 'true', default: 'false', hasDefault: true },
        {
          key: 'Season',
          type: 'enum',
          value: 'Fall',
          hasDefault: true,
          default: 'Spring',
          values: ['Spring', 'Fall'],
        },
        {
          key: 'Crops',
          type: 'enum',
          value: 'Corn, Kale',
          hasDefault: false,
          values: ['Corn', 'Kale'],
          flags: true,
        },
        { key: 'Nick', type: 'string', value: 'default', hasDefault: false },
      ],
    },
  ])
  const entries = section?.entries ?? []
  expect(entries.map((e) => e.type)).toEqual(['bool', 'enum', 'multi', 'string'])
  expect(entries.map((e) => formatValue(e.value))).toEqual([
    'true',
    'Fall',
    'Corn, Kale',
    'default',
  ])
  expect(formatValue(entries[0]?.default ?? '')).toBe('false')
})
