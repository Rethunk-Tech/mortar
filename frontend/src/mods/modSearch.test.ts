import { expect, test } from 'bun:test'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { customCategoryById } from './group.ts'
import {
  categoryNames,
  hasAllTags,
  matchesQuery,
  searchFields,
  siteCategories,
  siteCategory,
} from './modSearch.ts'

test('search matches category names case-insensitively', () => {
  const custom = customCategoryById([{ id: 'c1', name: 'Farm Life' }])
  const names = categoryNames({ categoryOverride: 'c1' } as Entry, 'Visuals', custom)
  expect(matchesQuery('farm', names)).toBe(true)
  expect(matchesQuery('visuals', categoryNames(undefined, 'Visuals', custom))).toBe(true)
  expect(matchesQuery('zzz', names)).toBe(false)
  expect(matchesQuery('', [])).toBe(true)
})

test('tag chips AND across the selection', () => {
  expect(hasAllTags(['a', 'b'], ['a', 'b'])).toBe(true)
  expect(hasAllTags(['a'], ['a', 'b'])).toBe(false)
  expect(hasAllTags(undefined, [])).toBe(true)
})

test('a Thunderstore package takes its recorded category, a Nexus mod its page category', () => {
  const pkg = { source: { kind: 'thunderstore', category: 'Tools' } } as Entry
  const nexus = { source: { kind: 'nexus', category: 'MAIN' } } as Entry
  expect(siteCategory(pkg, undefined)).toBe('Tools')
  expect(siteCategory(nexus, 'Visuals')).toBe('Visuals')
})

test('every site category of a package is searchable, not only the one picked', () => {
  const pkg = {
    source: { kind: 'thunderstore', category: 'Tools' },
  } as Entry
  const listed = siteCategories(pkg, ['Tools', 'Mods', 'Libraries'])
  expect(listed).toEqual(['Tools', 'Mods', 'Libraries'])
  expect(matchesQuery('librar', categoryNames(pkg, 'Tools', customCategoryById([]), listed))).toBe(
    true,
  )
  expect(
    matchesQuery(
      'librar',
      searchFields(
        { name: 'x', author: '', id: 'x' } as never,
        pkg,
        {},
        {
          customById: customCategoryById([]),
          siteCategories: listed,
        },
      ),
    ),
  ).toBe(true)
  expect(
    matchesQuery(
      'zzz',
      searchFields(
        { name: 'x', author: '', id: 'x' } as never,
        pkg,
        {},
        {
          customById: customCategoryById([]),
          siteCategories: listed,
        },
      ),
    ),
  ).toBe(false)
})

test('a package the cached listing lacks keeps its recorded category alone', () => {
  const pkg = { source: { kind: 'thunderstore', category: 'Tools' } } as Entry
  expect(siteCategories(pkg, undefined)).toEqual(['Tools'])
  expect(siteCategories(pkg, [])).toEqual(['Tools'])
  expect(siteCategories({ source: { kind: 'nexus' } } as Entry, undefined)).toBeUndefined()
})
