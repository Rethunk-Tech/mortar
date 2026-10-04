import { beforeEach, expect, mock, test } from 'bun:test'
import { Hint } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { State as LaunchState } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useLaunch } from '../launch/store.ts'
import { useMods } from '../mods/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { testProfile } from '../profiles/testProfile.ts'
import { useToasts } from '../toasts/store.ts'
import {
  dropInstallGate,
  entryForNames,
  shouldConsiderMissing,
  undoArchiveInstall,
  useInstall,
} from './store.ts'

const calls = {
  remove: [] as string[],
  roll: [] as string[],
}

mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts', () => ({
  InstallArchive: async () => ({
    profile: baseProfile(),
    added: ['SpaceCore'],
    updated: false,
    versionChanged: false,
  }),
  InstallRemap: async () => ({
    profile: baseProfile(),
    added: ['SpaceCore'],
    updated: false,
    versionChanged: false,
  }),
  RemoveEntry: async (_game: string, _id: string, key: string) => {
    calls.remove.push(key)
    return baseProfile()
  },
  RollBack: async (_game: string, _id: string, key: string) => {
    calls.roll.push(key)
    return baseProfile()
  },
}))

function baseProfile(): Profile {
  return testProfile({
    id: 'p1',
    name: 'Main',
    entries: [
      {
        key: 'k1',
        previousKey: '',
        source: { kind: 'local', name: 'a.zip', picture: '' },
        mods: [{ uniqueId: 'SpaceCore', version: '1', name: 'SpaceCore', author: '', folder: '.' }],
        disabled: null,
      },
    ],
  })
}

beforeEach(() => {
  calls.remove = []
  calls.roll = []
  useToasts.setState(useToasts.getInitialState(), true)
  useInstall.setState(useInstall.getInitialState(), true)
  useLaunch.setState(useLaunch.getInitialState(), true)
  useMods.setState({ ...useMods.getInitialState(), load: async () => undefined }, true)
  useProfiles.setState(
    { ...useProfiles.getInitialState(), profiles: [baseProfile()], openId: 'p1' },
    true,
  )
})

test('undo of a new archive install removes the entry', async () => {
  expect(entryForNames(baseProfile(), ['SpaceCore'])?.key).toBe('k1')
  await undoArchiveInstall('stardew', 'p1', 'k1', false)
  expect(calls.remove).toEqual(['k1'])
  expect(calls.roll).toEqual([])
})

test('undo of an archive update rolls back', async () => {
  await undoArchiveInstall('stardew', 'p1', 'k1', true)
  expect(calls.roll).toEqual(['k1'])
  expect(calls.remove).toEqual([])
})

test('undo of an archive install does nothing while the profile is locked', async () => {
  useLaunch.setState({
    status: {
      game: 'stardew',
      state: LaunchState.Running,
      profile: 'p1',
      since: 0,
      hint: Hint.$zero,
      error: '',
    },
  })
  await undoArchiveInstall('stardew', 'p1', 'k1', false)
  expect(calls.remove).toEqual([])
})

test('dropInstallGate toasts locked as locked and other missing context as skip', () => {
  expect(dropInstallGate(true, true, true)).toBe('locked')
  expect(dropInstallGate(true, true, false)).toBe('ok')
  expect(dropInstallGate(false, true, true)).toBe('skip')
  expect(dropInstallGate(true, false, true)).toBe('skip')
})

test('missing-deps offers skip when the open profile is no longer the install target', () => {
  expect(shouldConsiderMissing('p2', 'p1')).toBe(false)
  expect(shouldConsiderMissing('p1', 'p1')).toBe(true)
})
