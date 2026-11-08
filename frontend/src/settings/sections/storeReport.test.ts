import { expect, test } from 'bun:test'
import type {
  Item as LeftoverItem,
  Preview,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'
import type {
  Item,
  Report,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/store/models.ts'
import { groupLeftovers, removable } from './storeReport.ts'

const SEP = '\u0000'

const item = (key: string, over: Partial<Item> = {}): Item => ({
  key,
  name: key,
  version: '1.0.0',
  lastUsed: '2026-01-01T00:00:00Z',
  size: 1024,
  inUse: false,
  ...over,
})

const leftover = (kind: string, rel: string, size: number): LeftoverItem => ({
  kind,
  label: rel,
  rel,
  size,
})

const preview = (items: LeftoverItem[]): Preview => ({
  items,
  total: items.reduce((n, i) => n + i.size, 0),
})

test('lists an unused item and every copy of a duplicate but the newest', () => {
  const rep: Report = {
    stardew: {
      unused: [item('local-1')],
      duplicates: [
        [
          item('nexus-1-1', { lastUsed: '2026-02-01T00:00:00Z' }),
          item('nexus-1-2', { lastUsed: '2026-01-01T00:00:00Z' }),
        ],
      ],
    },
  }
  const { unused, older } = removable(rep)
  expect(unused.map((s) => s.id)).toEqual([`stardew${SEP}local-1`])
  expect(older.map((s) => s.id)).toEqual([`stardew${SEP}nexus-1-2`])
})

test('keeps every copy a profile uses, even an older one', () => {
  const rep: Report = {
    lethalcompany: {
      unused: [],
      duplicates: [
        [
          item('nexus-7-1', { lastUsed: '2026-03-01T00:00:00Z' }),
          item('nexus-7-2', { lastUsed: '2026-01-01T00:00:00Z', inUse: true }),
        ],
        [item('nexus-8-1', { inUse: true }), item('nexus-8-2', { inUse: true })],
      ],
    },
  }
  const { unused, older } = removable(rep)
  expect(unused).toEqual([])
  expect(older.map((s) => s.id)).toEqual([`lethalcompany${SEP}nexus-7-1`])
  expect(older.map((s) => s.item.key)).toEqual(['nexus-7-1'])
})

test('ignores a duplicate group of one and an empty report', () => {
  const rep: Report = {
    stardew: { unused: [], duplicates: [[item('only')], null, []] },
  }
  expect(removable(rep)).toEqual({ unused: [], older: [] })
  expect(removable(null)).toEqual({ unused: [], older: [] })
})

test('groups leftovers by where they live, largest group first', () => {
  const groups = groupLeftovers(
    preview([
      leftover('cache', 'cache/thunderstore/1.json', 100),
      leftover('cache', 'cache/nexus/2.json', 300),
      leftover('cache', 'cache/github-releases/3.json', 200),
      leftover('temp', 'temp/unpack', 50),
      leftover('temp', 'cache/nexus/half-download', 25),
    ]),
  )
  // A temp item stays temp even where its path looks like a cached Nexus file.
  expect(groups.map((g) => [g.kind, g.items.length, g.size])).toEqual([
    ['nexus', 1, 300],
    ['github', 1, 200],
    ['cache', 1, 100],
    ['temp', 2, 75],
  ])
})

test('a store item belongs to no leftover group', () => {
  expect(groupLeftovers(preview([leftover('store', 'store/stardew', 4096)]))).toEqual([])
  expect(groupLeftovers(null)).toEqual([])
})
