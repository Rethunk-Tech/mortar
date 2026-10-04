import { expect, test } from 'bun:test'
import type { Entry } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { customCategoryById } from './group.ts'
import { categoryNames, hasAllTags, matchesQuery } from './modSearch.ts'

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
