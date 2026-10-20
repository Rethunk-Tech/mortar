import { expect, test } from 'bun:test'
import { groupConflicts } from './fileConflicts.ts'

test('files won from the same mods read as one row, the biggest first', () => {
  const rows = groupConflicts([
    { path: 'a.cfg', winner: 'c', beaten: ['a', 'b'] },
    { path: 'b.cfg', winner: 'b', beaten: ['a'] },
    { path: 'c.cfg', winner: 'c', beaten: ['a', 'b'] },
  ])
  expect(rows.map((r) => [r.winner, r.beaten.join(), r.paths.length])).toEqual([
    ['c', 'a,b', 2],
    ['b', 'a', 1],
  ])
})
