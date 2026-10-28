import { expect, test } from 'bun:test'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { listItems } from './gameList.ts'

const game = (id: string) => ({ id, name: id }) as unknown as GameInfo
const groups = [
  { id: 'recent', label: 'Recent', games: [game('a'), game('b')] },
  { id: 'empty', label: 'Empty', games: [] },
  { id: 'rest', label: 'Rest', games: [game('c')] },
]

test('an empty group is dropped and a heading carries the count', () => {
  const items = listItems(groups, new Set())
  expect(items.map((i) => i.key)).toEqual(['h:recent', 'a', 'b', 'h:rest', 'c'])
  expect(items[0]).toMatchObject({ kind: 'heading', label: 'Recent', count: 2, collapsed: false })
})

test('a collapsed group shows only its heading with the full count', () => {
  const items = listItems(groups, new Set(['recent']))
  expect(items.map((i) => i.key)).toEqual(['h:recent', 'h:rest', 'c'])
  expect(items[0]).toMatchObject({ count: 2, collapsed: true })
})
