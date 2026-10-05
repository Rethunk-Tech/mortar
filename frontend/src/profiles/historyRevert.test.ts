import { expect, mock, test } from 'bun:test'
import { useToasts } from '../toasts/store.ts'

// Macros are compiled only in the app build; here a message is its English text.
mock.module('@lingui/core/macro', () => ({
  msg: (parts: TemplateStringsArray, ...values: unknown[]) => String.raw({ raw: parts }, ...values),
}))
mock.module('../toasts/report.ts', () => ({ errorMessage: (e: Error) => e.message }))
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts', () => ({
  History: async () => [],
  Revert: async () => {
    throw new Error('revert: missing from the store: nexus-1-2, nexus-3-4')
  },
  Snapshot: async () => {
    throw new Error('the record is gone')
  },
}))
const { pushMissingToast, revertHistoryEvent } = await import('./historyRevert.ts')

test('a revert reports missing mods to its caller and toasts nothing itself', async () => {
  const before = useToasts.getState().toasts.length
  const out = await revertHistoryEvent('stardew', 'p1', 'ev1', [])
  expect(useToasts.getState().toasts.length).toBe(before)
  expect(out.missingUnread).toBe(true)
  expect(out.missingNames).toEqual([])
})

test('an unread record never names store keys or claims they cannot be downloaded', async () => {
  const out = await revertHistoryEvent('stardew', 'p1', 'ev1', [])
  pushMissingToast(out)
  const toast = useToasts.getState().toasts.at(-1)
  expect(JSON.stringify(toast)).not.toContain('nexus-1-2')
  expect(toast?.body).toBeUndefined()
})
