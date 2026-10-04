import { expect, test } from 'bun:test'
import { lockedIn } from './useLocked.ts'

test('the starting window locks only the profile being started', () => {
  const idle = { status: null, starting: true, startingProfile: 'a' }
  expect(lockedIn(idle, 'a')).toBe(true)
  expect(lockedIn(idle, 'b')).toBe(false)
  expect(lockedIn({ ...idle, starting: false }, 'a')).toBe(false)
})
