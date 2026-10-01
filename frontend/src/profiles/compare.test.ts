import { expect, test } from 'bun:test'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { compareProfiles } from './compare.ts'

function profile(entries: Profile['entries'], id = 'p', name = 'Profile'): Profile {
  return {
    id,
    name,
    entries,
    hidden: false,
    color: 'teal',
    icon: 'sprout',
  }
}

function entry(
  key: string,
  mods: { uniqueId: string; name: string; version: string }[],
  disabled: string[] = [],
) {
  return {
    key,
    previousKey: '',
    source: { kind: 'nexus', modId: 1 },
    mods: mods.map((m) => ({ ...m, folder: '.' })),
    disabled,
  }
}

test('compare splits version, enabled, and identical mods in both profiles', () => {
  const a = profile([
    entry('only-a', [{ uniqueId: 'Me.OnlyA', name: 'Only A', version: '1.0' }]),
    entry('ver', [{ uniqueId: 'Me.Ver', name: 'Version', version: '1.0' }]),
    entry('en', [{ uniqueId: 'Me.En', name: 'Enabled', version: '2.0' }], ['Me.En']),
    entry('same', [{ uniqueId: 'Me.Same', name: 'Same', version: '3.0' }]),
  ])
  const b = profile([
    entry('only-b', [{ uniqueId: 'Me.OnlyB', name: 'Only B', version: '1.0' }]),
    entry('ver-b', [{ uniqueId: 'Me.Ver', name: 'Version', version: '2.0' }]),
    entry('en-b', [{ uniqueId: 'Me.En', name: 'Enabled', version: '2.0' }]),
    entry('same-b', [{ uniqueId: 'Me.Same', name: 'Same', version: '3.0' }]),
  ])

  const d = compareProfiles(a, b)
  expect(d.onlyA.map((m) => m.uniqueId)).toEqual(['Me.OnlyA'])
  expect(d.onlyB.map((m) => m.uniqueId)).toEqual(['Me.OnlyB'])
  expect(d.differentVersion.map((p) => p.uniqueId)).toEqual(['Me.Ver'])
  expect(d.differentEnabled.map((p) => p.uniqueId)).toEqual(['Me.En'])
  expect(d.identical.map((p) => p.uniqueId)).toEqual(['Me.Same'])
})

test('a mod with both version and enabled differences appears in both diff groups', () => {
  const a = profile([entry('x', [{ uniqueId: 'Me.Both', name: 'Both', version: '1.0' }])])
  const b = profile([
    entry('y', [{ uniqueId: 'Me.Both', name: 'Both', version: '2.0' }], ['Me.Both']),
  ])
  const d = compareProfiles(a, b)
  expect(d.differentVersion.map((p) => p.uniqueId)).toEqual(['Me.Both'])
  expect(d.differentEnabled.map((p) => p.uniqueId)).toEqual(['Me.Both'])
  expect(d.identical).toEqual([])
})
