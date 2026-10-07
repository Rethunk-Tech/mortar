import { expect, test } from 'bun:test'
import { effectiveSort, MERGED_SORTS, sortsFor } from './browseSort.ts'

const sources = [
  { id: 'nexus', name: 'Nexus', sorts: ['endorsements', 'downloads'] },
  { id: 'github', name: 'GitHub', sorts: ['stars', 'forks', 'updated'] },
  { id: 'itch', name: 'itch.io', sorts: null },
]

test('sortsFor offers a source its own sorts and All sources only the comparable ones', () => {
  expect(sortsFor('github', sources)).toEqual(['stars', 'forks', 'updated'])
  expect(sortsFor('itch', sources)).toEqual([])
  expect(sortsFor('all', sources)).toEqual(MERGED_SORTS)
  expect(MERGED_SORTS).not.toContain('stars')
})

test('effectiveSort drops a sort the selected source does not honour', () => {
  expect(effectiveSort('stars', sortsFor('github', sources))).toBe('stars')
  expect(effectiveSort('stars', sortsFor('nexus', sources))).toBe('')
  expect(effectiveSort('downloads', sortsFor('github', sources))).toBe('')
})
