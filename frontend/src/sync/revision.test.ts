import { expect, test } from 'bun:test'
import { diffIsStale, revisionToAnswer } from './revision.ts'

test('an answer targets the revision shown, and a newer offer marks the diff stale', () => {
  const shown = { diff: { add: ['a'], remove: [] } as never, revision: 'r1' }
  expect(revisionToAnswer('r2', shown)).toBe('r1')
  expect(revisionToAnswer('r2', null)).toBe('r2')
  expect(diffIsStale('r2', shown)).toBe(true)
  expect(diffIsStale('r1', shown)).toBe(false)
  expect(diffIsStale('r1', null)).toBe(false)
})
