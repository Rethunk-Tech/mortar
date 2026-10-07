import { afterEach, beforeEach, expect, test } from 'bun:test'
import { choose, lanOriginOf, mappedProfile, rememberLanProfile } from './resume.ts'

const store = new Map<string, string>()
const original = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')

beforeEach(() => {
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: {
      getItem: (k: string) => store.get(k) ?? null,
      setItem: (k: string, v: string) => {
        store.set(k, v)
      },
    },
  })
})
afterEach(() => {
  store.clear()
  if (original) {
    Object.defineProperty(globalThis, 'localStorage', original)
  } else {
    Reflect.deleteProperty(globalThis, 'localStorage')
  }
})

const origin = { game: 'stardew', senderId: 'pc-1', profileId: 'p-9' }
const farm = { id: 'local-1', name: 'Farm' }

test('applying a share records where it went, and the next share of that profile finds it', () => {
  expect(mappedProfile(origin, [farm])).toBeUndefined()
  rememberLanProfile(origin, 'local-1')
  expect(mappedProfile(origin, [farm])).toEqual(farm)
  expect(mappedProfile({ ...origin, senderId: 'pc-2' }, [farm])).toBeUndefined()
})

test('a mapped profile that was deleted is ignored', () => {
  rememberLanProfile(origin, 'local-1')
  expect(mappedProfile(origin, [{ id: 'other', name: 'Other' }])).toBeUndefined()
})

test('only a share naming its sender and profile can be resumed', () => {
  expect(lanOriginOf({ game: 'stardew', senderId: 'pc-1' })).toBeUndefined()
  expect(lanOriginOf({ game: 'stardew', senderId: 'pc-1', profileId: 'p-9' })).toEqual(origin)
})

test('the primary action updates the mapped profile, else makes a new one', () => {
  expect(choose(farm)).toEqual({ kind: 'update', profileId: 'local-1', name: 'Farm' })
  expect(choose(undefined)).toEqual({ kind: 'new' })
})
