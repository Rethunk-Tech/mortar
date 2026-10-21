import { expect, test } from 'bun:test'
import { hasThunderstore } from './hasThunderstore.ts'

test('only a game with a Thunderstore source offers Thunderstore sharing', () => {
  expect(hasThunderstore({ sources: ['nexus', 'thunderstore'] })).toBe(true)
  expect(hasThunderstore({ sources: ['nexus'] })).toBe(false)
  expect(hasThunderstore({ sources: null })).toBe(false)
  expect(hasThunderstore(null)).toBe(false)
})
