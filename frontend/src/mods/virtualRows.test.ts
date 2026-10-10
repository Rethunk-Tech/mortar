import { expect, test } from 'bun:test'
import { TYPE_RESET_MS } from '../shell/typeahead.ts'
import {
  flattenModGroups,
  gridColumnCount,
  groupKeyHolding,
  orderedModIds,
  stepId,
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

test('typed letters and digits add to the search until a pause, and typing in a field is left alone', () => {
  expect(typeaheadQuery('A', 0, 'u', TYPE_RESET_MS + 1)).toBe('u')
  expect(typeaheadQuery('A', 0, 'u', TYPE_RESET_MS - 1)).toBe('Au')
  const key = (k: string) => ({
    key: k,
    ctrlKey: false,
    altKey: false,
    metaKey: false,
    target: null,
  })
  expect(typeaheadChar(key('4'))).toBe('4')
  expect(typeaheadChar(key('.'))).toBeUndefined()
  expect(typeaheadChar(key(' '))).toBeUndefined()
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

test('a collapse saved under a grouping never hides the ungrouped list', () => {
  const rows = flattenModGroups([{ key: '', items: [{ id: 'a' }] }], {
    grouped: false,
    collapsed: { '': true },
    idOf,
  })
  expect(rows.map((row) => row.key)).toEqual(['r:a'])
})

// The card grid draws every card of the visible lanes as children of one grid, keyed by mod id, and lets CSS lay out the
// columns. That keeps each card mounted when the column count changes only while the cards come out in one order,
// whatever the count, and the headers keep their place between them.
test('the cards and headers flow in one order at every column count', () => {
  const enabled = Array.from({ length: 11 }, (_, i) => ({ id: `e${i}` }))
  const disabled = Array.from({ length: 5 }, (_, i) => ({ id: `d${i}` }))
  const many = [
    { key: 'enabled', items: enabled },
    { key: 'disabled', items: disabled },
  ]
  const flow = (columns: number) =>
    flattenModGroups(many, { grouped: true, collapsed: {}, idOf, columns }).flatMap((row) => {
      if (row.kind === 'header') {
        return [row.key]
      }
      return row.kind === 'lane' ? row.items.map(idOf) : []
    })
  const wide = flow(5)
  for (const columns of [1, 2, 3, 4, 6, 20]) {
    expect(flow(columns)).toEqual(wide)
  }
  expect(wide).toEqual(['h:enabled', ...enabled.map(idOf), 'h:disabled', ...disabled.map(idOf)])
})
