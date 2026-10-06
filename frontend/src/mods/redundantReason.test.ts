import { expect, test } from 'bun:test'
import { sameJobGroups } from './redundantReason.ts'

test('same-job pairs join into one group per job', () => {
  const pair = (key: string, others: string[], detail = 'Farmer.CurrentToolIndex') => ({
    kind: 'sameJob',
    key,
    id: key,
    name: key,
    by: others.map((k) => ({ key: k, name: k })),
    detail,
  })
  const groups = sameJobGroups([
    pair('a', ['b', 'c']),
    pair('b', ['a', 'c']),
    pair('c', ['a', 'b']),
    pair('x', ['y'], 'NPC.hasBeenKissedToday'),
    pair('y', ['x'], 'NPC.hasBeenKissedToday'),
  ])
  expect(groups).toEqual([
    { keys: ['a', 'b', 'c'], detail: 'Farmer.CurrentToolIndex' },
    { keys: ['x', 'y'], detail: 'NPC.hasBeenKissedToday' },
  ])
})

test('a pair listed once still forms a group with the mod that only appears in by', () => {
  const groups = sameJobGroups([
    {
      kind: 'sameJob',
      key: 'a',
      id: 'a',
      name: 'a',
      by: [{ key: 'b', name: 'b' }],
      detail: 'Farmer.CurrentToolIndex',
    },
  ])
  expect(groups).toEqual([{ keys: ['a', 'b'], detail: 'Farmer.CurrentToolIndex' }])
})
