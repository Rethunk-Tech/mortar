import { expect, test } from 'bun:test'
import { isAbstained, isEndorsed, isTracked } from './nexusAccount.ts'

test('endorsement status matches Nexus endorse_status values', () => {
  expect(isEndorsed('Endorsed')).toBe(true)
  expect(isEndorsed('Abstained')).toBe(false)
  expect(isEndorsed('Undecided')).toBe(false)
  expect(isEndorsed('')).toBe(false)
  expect(isAbstained('Abstained')).toBe(true)
  expect(isAbstained('Endorsed')).toBe(false)
})

test('tracked list matches this game by mod id', () => {
  const mods = [
    { modId: 541, domainName: 'stardewvalley' },
    { modId: 1, domainName: 'skyrim' },
  ]
  expect(isTracked(mods, 541, 'stardewvalley')).toBe(true)
  expect(isTracked(mods, 1, 'stardewvalley')).toBe(false)
  expect(isTracked(undefined, 541, 'stardewvalley')).toBe(false)
})
