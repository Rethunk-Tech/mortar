import { expect, test } from 'bun:test'
import { chosenSection, defaultSection, stepSection } from './problemSection.ts'

const tabs = [
  { id: 'cosmetic', label: 'Cosmetic', count: 3, errors: false },
  { id: 'missing', label: 'Missing', count: 1, errors: true },
  { id: 'cleanup', label: 'Cleanup', count: 2, errors: false },
]

test('the first section with errors leads, else the first with anything', () => {
  expect(defaultSection(tabs)).toBe('missing')
  expect(defaultSection([tabs[0], tabs[2]].flatMap((t) => (t ? [t] : [])))).toBe('cosmetic')
  expect(defaultSection([])).toBe('')
})

test('a remembered section wins while it exists, and arrows stop at the ends', () => {
  expect(chosenSection(tabs, 'cleanup')).toBe('cleanup')
  expect(chosenSection(tabs, 'gone')).toBe('missing')
  expect(stepSection(tabs, 'missing', 1)).toBe('cleanup')
  expect(stepSection(tabs, 'cleanup', 1)).toBe('cleanup')
  expect(stepSection(tabs, 'cosmetic', -1)).toBe('cosmetic')
})
