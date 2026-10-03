import { expect, test } from 'bun:test'
import {
  authorFieldIncludes,
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
