import { afterEach, beforeEach, expect, mock, test } from 'bun:test'
import { msg } from '@lingui/core/macro'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import * as profileService from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { testProfile } from '../profiles/testProfile.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { useToasts } from '../toasts/store.ts'
import { dropMods } from './storeEntries.ts'

const mod = { key: 'k1', id: 'SpaceCore', name: 'SpaceCore' } as Mod
const stale = {
  disabled: true,
  reason: i18n._(msg`This is no longer the latest change to that mod.`),
}

let profilesBefore: ReturnType<typeof useProfiles.getState>

beforeEach(() => {
  profilesBefore = useProfiles.getState()
  useToasts.setState(useToasts.getInitialState(), true)
})

afterEach(() => {
  mock.restore()
  useToasts.setState(useToasts.getInitialState(), true)
  useProfiles.setState(profilesBefore, true)
})

test('remove-undo is disabled once the entry is gone or superseded', async () => {
  const profile = testProfile({ id: 'p1', name: 'Farm', entries: [{ key: 'k1', mods: [mod] }] })
  useProfiles.setState({
    game: { id: 'stardew' } as never,
    openId: 'p1',
    profiles: [profile],
  })
  stubBinding(profileService, 'RemoveEntries', () => ({ ...profile, entries: [] }))
  await dropMods(() => ({ load: async () => undefined }), [mod])
  const live = useToasts.getState().toasts.at(-1)?.action?.live
  expect(live?.()).toEqual(stale)

  useProfiles.getState().replace({ ...profile, entries: [{ key: 'k2', mods: [mod] }] })
  expect(live?.()).toEqual(stale)

  useProfiles.getState().replace(profile)
  expect(live?.()).toEqual({ disabled: false })
})
