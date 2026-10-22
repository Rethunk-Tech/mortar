import { expect, test } from 'bun:test'
import { contractions, duplicates, positionals, products, variants } from './i18n-dupes.ts'

test('messages that differ only in case, punctuation or placeholder names collide', () => {
  expect(duplicates(['Restore {when}', 'Restore {label}', 'Disk space', 'disk space.'])).toEqual([
    ['Restore {when}', 'Restore {label}'],
    ['Disk space', 'disk space.'],
  ])
})

test('dashes and parentheses are punctuation too', () => {
  expect(
    duplicates([
      'Update ready — applies when you close Mortar.',
      'Update ready: applies when you close Mortar.',
      'Sign in (Nexus Mods)',
      'Sign in: Nexus Mods',
    ]),
  ).toEqual([
    [
      'Update ready — applies when you close Mortar.',
      'Update ready: applies when you close Mortar.',
    ],
    ['Sign in (Nexus Mods)', 'Sign in: Nexus Mods'],
  ])
})

test('a control that opens a dialog and its title stay distinct', () => {
  expect(
    duplicates(['Save as template…', 'Save as template', 'Delete {0}?', 'Delete {name}']),
  ).toEqual([])
})

test('contractions are caught, possessives are not', () => {
  expect(
    contractions(["Mortar can't read this", 'What’s new', "The game's Mods folder", 'Cannot open']),
  ).toEqual(["Mortar can't read this", 'What’s new'])
})

test('curly quotes, dashes, arrows and middle dots are variants; the ellipsis and menu separator are not', () => {
  expect(
    variants(['the search is “x”', 'Ready — go', '1 → 2', 'Open…', 'A · B', 'Settings › General']),
  ).toEqual(['the search is “x”', 'Ready — go', '1 → 2', 'A · B'])
})

test('an article is not a different message', () => {
  expect(
    duplicates(['Add the missing mods', 'Add missing mods', 'Stop a game', 'Stop game?']),
  ).toEqual([['Add the missing mods', 'Add missing mods']])
})

test('a positional placeholder is caught, a named one is not', () => {
  expect(
    positionals([
      'Delete {0}?',
      '{0, plural, one {# mod} other {# mods}}',
      'Delete {name}?',
      '{count, plural, one {# a} other {# b}}',
    ]),
  ).toEqual(['Delete {0}?', '{0, plural, one {# mod} other {# mods}}'])
})

test('a product name is caught by word, Mortar and placeholder names are not', () => {
  expect(
    products([
      'Sign in to Nexus Mods',
      'Hidden on {source}',
      'Mortar can start {game}',
      'Windows-format symbols',
      'Version {version}',
      '{launcher} was not found',
      'Open {steam}',
    ]),
  ).toEqual(['Sign in to Nexus Mods', 'Windows-format symbols'])
})
