import { describe, expect, test } from 'bun:test'
import { beginWork, failureSink } from './usePending.ts'

describe('usePending', () => {
  test('same-tick double run is a no-op', () => {
    const lock = { current: false }
    expect(beginWork(lock)).toBe(true)
    expect(beginWork(lock)).toBe(false)
  })

  test('quietExpected leaves network and locked failures unreported, and only those', () => {
    const seen: unknown[] = []
    const quiet = failureSink((e) => seen.push(e), true)
    for (const e of ['[network] offline', '[locked] keyring', '[invalid] bad']) {
      quiet(new Error(e))
    }
    failureSink((e) => seen.push(e))(new Error('[network] offline'))
    expect(seen.map(String)).toEqual(['Error: [invalid] bad', 'Error: [network] offline'])
  })
})
