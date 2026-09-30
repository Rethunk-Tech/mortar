import { describe, expect, test } from 'bun:test'
import { beginWork } from './usePending.ts'

describe('usePending', () => {
  test('same-tick double run is a no-op', () => {
    const lock = { current: false }
    expect(beginWork(lock)).toBe(true)
    expect(beginWork(lock)).toBe(false)
  })
})
