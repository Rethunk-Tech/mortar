import { expect, test } from 'bun:test'
import type { UpdatesResult } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { updateCount, updatesForReview } from '../lookup.ts'
import { withheldUpdates } from './withheld.ts'

const result = {
  updates: [{ key: 'k', id: 'A', name: 'A', installed: '1', version: '2', nexusId: 7, url: '' }],
} as unknown as UpdatesResult
const hidden = { 7: { details: { page: { status: 'published', available: false } } } }

test('an update on a hidden Nexus page is withheld, and the chip counts what the dialog lists', () => {
  expect(updatesForReview(result, undefined, hidden)).toHaveLength(0)
  expect(withheldUpdates(result, undefined, hidden).map((u) => u.name)).toEqual(['A'])
  expect(updateCount(result, undefined, hidden)).toBe(
    updatesForReview(result, undefined, hidden).length,
  )
  expect(withheldUpdates(result, undefined, {})).toHaveLength(0)
})
