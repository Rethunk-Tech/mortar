import { expect, test } from 'bun:test'
import { goldText, hoursPlayed } from './card.ts'

test('playtime is whole hours from millisecondsPlayed', () => {
  expect(hoursPlayed(151_200_000)).toBe(42)
  expect(hoursPlayed(0)).toBe(0)
})

test('money uses grouping and a g suffix', () => {
  expect(goldText(125_300)).toBe('125,300g')
})
