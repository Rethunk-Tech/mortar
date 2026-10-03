import { expect, test } from 'bun:test'
import { orderProfiles } from './profileOrder.ts'

const profiles = [
  { id: 'b', name: 'Beta' },
  { id: 'a', name: 'Alpha' },
  { id: 'c', name: 'Gamma' },
]

test('manual keeps the given order', () => {
  expect(orderProfiles(profiles, 'manual', 'c').map((p) => p.id)).toEqual(['b', 'a', 'c'])
})

test('name sorts alphabetically', () => {
  expect(orderProfiles(profiles, 'name').map((p) => p.id)).toEqual(['a', 'b', 'c'])
})

test('lastPlayed puts the settings last-played profile first', () => {
  expect(orderProfiles(profiles, 'lastPlayed', 'c').map((p) => p.id)).toEqual(['c', 'b', 'a'])
})
