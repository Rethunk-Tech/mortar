import { beforeEach, expect, mock, test } from 'bun:test'
import { Hint } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import { State as LaunchState } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import type {
  Entry,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import { useLaunch } from '../launch/store.ts'
import { useMods } from '../mods/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { entryForItem, shouldRollBack, singleNexusFailure, undoInstall } from './store.ts'

const calls = { remove: [] as string[], roll: [] as string[] }

mock.module('../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts', () => ({
  RemoveEntry: async (_game: string, _id: string, key: string) => {
    calls.remove.push(key)
    return baseProfile()
  },
  RollBack: async (_game: string, _id: string, key: string) => {
    calls.roll.push(key)
    return baseProfile()
  },
}))

const entry = (): Entry => ({
  key: 'k1',
  previousKey: '',
  source: { kind: 'nexus', name: 'a.zip', modId: 1, picture: 'https://example.test/p.png' },
  mods: [{ uniqueId: 'SpaceCore', version: '1', name: 'SpaceCore', author: '', folder: '.' }],
  disabled: null,
})

function baseProfile(): Profile {
  return {
    id: 'p1',
    name: 'Main',
    notes: '',
    cover: '',
    order: 0,
    hidden: false,
    created: '',
    updated: '',
    entries: [entry()],
  }
}

const item = (over: Partial<Item> = {}): Item => ({
  id: 'q1',
  kind: 'install',
  game: 'stardew',
  profileId: 'p1',
  modId: 1,
  fileId: 10,
  currentFileId: 0,
  name: 'SpaceCore',
  fileName: 'a.zip',
  version: '1',
  sizeKb: 1,
  state: 'done',
  progress: 100,
  speed: 0,
  error: '',
  repo: '',
  tag: '',
  asset: '',
  assets: null,
  unverified: false,
  ...over,
})

beforeEach(() => {
  calls.remove = []
  calls.roll = []
  useLaunch.setState(useLaunch.getInitialState(), true)
  useMods.setState({ ...useMods.getInitialState(), load: async () => undefined }, true)
  useProfiles.setState(
    {
      ...useProfiles.getInitialState(),
      game: {
        id: 'stardew',
        name: 'Stardew Valley',
        appId: '',
        loader: '',
        available: true,
        installed: true,
        installDir: '',
        artUrl: '',
      },
      openId: 'p1',
      profiles: [baseProfile()],
      refresh: async () => undefined,
    },
    true,
  )
})

test('an update rolls back; a new install is removed', () => {
  expect(shouldRollBack(item())).toBe(false)
  expect(shouldRollBack(item({ kind: 'update' }))).toBe(true)
  expect(entryForItem(baseProfile(), item())?.key).toBe('k1')
})

test('undo of a new install removes the entry', async () => {
  await undoInstall(item(), entry())
  expect(calls.remove).toEqual(['k1'])
  expect(calls.roll).toEqual([])
})

test('undo of an update rolls back', async () => {
  await undoInstall(item({ kind: 'update' }), entry())
  expect(calls.roll).toEqual(['k1'])
  expect(calls.remove).toEqual([])
})

test('undo does nothing while the profile is locked', async () => {
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
  await undoInstall(item(), entry())
  expect(calls.remove).toEqual([])
})

test('a lone Nexus download failure is the one retried', () => {
  expect(singleNexusFailure([item({ state: 'failed', repo: '' })])?.id).toBe('q1')
  expect(singleNexusFailure([item({ state: 'failed', repo: 'me/mod' })])).toBeUndefined()
  expect(
    singleNexusFailure([item({ id: 'a', state: 'failed' }), item({ id: 'b', state: 'failed' })]),
  ).toBeUndefined()
})
