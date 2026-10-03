import { expect, test } from 'bun:test'
import { applyRow, choiceFromOverride, resolveOverride, rowValue } from './overrideValue.ts'

test('rowValue maps use-game to unset and an explicit value to that string', () => {
  expect(rowValue({ useGame: true })).toBeUndefined()
  expect(rowValue({ useGame: false, value: 'direct' })).toBe('direct')
  expect(choiceFromOverride(undefined)).toEqual({ useGame: true })
  expect(choiceFromOverride('')).toEqual({ useGame: true })
  expect(choiceFromOverride('direct')).toEqual({ useGame: false, value: 'direct' })
  expect(
    applyRow({ defaultLaunchMethod: 'direct' }, 'defaultLaunchMethod', { useGame: true }),
  ).toEqual({})
  expect(applyRow({}, 'defaultLaunchMethod', { useGame: false, value: 'direct' })).toEqual({
    defaultLaunchMethod: 'direct',
  })
})

test('resolveOverride prefers a stored profile value over the game setting', () => {
  expect(resolveOverride('defaultLaunchMethod', 'steam', undefined)).toBe('steam')
  expect(resolveOverride('defaultLaunchMethod', 'steam', { defaultLaunchMethod: 'direct' })).toBe(
    'direct',
  )
})
