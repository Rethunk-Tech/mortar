import { expect, test } from 'bun:test'
import { keyAfterRollBack, useDetail, versionAfterRollBack } from './detail.ts'

test('redo update targets the entry key after a roll back', () => {
  expect(keyAfterRollBack([{ key: 'old', mods: [{ id: 'SpaceCore' }] }], 'SpaceCore', 'new')).toBe(
    'old',
  )
  expect(keyAfterRollBack(undefined, 'SpaceCore', 'new')).toBe('new')
})

test('roll-back body uses the version now in the profile', () => {
  expect(
    versionAfterRollBack([{ mods: [{ id: 'SpaceCore', version: '1.2.0' }] }], 'SpaceCore'),
  ).toBe('1.2.0')
  expect(versionAfterRollBack(undefined, 'SpaceCore')).toBeUndefined()
})

test('showAfterLoad keeps a pending id that takePending consumes once', () => {
  useDetail.setState(useDetail.getInitialState(), true)
  useDetail.getState().showAfterLoad({ key: 'k', id: 'A.Mod' })
  expect(useDetail.getState().pendingId).toBe('k/A.Mod')
  expect(useDetail.getState().takePending()).toBe('k/A.Mod')
  expect(useDetail.getState().pendingId).toBe('')
})
