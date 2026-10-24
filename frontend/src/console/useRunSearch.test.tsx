import { afterAll, afterEach, beforeEach, expect, jest, test } from 'bun:test'
import type { RunSearch } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'

const testing = await import('@testing-library/react')
const { useRunSearch } = await import('./useRunSearch.ts')

afterAll(() => {
  testing.cleanup()
})
beforeEach(() => {
  jest.useFakeTimers()
})
afterEach(() => {
  jest.useRealTimers()
})

// Runs the debounce out, then lets the answers it started settle.
const wait = (ms: number) =>
  testing.act(async () => {
    jest.advanceTimersByTime(ms)
    await Promise.resolve()
  })
const answer = (line: string): RunSearch => ({
  hits: [{ runId: 'r', lineNumber: 1, line, started: '', outcome: '' }],
  truncated: false,
})

// The pure loader test covers the guard itself; this one covers useDebouncedLoad's timer and React state wiring.
test('a slow answer for an older query does not replace the newer one', async () => {
  const slow = Promise.withResolvers<RunSearch>()
  const search = (text: string) => (text === 'ab' ? Promise.resolve(answer('ab')) : slow.promise)
  const { result, rerender } = testing.renderHook(({ q }) => useRunSearch(true, q, search), {
    initialProps: { q: 'a' },
  })
  await wait(300)
  rerender({ q: 'ab' })
  await wait(300)
  expect(result.current.result.hits?.[0]?.line).toBe('ab')
  slow.resolve(answer('a'))
  await wait(0)
  expect(result.current.result.hits?.[0]?.line).toBe('ab')
  expect(result.current.loading).toBe(false)
})
