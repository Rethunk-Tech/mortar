import { expect, test } from 'bun:test'
import { settledLastProfile } from './lastProfile.ts'

test('a remembered profile that is gone falls back to the first remaining one, or to none', () => {
  const list = [{ id: 'a' }, { id: 'b' }]
  expect(settledLastProfile(list, 'b')).toBe('b')
  expect(settledLastProfile(list, 'gone')).toBe('a')
  expect(settledLastProfile(list, undefined)).toBe('a')
  expect(settledLastProfile([], 'gone')).toBe('')
})
