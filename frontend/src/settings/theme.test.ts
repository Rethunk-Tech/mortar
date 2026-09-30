import { expect, test } from 'bun:test'
import { isAccent } from './theme.ts'

test('isAccent rejects unknown names', () => {
  expect(isAccent('copper')).toBe(true)
  expect(isAccent('neon')).toBe(false)
})
