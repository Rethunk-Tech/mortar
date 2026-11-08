import { expect, test } from 'bun:test'
import { seasonName, yearSeasonLabel } from './homeView.ts'

test('a season index names its season, and one out of range names none', () => {
  expect(seasonName(0)).toBe('Spring')
  expect(seasonName(3)).toBe('Winter')
  expect(seasonName(4)).toBe('')
  expect(seasonName(-1)).toBe('')
})

test('a save date reads as its year and season', () => {
  expect(yearSeasonLabel(2, 1)).toBe('Year 2 Summer')
})
