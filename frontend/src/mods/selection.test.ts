import { expect, test } from 'bun:test'
import {
  applyClick,
  clearSelection,
  getInitialState,
  rangeRows,
  selectAllRows,
  toggleRow,
  useSelection,
} from './selection.ts'

test('toggle adds and removes without dropping other ids', () => {
  useSelection.setState(useSelection.getInitialState(), true)
  const one = toggleRow(getInitialState(), 'b')
  expect(one.ids).toEqual(['b'])
  expect(toggleRow(one, 'a').ids).toEqual(['b', 'a'])
  expect(toggleRow(toggleRow(one, 'a'), 'a').ids).toEqual(['b'])
})

test('range uses current sort order between the anchor and the click', () => {
  useSelection.setState(useSelection.getInitialState(), true)
  const ordered = ['a', 'c', 'b', 'd']
  const fromC = applyClick(getInitialState(), ordered, 'c', {
    shiftKey: false,
    ctrlKey: false,
    metaKey: false,
  })
  expect(fromC.ids).toEqual(['c'])
  expect(rangeRows(fromC, ordered, 'd').ids).toEqual(['c', 'b', 'd'])
  expect(rangeRows(fromC, ordered, 'a').ids).toEqual(['a', 'c'])
})

test('select all covers only the visible filtered ids', () => {
  useSelection.setState(useSelection.getInitialState(), true)
  expect(selectAllRows(['x', 'y']).ids).toEqual(['x', 'y'])
  expect(clearSelection().ids).toEqual([])
})

test('prune does not replace state when every selected id is still visible', () => {
  useSelection.setState({ ids: ['a', 'b'], anchor: 'a' })
  const before = useSelection.getState()
  useSelection.getState().prune(['a', 'b', 'c'])
  expect(useSelection.getState()).toBe(before)
})
