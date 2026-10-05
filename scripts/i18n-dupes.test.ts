import { expect, test } from 'bun:test'
import { duplicates } from './i18n-dupes.ts'

test('messages that differ only in case, punctuation or placeholder names collide', () => {
  expect(duplicates(['Restore {when}', 'Restore {label}', 'Disk space', 'disk space.'])).toEqual([
    ['Restore {when}', 'Restore {label}'],
    ['Disk space', 'disk space.'],
  ])
})

test('a control that opens a dialog and its title stay distinct', () => {
  expect(
    duplicates(['Save as template…', 'Save as template', 'Delete {0}?', 'Delete {name}']),
  ).toEqual([])
})
