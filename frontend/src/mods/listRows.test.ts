import { expect, test } from 'bun:test'
import type {
  Entry,
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { i18n } from '../i18n/index.ts'
import { sameLibraries, toListRows } from './listRows.ts'

i18n.activate('en')

const mod = (key: string) =>
  ({ key, id: key.toUpperCase(), name: key, enabled: true }) as unknown as Mod
const entry = (key: string, source: Record<string, unknown>) =>
  ({ key, source: { name: key, ...source } }) as unknown as Entry
const data = (over: Partial<Parameters<typeof toListRows>[2]> = {}) => ({
  byId: {},
  customById: new Map(),
  sizes: {},
  costs: {},
  wins: {},
  facts: {},
  ...over,
})
const rows = (profile: Partial<Profile>, mods: Mod[], d = data()) =>
  toListRows(mods, profile as Profile, d)

test('a Nexus entry with no page or file is unsourced with its reason', () => {
  const [page, file, ok] = rows(
    {
      entries: [
        entry('a', { kind: 'nexus', modId: 0 }),
        entry('b', { kind: 'nexus', modId: 5, fileId: 0 }),
        entry('c', { kind: 'nexus', modId: 5, fileId: 9 }),
      ],
    },
    [mod('a'), mod('b'), mod('c')],
  )
  expect(page?.source).toBe('Unknown source')
  expect(page?.unsourced).toBe(true)
  expect(file?.source).toContain('file unknown')
  expect(file?.unsourced).toBe(true)
  expect(ok?.unsourced).toBe(false)
})

test('a key in two groups takes the first group, and order is the 1-based entry index', () => {
  const [a, b, missing] = rows(
    {
      entries: [entry('a', { kind: 'archive' }), entry('b', { kind: 'archive' })],
      groups: [
        { name: 'One', keys: ['b'] },
        { name: 'Two', keys: ['a', 'b'] },
      ],
    },
    [mod('a'), mod('b'), mod('z')],
  )
  expect(a?.groupName).toBe('Two')
  expect(b?.groupName).toBe('One')
  expect([a?.order, b?.order, missing?.order]).toEqual([1, 2, 0])
})

test('size and startupMs are absent when the maps lack the key', () => {
  const profile = { entries: [entry('a', { kind: 'archive' })] }
  const [bare] = rows(profile, [mod('a')])
  expect('size' in (bare ?? {})).toBe(false)
  expect('startupMs' in (bare ?? {})).toBe(false)
  const [full] = rows(profile, [mod('a')], data({ sizes: { a: 7 }, costs: { a: 3 } }))
  expect(full?.size).toBe(7)
  expect(full?.startupMs).toBe(3)
})

test('sameLibraries compares by identity, then length and mod order', () => {
  const a = mod('a')
  const b = mod('b')
  const lib = (...mods: Mod[]) => mods.map((m) => ({ mod: m })) as never
  const one = lib(a, b)
  expect(sameLibraries(one, one)).toBe(true)
  expect(sameLibraries(lib(a, b), lib(a, b))).toBe(true)
  expect(sameLibraries(lib(a, b), lib(b, a))).toBe(false)
  expect(sameLibraries(undefined, undefined)).toBe(true)
  expect(sameLibraries(undefined, lib(a))).toBe(false)
  expect(sameLibraries(lib(a), lib(a, b))).toBe(false)
})
