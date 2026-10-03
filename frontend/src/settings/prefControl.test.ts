import { expect, test } from 'bun:test'
import { choiceStyle, prefControl } from './prefControl.ts'
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

test('choiceStyle picks cards, a button strip or a list', () => {
  const o = (label: string, hint?: string) => (hint ? { label, hint } : { label })
  expect(choiceStyle([o('Dark'), o('Light'), o('System')])).toBe('segmented')
  expect(choiceStyle([o('Stay open', 'a'), o('Minimise', 'b')])).toBe('cards')
  expect(choiceStyle([o('a'), o('b'), o('c'), o('d'), o('e')])).toBe('select')
  expect(
    choiceStyle([o('A very long option label here'), o('Another very long option label')]),
  ).toBe('select')
})
