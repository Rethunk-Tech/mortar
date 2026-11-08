import { afterEach, expect, test } from 'bun:test'
import { cleanup, renderHook } from '@testing-library/react'
import type { UpdatesResult } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { useToasts } from '../../toasts/store.ts'
import { updateCount, updatesForReview } from '../lookup.ts'
import { useUpdates } from '../updates.ts'
import { useEmptyReviewNotice, withheldUpdates } from './withheld.ts'

afterEach(() => {
  cleanup()
  useUpdates.setState({ reviewing: false })
})

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

test('a review with nothing to show closes and says so', () => {
  useUpdates.setState({ reviewing: true })
  useToasts.setState({ toasts: [] })
  renderHook(() => useEmptyReviewNotice(true))
  expect(useUpdates.getState().reviewing).toBe(false)
  expect(useToasts.getState().toasts.map((toast) => toast.title)).toEqual(['No updates to review'])
})

test('a review with something to show stays open and says nothing', () => {
  useUpdates.setState({ reviewing: true })
  useToasts.setState({ toasts: [] })
  renderHook(() => useEmptyReviewNotice(false))
  expect(useUpdates.getState().reviewing).toBe(true)
  expect(useToasts.getState().toasts).toHaveLength(0)
})
