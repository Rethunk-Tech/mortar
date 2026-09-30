import { describe, expect, test } from 'bun:test'
import { shouldRunStartupCheck } from './startupChecks.ts'

describe('shouldRunStartupCheck', () => {
  test('runs each check only when its settings toggle is on', () => {
    expect(shouldRunStartupCheck(true)).toBe(true)
    expect(shouldRunStartupCheck(undefined)).toBe(true)
    expect(shouldRunStartupCheck(null)).toBe(true)
    expect(shouldRunStartupCheck(false)).toBe(false)
  })
})
