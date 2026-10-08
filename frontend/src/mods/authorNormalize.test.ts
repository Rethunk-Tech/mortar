import { expect, test } from 'bun:test'
import {
  authorFieldIncludes,
  formatAuthors,
  normalizeAuthorName,
  splitManifestAuthors,
} from './authorNormalize.ts'

test('normalises author case and whitespace', () => {
  expect(normalizeAuthorName('  Pathoschild  ')).toBe('pathoschild')
  expect(normalizeAuthorName('Space\tChase')).toBe('space chase')
})

test('splits manifest authors on commas and ampersands', () => {
  expect(splitManifestAuthors('A, B & C')).toEqual(['A', 'B', 'C'])
  expect(splitManifestAuthors('')).toEqual([])
})

test('matches any listed author in a manifest field', () => {
  expect(authorFieldIncludes('Pathoschild, SpaceChase0', 'spacechase0')).toBe(true)
  expect(authorFieldIncludes('Only One', 'Other')).toBe(false)
})

test('splits on commas, ampersands and the word and', () => {
  expect(splitManifestAuthors('A, B and C')).toEqual(['A', 'B', 'C'])
  expect(splitManifestAuthors('A, B, and C')).toEqual(['A', 'B', 'C'])
  expect(splitManifestAuthors('A & B & C')).toEqual(['A', 'B', 'C'])
  expect(splitManifestAuthors('A and B')).toEqual(['A', 'B'])
  expect(splitManifestAuthors('Sandy Anderson')).toEqual(['Sandy Anderson'])
})

test('formats each name once', () => {
  expect(formatAuthors('FlashShifter & Esca & and kittycatcasey')).toBe(
    'FlashShifter, Esca, and kittycatcasey',
  )
  expect(formatAuthors('A and B')).toBe('A and B')
  expect(formatAuthors('Solo')).toBe('Solo')
  expect(formatAuthors('')).toBe('')
  expect(formatAuthors('A, a & B')).toBe('A and B')
})
