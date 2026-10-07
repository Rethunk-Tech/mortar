import { expect, test } from 'bun:test'
import type {
  Component,
  Entry,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { applyPlan, compareProfiles, compareView } from './compare.ts'
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

test('the same mod from two sources is one mod with different sources', () => {
  const a = testProfile({
    id: 'a',
    name: 'A',
    entries: [entry('x', [mod('Me.More', 'More', '1.0')])],
  })
  const fromThunderstore = {
    ...entry('y', [mod('Me.More', 'More', '1.0')]),
    source: { kind: 'thunderstore', name: 'Me-More', version: '1.0' },
  }
  const d = compareProfiles(a, testProfile({ id: 'b', name: 'B', entries: [fromThunderstore] }))
  expect(d.differentSource.map((p) => p.id)).toEqual(['Me.More'])
  expect([d.onlyA, d.onlyB, d.identical]).toEqual([[], [], []])
})

test('compareView hides only-in-A and identical until shown; applyPlan picks per kind and direction', () => {
  const a = testProfile({
    id: 'a',
    name: 'A',
    entries: [
      entry('only-a', [mod('Me.OnlyA', 'Only A', '1.0')]),
      entry('ver-a', [mod('Me.Ver', 'Version', '1.0')]),
      entry('same', [mod('Me.Same', 'Same', '1.0')]),
    ],
  })
  const b = testProfile({
    id: 'b',
    name: 'B',
    entries: [
      entry('only-b', [mod('Me.OnlyB', 'Only B', '1.0')]),
      entry('ver-b', [mod('Me.Ver', 'Version', '2.0')]),
      entry('same', [mod('Me.Same', 'Same', '1.0')]),
    ],
  })
  const diff = compareProfiles(a, b)
  const hidden = compareView(diff, '', false)
  expect(hidden.groups.map((g) => g.kind)).toEqual(['version', 'onlyB'])
  expect([hidden.differences, hidden.onlyA, hidden.identical]).toEqual([2, 1, 1])
  const all = compareView(diff, '', true)
  expect(all.groups.map((g) => g.kind)).toEqual(['version', 'onlyB', 'onlyA', 'identical'])
  expect(compareView(diff, 'only b', true).groups.map((g) => g.kind)).toEqual(['onlyB'])
  const rows = all.groups.flatMap((g) => g.rows)
  expect(applyPlan(rows, true)).toEqual({
    copy: ['Me.OnlyA'],
    moves: [{ oldKey: 'ver-b', newKey: 'ver-a' }],
  })
  expect(applyPlan(rows, false)).toEqual({
    copy: ['Me.OnlyB'],
    moves: [{ oldKey: 'ver-a', newKey: 'ver-b' }],
  })
})
