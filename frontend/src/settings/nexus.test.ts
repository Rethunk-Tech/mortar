import { beforeEach, expect, test } from 'bun:test'
import { applyNexusAccount, getInitialState, useNexus } from './nexus.ts'

beforeEach(() => {
  useNexus.setState(getInitialState(), true)
})

test('a limits event with known counts replaces the empty meter copy', () => {
  useNexus.setState(
    applyNexusAccount({
      signedIn: true,
      name: 'Ada',
      premium: true,
      limits: {
        known: true,
        daily: { remaining: 40, limit: 2500, reset: '2026-10-01T00:00:00Z' },
        hourly: { remaining: 20, limit: 500, reset: '2026-09-30T19:00:00Z' },
      },
    }),
    true,
  )
  expect(useNexus.getState().limits.known).toBe(true)
  expect(useNexus.getState().limits.daily.remaining).toBe(40)
})

test('a later fetch without counts does not wipe known limits', () => {
  useNexus.setState(
    applyNexusAccount({
      signedIn: true,
      name: 'Ada',
      premium: false,
      limits: {
        known: true,
        daily: { remaining: 9, limit: 2500, reset: '' },
        hourly: { remaining: 9, limit: 500, reset: '' },
      },
    }),
    true,
  )
  useNexus.setState(
    applyNexusAccount(
      {
        signedIn: true,
        name: 'Ada',
        premium: false,
        limits: getInitialState().limits,
      },
      useNexus.getState().limits,
    ),
    true,
  )
  expect(useNexus.getState().limits.known).toBe(true)
  expect(useNexus.getState().limits.daily.remaining).toBe(9)
})
