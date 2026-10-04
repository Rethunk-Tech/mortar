import { expect, test } from 'bun:test'
import { sameJobGroups } from './redundantReason.ts'

test('same-job pairs join into one group per job', () => {
  const pair = (key: string, others: string[], detail = 'Farmer.CurrentToolIndex') => ({
    kind: 'sameJob',
    key,
    uniqueId: key,
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
