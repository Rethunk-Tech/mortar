import { expect, test } from 'bun:test'
import { hasThunderstore, packageLine } from './packImport.ts'

test('only a game with a Thunderstore source offers the pack import', () => {
  expect(hasThunderstore({ sources: ['nexus', 'thunderstore'] })).toBe(true)
  expect(hasThunderstore({ sources: ['nexus'] })).toBe(false)
  expect(hasThunderstore({ sources: null })).toBe(false)
  expect(hasThunderstore(null)).toBe(false)
})

test('a pack package reads as name and version, marked when the pack switched it off', () => {
  const p = { source: 'thunderstore', native: 'Ns-Mod', version: '1.2.3', disabled: false }
  expect(packageLine(p, 'off')).toBe('Ns-Mod 1.2.3')
  expect(packageLine({ ...p, disabled: true }, 'off')).toBe('Ns-Mod 1.2.3 · off')
})
