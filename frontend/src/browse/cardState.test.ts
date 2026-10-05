import { expect, test } from 'bun:test'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import { cardState, inProfile, shownState } from './cardState.ts'

const item = (over: Partial<Item>): Item =>
  ({
    id: 'a',
    kind: 'install',
    profileId: 'p',
    modId: 0,
    repo: '',
    state: 'queued',
    progress: 0,
    ...over,
  }) as Item

test('a card follows its own item by source and id', () => {
  const items = [
    item({ id: 'n', modId: 7, state: 'waiting-click' }),
    item({ id: 'm', source: 'modrinth', package: 'AANob', state: 'downloading', progress: 41.6 }),
    item({ id: 'd', kind: 'dependency', source: 'modrinth', package: 'AANob', state: 'failed' }),
    item({ id: 't', package: 'Me-Mod', state: 'installing' }),
    item({ id: 'o', profileId: 'other', repo: 'a/b', state: 'done' }),
  ]
  expect(cardState(items, 'nexus', '7', 'p')).toEqual({ kind: 'waiting-nexus' })
  expect(cardState(items, 'modrinth', 'AANob', 'p')).toEqual({ kind: 'downloading', percent: 42 })
  expect(cardState(items, 'thunderstore', 'me-mod', 'p')).toEqual({ kind: 'installing' })
  expect(cardState(items, 'github', 'a/b', 'p')).toEqual({ kind: 'idle' })
  expect(cardState(items, 'itch', '1', 'p')).toEqual({ kind: 'idle' })
})

test('final and failed states', () => {
  const at = (state: string) =>
    cardState([item({ id: 'x', repo: 'a/b', state })], 'github', 'a/b', 'p')
  expect(at('done')).toEqual({ kind: 'done' })
  expect(at('failed')).toEqual({ kind: 'failed', itemId: 'x' })
  expect(at('skipped')).toEqual({ kind: 'idle' })
  expect(at('needs-choice')).toEqual({ kind: 'attention' })
})

test('the profile outranks finished items, and only active items outrank the profile', () => {
  const done = { kind: 'done' } as const
  const failed = { kind: 'failed', itemId: 'x' } as const
  const queued = { kind: 'queued' } as const
  expect(shownState(queued, true, true)).toEqual(queued)
  expect(shownState(failed, true, false)).toEqual({ kind: 'idle' })
  expect(shownState(done, true, true)).toEqual({ kind: 'idle' })
  expect(shownState(done, false, false)).toEqual({ kind: 'idle' })
  expect(shownState(done, false, true)).toEqual(done)
  expect(shownState(failed, false, false)).toEqual(failed)
})

test('inProfile matches an entry by its own source', () => {
  const entry = (kind: string, rest: object) => ({ source: { kind, name: '', ...rest } })
  const profile = {
    entries: [
      entry('nexus', { modId: 5 }),
      entry('github', { repo: 'Owner/Mod' }),
      entry('modrinth', { name: 'Sodium' }),
    ],
  } as unknown as Parameters<typeof inProfile>[0]
  expect(inProfile(profile, 'nexus', '5')).toBe(true)
  expect(inProfile(profile, 'nexus', '6')).toBe(false)
  expect(inProfile(profile, 'github', 'owner/mod')).toBe(true)
  expect(inProfile(profile, 'modrinth', 'sodium')).toBe(true)
  expect(inProfile(profile, 'itch', 'sodium')).toBe(false)
  expect(inProfile(undefined, 'nexus', '5')).toBe(false)
})
