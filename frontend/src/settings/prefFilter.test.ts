import { expect, test } from 'bun:test'
import { sectionVisible } from './prefFilter.ts'

test('a settings section hides when search matches none of its rows', () => {
  const rows = [
    { label: 'Keep in tray', description: 'Close to the tray' },
    { label: 'Language', description: '' },
  ]
  expect(sectionVisible('', rows)).toBe(true)
  expect(sectionVisible('tray', rows)).toBe(true)
  expect(sectionVisible('firewall', rows)).toBe(false)
})
