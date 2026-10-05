import { beforeEach, expect, test } from 'bun:test'
import { useDetail } from '../mods/detail.ts'
import { useNav } from '../nav/store.ts'
import { findModInProfiles, openModInProfile } from './findMod.ts'
import { useProfiles } from './store.ts'
import { testProfile } from './testProfile.ts'

beforeEach(() => {
  useDetail.setState(useDetail.getInitialState(), true)
  useNav.setState(useNav.getInitialState(), true)
  useProfiles.setState(useProfiles.getInitialState(), true)
})

test('finds mods by name or UniqueID across profiles', () => {
  const profiles = [
    testProfile({
      id: 'aaaa',
      name: 'Farm',
      entries: [
        {
          key: 'k1',
          previousKey: '',
          source: { kind: 'nexus', name: 'CP' },
          mods: [
            {
              id: 'Pathoschild.ContentPatcher',
              version: '2.1',
              name: 'Content Patcher',
              author: '',
              folder: '.',
              needs: null,
            },
          ],
          disabled: ['Pathoschild.ContentPatcher'],
          added: '',
        },
      ],
    }),
    testProfile({
      id: 'bbbb',
      name: 'Co-op',
      entries: [
        {
          key: 'k2',
          previousKey: '',
          source: { kind: 'local', name: 'x.zip' },
          mods: [
            {
              id: 'SpaceChase0.SpaceCore',
              version: '1.0',
              name: 'SpaceCore',
              author: '',
              folder: '.',
              needs: null,
            },
          ],
          disabled: [],
          added: '',
        },
      ],
    }),
  ]
  expect(findModInProfiles(profiles, 'content').map((h) => h.profileId)).toEqual(['aaaa'])
  expect(findModInProfiles(profiles, 'spacecore').map((h) => h.id)).toEqual([
    'SpaceChase0.SpaceCore',
  ])
  expect(findModInProfiles(profiles, '  ')).toEqual([])
})

test('opening a hit keeps the UniqueID to select after the profile loads', () => {
  useNav.setState({ route: { name: 'profiles', game: 'stardew' } })
  openModInProfile({ profileId: 'bbbb', key: 'k2', id: 'SpaceChase0.SpaceCore' })
  expect(useProfiles.getState().openId).toBe('bbbb')
  expect(useNav.getState().route.name).toBe('game')
  expect(useDetail.getState().pendingId).toBe('k2/SpaceChase0.SpaceCore')
})
