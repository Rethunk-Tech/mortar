import { afterEach, beforeEach, expect, mock, test } from 'bun:test'
import * as profileService from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { useToasts } from '../toasts/store.ts'
import { pushMissingToast, revertHistoryEvent } from './historyRevert.ts'

beforeEach(() => {
  stubBinding(profileService, 'History', () => [])
  stubBinding(profileService, 'Revert', () => {
    throw new Error('revert: missing from the store: nexus-1-2, nexus-3-4')
  })
  stubBinding(profileService, 'Snapshot', () => {
    throw new Error('the record is gone')
  })
})
afterEach(() => {
  mock.restore()
  useToasts.setState(useToasts.getInitialState(), true)
})

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
