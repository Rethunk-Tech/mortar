import { expect, test } from 'bun:test'
import { matchPaletteItems, type PaletteItem } from './match.ts'

const items: PaletteItem[] = [
  { id: 'action:play', kind: 'action', label: 'Play' },
  { id: 'profile:abc', kind: 'profile', label: 'Cookie farm', hint: 'Open profile' },
  {
    id: 'mod:k/Pathoschild.ContentPatcher',
    kind: 'mod',
    label: 'Content Patcher',
    hint: 'Open mod',
    match: 'Pathoschild.ContentPatcher',
  },
  { id: 'settings:nexus', kind: 'settings', label: 'Nexus Mods', hint: 'Settings' },
]

test('empty query keeps every item, sorted by label', () => {
  expect(matchPaletteItems(items, '').map((i) => i.id)).toEqual([
    'mod:k/Pathoschild.ContentPatcher',
    'profile:abc',
    'settings:nexus',
    'action:play',
  ])
})

test('matches label, UniqueID and hint fuzzily', () => {
  expect(matchPaletteItems(items, 'cook').map((i) => i.id)).toEqual(['profile:abc'])
  expect(matchPaletteItems(items, 'contentpatcher').map((i) => i.id)).toEqual([
    'mod:k/Pathoschild.ContentPatcher',
  ])
  expect(matchPaletteItems(items, 'pathoschild').map((i) => i.id)).toEqual([
    'mod:k/Pathoschild.ContentPatcher',
  ])
  expect(matchPaletteItems(items, 'settings').map((i) => i.id)).toEqual(['settings:nexus'])
})

test('ranks an exact label above a fuzzy hit', () => {
  const ranked = matchPaletteItems(
    [...items, { id: 'action:other', kind: 'action', label: 'Playtime' }],
    'Play',
  )
  expect(ranked[0]?.id).toBe('action:play')
})

test('ranks contiguous and word-start matches above scattered matches', () => {
  const ranked = matchPaletteItems(
    [
      { id: 'scattered', kind: 'action', label: 'C o n' },
      { id: 'word', kind: 'action', label: 'Console settings' },
      { id: 'contiguous', kind: 'action', label: 'Console' },
    ],
    'con',
  )
  expect(ranked.map((item) => item.id)).toEqual(['contiguous', 'word', 'scattered'])
})
