import { expect, test } from 'bun:test'
import { filterAndSortSaves } from './filterAndSortSaves.ts'

test('saves default to last played and filter by farm name', () => {
  const fits = [
    { farm: 'Old', folder: 'Old_1', played: 1 },
    { farm: 'New', folder: 'New_1', played: 3 },
    { farm: 'Mid', folder: 'Mid_1', played: 2 },
  ]
  expect(filterAndSortSaves(fits, '').map((fit) => fit.farm)).toEqual(['New', 'Mid', 'Old'])
  expect(filterAndSortSaves(fits, 'mid').map((fit) => fit.folder)).toEqual(['Mid_1'])
})
