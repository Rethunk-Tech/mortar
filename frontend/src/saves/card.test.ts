import { expect, test } from 'bun:test'
import { farmTypeName, goldText, hoursPlayed } from './card.ts'

test('farm type ids match Stardew whichFarm', () => {
  expect(farmTypeName(0)).toBe('Standard')
  expect(farmTypeName(1)).toBe('Riverland')
  expect(farmTypeName(2)).toBe('Forest')
  expect(farmTypeName(3)).toBe('Hill-top')
  expect(farmTypeName(4)).toBe('Wilderness')
  expect(farmTypeName(5)).toBe('Four Corners')
  expect(farmTypeName(6)).toBe('Beach')
  expect(farmTypeName(7)).toBe('Meadowlands')
  expect(farmTypeName(-1)).toBe('')
  expect(farmTypeName(8)).toBe('')
})

test('playtime is whole hours from millisecondsPlayed', () => {
  expect(hoursPlayed(151_200_000)).toBe(42)
  expect(hoursPlayed(0)).toBe(0)
})

test('money uses grouping and a g suffix', () => {
  expect(goldText(125_300)).toBe('125,300g')
})
