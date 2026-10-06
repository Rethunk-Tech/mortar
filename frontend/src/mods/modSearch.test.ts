import { expect, test } from 'bun:test'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { customCategoryById } from './group.ts'
import { categoryNames, hasAllTags, matchesQuery, siteCategory } from './modSearch.ts'

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
