import { expect, test } from 'bun:test'
import { winnable } from './winnable.ts'

test('winnable leaves out a pack that another of the conflict waits for', () => {
  const mods = [
    { id: 'Me.Cape', needs: [] },
    { id: 'Me.Annetta', needs: ['me.cape'] },
    { id: 'Me.Other', needs: [] },
  ]
  expect(winnable(mods, ['Me.Cape', 'Me.Annetta', 'Me.Other'])).toEqual([1, 2])
})

test('winnable counts optional needs and the framework a pack is for', () => {
  const mods = [
    { id: 'Me.A', needs: ['Me.B'] },
    { id: 'Me.B', contentPackFor: 'Me.A' },
  ]
  expect(winnable(mods, ['Me.A', 'Me.B'])).toEqual([])
})
