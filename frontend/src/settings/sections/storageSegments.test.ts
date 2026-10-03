import { expect, test } from 'bun:test'
import { storageSegments } from './storageSegments.ts'

test('storage segments are disjoint and add up to the total', () => {
  const usage = {
    profiles: [{ size: 100 }, { size: 50 }],
    store: 400,
    cache: 30,
    backups: 20,
    trash: 0,
    total: 620,
  }
  const segs = storageSegments(usage)
  expect(segs.map((s) => [s.id, s.size])).toEqual([
    ['profiles', 150],
    ['store', 400],
    ['cache', 30],
    ['backups', 20],
    ['trash', 0],
    ['other', 20],
  ])
  expect(segs.reduce((n, s) => n + s.size, 0)).toBe(usage.total)
})
