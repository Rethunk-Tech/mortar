import { expect, test } from 'bun:test'
import {
  firstNamePrefix,
  flattenModGroups,
  gridColumnCount,
  groupKeyHolding,
  orderedModIds,
  stepId,
  TYPEAHEAD_MS,
  typeaheadChar,
  typeaheadQuery,
  virtualIndexOf,
} from './virtualRows.ts'

const groups = [
  { key: 'enabled', items: [{ id: 'a' }, { id: 'b' }, { id: 'c' }] },
  { key: 'disabled', items: [{ id: 'd' }] },
]
const idOf = (item: { id: string }) => item.id

test('flattens grouped list rows, skips collapsed items, lanes for the grid', () => {
  expect(flattenModGroups(groups, { grouped: false, collapsed: {}, idOf })).toEqual([
    { kind: 'row', key: 'r:a', groupKey: 'enabled', item: { id: 'a' }, stripe: false },
    { kind: 'row', key: 'r:b', groupKey: 'enabled', item: { id: 'b' }, stripe: true },
    { kind: 'row', key: 'r:c', groupKey: 'enabled', item: { id: 'c' }, stripe: false },
    { kind: 'row', key: 'r:d', groupKey: 'disabled', item: { id: 'd' }, stripe: false },
  ])
  const listed = flattenModGroups(groups, {
    grouped: true,
    collapsed: { disabled: true },
    idOf,
  })
  expect(listed.map((row) => [row.kind, row.key])).toEqual([
    ['header', 'h:enabled'],
    ['row', 'r:a'],
    ['row', 'r:b'],
    ['row', 'r:c'],
    ['header', 'h:disabled'],
  ])
  expect(
    flattenModGroups(groups, { grouped: true, collapsed: {}, idOf, columns: 2 }).map(
      (row) => row.key,
    ),
  ).toEqual(['h:enabled', 'l:enabled:0', 'l:enabled:2', 'h:disabled', 'l:disabled:0'])
  expect(gridColumnCount(0)).toBe(1)
  expect(gridColumnCount(332)).toBe(1)
  expect(gridColumnCount(638)).toBe(2)
  expect(groupKeyHolding(groups, (item) => item.id === 'd')).toBe('disabled')
  expect(virtualIndexOf(listed, 'c', idOf)).toBe(3)
  expect(orderedModIds(listed, idOf)).toEqual(['a', 'b', 'c'])
})

test('firstNamePrefix matches the first name that starts with the typed text', () => {
  const mods = [{ name: 'Automate' }, { name: 'Bigger Backpack' }, { name: 'Auto-Gravestones' }]
  expect(firstNamePrefix(mods, 'au', (m) => m.name)?.name).toBe('Automate')
  expect(firstNamePrefix(mods, 'bi', (m) => m.name)?.name).toBe('Bigger Backpack')
  expect(firstNamePrefix(mods, 'zz', (m) => m.name)).toBeUndefined()
  expect(typeaheadQuery('A', 0, 'u', TYPEAHEAD_MS + 1)).toBe('u')
  expect(typeaheadQuery('A', 0, 'u', TYPEAHEAD_MS - 1)).toBe('Au')
  expect(
    typeaheadChar({
      key: 'a',
      ctrlKey: false,
      altKey: false,
      metaKey: false,
      target: { tagName: 'INPUT', isContentEditable: false } as HTMLElement,
    }),
  ).toBeUndefined()
  expect(
    typeaheadChar({
      key: 'a',
      ctrlKey: false,
      altKey: false,
      metaKey: false,
      target: { tagName: 'DIV', isContentEditable: false } as HTMLElement,
    }),
  ).toBe('a')
})

test('stepId clamps at both ends instead of wrapping', () => {
  expect(stepId(['a', 'b', 'c', 'd'], 'b', 3)).toBe('d')
  expect(stepId(['a', 'b', 'c', 'd'], 'b', -3)).toBe('a')
  expect(stepId(['a', 'b'], 'x', 1)).toBeUndefined()
})
