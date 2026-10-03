import { describe, expect, test } from 'bun:test'

import { clampPage, DEBOUNCE_MS, debounceDue, nextPage, prevPage } from './browseState.ts'

describe('browse debounce', () => {
  test('waits 300 ms', () => {
    expect(debounceDue(DEBOUNCE_MS - 1)).toBe(false)
    expect(debounceDue(DEBOUNCE_MS)).toBe(true)
  })
})

describe('browse paging', () => {
  test('advances and clamps to the last page', () => {
    expect(nextPage({ page: 1, total: 27 })).toBe(2)
    expect(nextPage({ page: 2, total: 27 })).toBe(2)
    expect(prevPage({ page: 1, total: 27 })).toBe(1)
    expect(clampPage({ page: 9, total: 27 })).toBe(2)
  })
})
