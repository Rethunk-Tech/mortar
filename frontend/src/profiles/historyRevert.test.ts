import { afterEach, beforeEach, expect, mock, test } from 'bun:test'
import { msg } from '@lingui/core/macro'
import * as profileService from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { useToasts } from '../toasts/store.ts'
import { pushMissingToast, revertHistoryEvent } from './historyRevert.ts'
import { useProfiles } from './store.ts'
import { testProfile } from './testProfile.ts'

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

test('revert undo goes stale when a later version keeps the same entry keys', async () => {
  const restored = testProfile({
    id: 'p1',
    name: 'Farm',
    entries: [{ key: 'k1', mods: [{ id: 'SpaceCore' }] } as never],
  })
  stubBinding(profileService, 'Revert', () => restored)
  const before = useProfiles.getState()
  try {
    useProfiles.setState({
      game: { id: 'stardew' } as never,
      openId: 'p1',
      profiles: [restored],
    })
    await revertHistoryEvent('stardew', 'p1', 'ev1', [{ id: 'before' } as never])
    useProfiles.getState().replace({
      ...restored,
      entries: [{ key: 'k1', mods: [{ id: 'SpaceCore.2' }] } as never],
    })
    expect(useToasts.getState().toasts.at(-1)?.action?.live?.()).toEqual({
      disabled: true,
      reason: i18n._(msg`This is no longer the latest change to that mod.`),
    })
  } finally {
    useProfiles.setState(before)
  }
})
