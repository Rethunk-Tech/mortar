import { expect, test } from 'bun:test'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { entryFavouriteKey, favouriteKey, isStarred } from './favourites.ts'

const entry = (source: Partial<Entry['source']>) =>
  ({
    key: 'k',
    previousKey: '',
    mods: [],
    disabled: [],
    source: { kind: 'local', name: '', ...source },
  }) as Entry

test('an installed mod and its Browse hit share one key', () => {
  expect(entryFavouriteKey(entry({ kind: 'nexus', modId: 42 }))).toBe(favouriteKey('nexus', '42'))
  expect(entryFavouriteKey(entry({ kind: 'github', repo: 'Me/Mod' }))).toBe('github:me/mod')
  expect(entryFavouriteKey(entry({ kind: 'thunderstore', name: 'Me-Mod' }))).toBe(
    'thunderstore:me-mod',
  )
  expect(entryFavouriteKey(entry({ kind: 'nexus' }))).toBe('')
  expect(entryFavouriteKey(undefined)).toBe('')
})

test('a mod is starred when the game lists its key', () => {
  expect(isStarred(['nexus:42'], 'nexus:42')).toBe(true)
  expect(isStarred(['nexus:42'], 'nexus:43')).toBe(false)
  expect(isStarred(undefined, 'nexus:42')).toBe(false)
  expect(isStarred(['nexus:42'], '')).toBe(false)
})

test('local archives of one file name have their own star, and a 0.3.8 star still reads', () => {
  const a = entryFavouriteKey({ ...entry({ kind: 'local', name: 'A.zip' }), key: 'k1' })
  const b = entryFavouriteKey({ ...entry({ kind: 'local', name: 'A.zip' }), key: 'k2' })
  expect(a).not.toBe(b)
  expect(isStarred([a], b)).toBe(false)
  expect(isStarred(['local:a.zip'], a)).toBe(true)
})
