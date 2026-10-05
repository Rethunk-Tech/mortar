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
  expect(mode?.type).toBe('string')
  expect(formatValue(['a'])).toBe('["a"]')
  expect(formatValue(false)).toBe('false')
})
