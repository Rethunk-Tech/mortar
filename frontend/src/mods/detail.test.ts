import { expect, test } from 'bun:test'
import { keyAfterRollBack, useDetail } from './detail.ts'

test('redo update targets the entry key after a roll back', () => {
  expect(
    keyAfterRollBack([{ key: 'old', mods: [{ uniqueId: 'SpaceCore' }] }], 'SpaceCore', 'new'),
  ).toBe('old')
  expect(keyAfterRollBack(undefined, 'SpaceCore', 'new')).toBe('new')
})

test('showAfterLoad keeps a pending id that takePending consumes once', () => {
  useDetail.setState(useDetail.getInitialState(), true)
  useDetail.getState().showAfterLoad({ key: 'k', uniqueId: 'A.Mod' })
  expect(useDetail.getState().pendingId).toBe('k/A.Mod')
  expect(useDetail.getState().takePending()).toBe('k/A.Mod')
  expect(useDetail.getState().pendingId).toBe('')
})
