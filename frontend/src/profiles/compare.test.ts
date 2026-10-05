import { expect, test } from 'bun:test'
import type {
  Component,
  Entry,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { compareProfiles } from './compare.ts'
import { testProfile } from './testProfile.ts'

function mod(id: string, name: string, version: string): Component {
  return { id, name, version, author: '', folder: '.' }
}

function entry(key: string, mods: Component[], disabled: string[] | null = null): Entry {
  return {
    key,
    previousKey: '',
    source: { kind: 'nexus', name: 'mod.zip', modId: 1 },
    mods,
    disabled,
  }
}

test('compare splits version, enabled, and identical mods in both profiles', () => {
  const a = testProfile({
    id: 'a',
    name: 'A',
    entries: [
      entry('only-a', [mod('Me.OnlyA', 'Only A', '1.0')]),
      entry('ver', [mod('Me.Ver', 'Version', '1.0')]),
      entry('en', [mod('Me.En', 'Enabled', '2.0')], ['Me.En']),
      entry('same', [mod('Me.Same', 'Same', '3.0')]),
    ],
  })
  const b = testProfile({
    id: 'b',
    name: 'B',
    entries: [
      entry('only-b', [mod('Me.OnlyB', 'Only B', '1.0')]),
      entry('ver-b', [mod('Me.Ver', 'Version', '2.0')]),
      entry('en-b', [mod('Me.En', 'Enabled', '2.0')]),
      entry('same-b', [mod('Me.Same', 'Same', '3.0')]),
    ],
  })

  const d = compareProfiles(a, b)
  expect(d.onlyA.map((m) => m.id)).toEqual(['Me.OnlyA'])
  expect(d.onlyB.map((m) => m.id)).toEqual(['Me.OnlyB'])
  expect(d.differentVersion.map((p) => p.id)).toEqual(['Me.Ver'])
  expect(d.differentEnabled.map((p) => p.id)).toEqual(['Me.En'])
  expect(d.identical.map((p) => p.id)).toEqual(['Me.Same'])
})

test('a mod with both version and enabled differences appears in both diff groups', () => {
  const a = testProfile({
    id: 'a',
    name: 'A',
    entries: [entry('x', [mod('Me.Both', 'Both', '1.0')])],
  })
  const b = testProfile({
    id: 'b',
    name: 'B',
    entries: [entry('y', [mod('Me.Both', 'Both', '2.0')], ['Me.Both'])],
  })
  const d = compareProfiles(a, b)
  expect(d.differentVersion.map((p) => p.id)).toEqual(['Me.Both'])
  expect(d.differentEnabled.map((p) => p.id)).toEqual(['Me.Both'])
  expect(d.identical).toEqual([])
})
