import { expect, test } from 'bun:test'
import { prefControl } from './prefControl.ts'
import { prefMatches, sectionVisible } from './prefFilter.ts'

test('prefControl maps registry types to fields', () => {
  expect(prefControl('bool')).toBe('switch')
  expect(prefControl('enum')).toBe('select')
  expect(prefControl('int')).toBe('number')
  expect(prefControl('string')).toBe('text')
})

test('prefMatches filters by label or description', () => {
  expect(prefMatches('', 'Backup before Play', 'When Mortar zips Saves')).toBe(true)
  expect(prefMatches('backup', 'Backup before Play', 'When Mortar zips Saves')).toBe(true)
  expect(prefMatches('zips', 'Backup before Play', 'When Mortar zips Saves')).toBe(true)
  expect(prefMatches('console', 'Backup before Play', 'When Mortar zips Saves')).toBe(false)
})

test('sectionVisible hides a section with no matching rows', () => {
  const rows = [
    { label: 'Backup before Play', description: 'When Mortar zips Saves' },
    { label: 'Dates', description: 'How timestamps are shown' },
  ]
  expect(sectionVisible('timestamp', rows)).toBe(true)
  expect(sectionVisible('nxm', rows)).toBe(false)
})
