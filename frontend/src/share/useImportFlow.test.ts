import { describe, expect, test } from 'bun:test'
import { shouldOpenQueueAfterImport } from './useImportFlow.ts'

describe('shouldOpenQueueAfterImport', () => {
  test('opens the queue sheet only when queued is greater than zero', () => {
    expect(shouldOpenQueueAfterImport(0)).toBe(false)
    expect(shouldOpenQueueAfterImport(1)).toBe(true)
    expect(shouldOpenQueueAfterImport(3)).toBe(true)
  })
})
