import { expect, test } from 'bun:test'
import {
  clipDescription,
  colorHex,
  isProfileColor,
  isProfileIcon,
  MAX_DESCRIPTION,
} from './appearance.ts'

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

test('description is trimmed and capped at 280', () => {
  expect(clipDescription('  co-op  ')).toBe('co-op')
  expect(clipDescription('é'.repeat(MAX_DESCRIPTION + 8)).length).toBe(MAX_DESCRIPTION)
})
