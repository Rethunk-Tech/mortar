import { expect, test } from 'bun:test'
import { colorHex, isProfileColor, isProfileIcon } from './appearance.ts'

test('only palette tokens have a colour', () => {
  expect(isProfileColor('teal')).toBe(true)
  expect(isProfileColor('neon')).toBe(false)
  expect(colorHex('teal')).toBe('#4db6ac')
  expect(colorHex('neon')).toBeUndefined()
  expect(colorHex(undefined)).toBeUndefined()
})

test('only curated Lucide names are icons', () => {
  expect(isProfileIcon('sprout')).toBe(true)
  expect(isProfileIcon('dragon')).toBe(false)
})
