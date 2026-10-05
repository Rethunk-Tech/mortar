import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { beginWork } from './usePending.ts'

describe('usePending', () => {
  test('same-tick double run is a no-op', () => {
    const lock = { current: false }
    expect(beginWork(lock)).toBe(true)
    expect(beginWork(lock)).toBe(false)
  })

  test('run passes errorTitle to reportError with a retry', () => {
    const src = readFileSync(join(import.meta.dir, 'usePending.ts'), 'utf8')
    expect(src).toContain('errorTitle?: string | undefined')
    expect(src).toContain('reportError(title, retry)')
  })
})
