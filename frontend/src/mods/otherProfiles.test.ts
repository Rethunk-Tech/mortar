import { expect, test } from 'bun:test'
import { selectableProfileIds } from './otherProfiles.ts'

test('selectableProfileIds drops unavailable ids', () => {
  expect(selectableProfileIds(['a', 'b', 'c'], new Set(['b']))).toEqual(['a', 'c'])
})
