import { expect, test } from 'bun:test'
import type {
  HistoryDiff,
  HistoryItem,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { diffLines, itemModKey, selectedPair } from './historyDiff.ts'

const items: HistoryItem[] = [
  { kind: 'added', mod: 'me.b', name: 'Beta', key: 'b', detail: 'added Beta' },
  {
    kind: 'updated',
    mod: 'me.a',
    name: 'Alpha',
    key: 'a2',
    old: '1',
    new: '2',
    detail: 'Alpha 1 → 2',
  },
  {
    kind: 'disabled',
    mod: 'me.c',
    name: 'Gamma',
    key: 'c',
    old: 'enabled',
    new: 'disabled',
    detail: 'Gamma enabled → disabled',
  },
  {
    kind: 'config',
    mod: 'me.d',
    name: 'Delta',
    key: 'd',
    file: 'config.json',
    detail: 'Delta config.json',
  },
]

const sample: HistoryDiff = {
  a: 'one',
  b: 'two',
  added: [{ id: 'me.b', name: 'Beta', version: '1', key: 'b' }],
  removed: [],
  versions: [{ id: 'me.a', name: 'Alpha', old: '1', new: '2', oldKey: 'a1', newKey: 'a2' }],
  enabled: [{ id: 'me.c', name: 'Gamma', key: 'c', old: true, new: false }],
  configs: [{ id: 'me.d', name: 'Delta', key: 'd', files: ['config.json'] }],
  items,
}

test('diffLines lists each item detail', () => {
  expect(diffLines(sample)).toEqual([
    'added Beta',
    'Alpha 1 → 2',
    'Gamma enabled → disabled',
    'Delta config.json',
  ])
  expect(diffLines({ ...sample, items: [] })).toEqual([])
})

test('selectedPair needs two snapshot ids', () => {
  expect(selectedPair(['a'])).toBeNull()
  expect(selectedPair(['a', 'b'])).toEqual(['a', 'b'])
})

test('itemModKey uses config path when present', () => {
  expect(itemModKey(items[0] as HistoryItem)).toBe('me.b')
  expect(itemModKey(items[3] as HistoryItem)).toBe('me.d/config.json')
})
