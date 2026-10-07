import { expect, test } from 'bun:test'
import { chosenSection, defaultSection } from './problemSection.ts'

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

test('a remembered section wins while it exists', () => {
  expect(chosenSection(tabs, 'cleanup')).toBe('cleanup')
  expect(chosenSection(tabs, 'gone')).toBe('missing')
})
