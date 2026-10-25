import { expect, test } from 'bun:test'
import { resolveOverride } from './overrideValue.ts'

test('resolveOverride prefers a stored profile value over the game setting', () => {
  expect(resolveOverride('defaultLaunchMethod', 'steam', undefined)).toBe('steam')
  expect(resolveOverride('defaultLaunchMethod', 'steam', { defaultLaunchMethod: 'direct' })).toBe(
    'direct',
  )
})
