import { expect, test } from 'bun:test'
import { keyAfterRollBack } from './detail.ts'

test('redo update targets the entry key after a roll back', () => {
  expect(
    keyAfterRollBack([{ key: 'old', mods: [{ uniqueId: 'SpaceCore' }] }], 'SpaceCore', 'new'),
  ).toBe('old')
  expect(keyAfterRollBack(undefined, 'SpaceCore', 'new')).toBe('new')
})
