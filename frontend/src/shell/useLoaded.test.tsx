import { afterAll, expect, test } from 'bun:test'

const testing = await import('@testing-library/react')
const { useLoaded } = await import('./useLoaded.ts')

afterAll(() => {
  testing.cleanup()
})

// Flushes pending promises inside act, so the state update they make renders before the test reads it.
const settle = () => testing.act(async () => new Promise<void>((resolve) => setTimeout(resolve, 0)))

function deferred<T>() {
  let resolve: (v: T) => void = () => undefined
  let reject: (e: unknown) => void = () => undefined
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

test('loads, then reloads with the same deps', async () => {
  let n = 0
  const next = () => {
    n += 1
    return Promise.resolve(n)
  }
  const { result } = testing.renderHook(() => useLoaded(next, [], 0))
  expect(result.current.loading).toBe(true)
  await settle()
  expect(result.current.data).toBe(1)
  expect(result.current.loading).toBe(false)
  testing.act(() => result.current.reload())
  await settle()
  expect(result.current.data).toBe(2)
})

test('a run superseded by new deps is dropped', async () => {
  const first = deferred<string>()
  const second = deferred<string>()
  const { result, rerender } = testing.renderHook(
    ({ key }) => useLoaded(() => (key === 'a' ? first.promise : second.promise), [key], ''),
    { initialProps: { key: 'a' } },
  )
  rerender({ key: 'b' })
  second.resolve('b')
  await settle()
  expect(result.current.data).toBe('b')
  first.resolve('a')
  await testing.act(() => first.promise)
  expect(result.current.data).toBe('b')
})

test('an answer after unmount is ignored and errors reach onError once', async () => {
  const late = deferred<string>()
  const seen: unknown[] = []
  const gone = testing.renderHook(() =>
    useLoaded(
      () => late.promise,
      [],
      '',
      (e) => seen.push(e),
    ),
  )
  gone.unmount()
  late.reject(new Error('late'))
  await late.promise.catch(() => undefined)
  expect(seen).toEqual([])

  const boom = new Error('boom')
  const { result } = testing.renderHook(() =>
    useLoaded(
      () => Promise.reject(boom),
      [],
      'fallback',
      (e) => seen.push(e),
    ),
  )
  await settle()
  expect(result.current.error).toBe(boom)
  expect(result.current.data).toBe('fallback')
  expect(result.current.loading).toBe(false)
  expect(seen).toEqual([boom])
})

test('new deps clear the old answer until the new one arrives', async () => {
  const second = deferred<string>()
  const { result, rerender } = testing.renderHook(
    ({ key }) => useLoaded(() => (key === 'a' ? Promise.resolve('a') : second.promise), [key], ''),
    { initialProps: { key: 'a' } },
  )
  await settle()
  expect(result.current.data).toBe('a')
  rerender({ key: 'b' })
  await settle()
  expect(result.current.data).toBe('')
  second.resolve('b')
  await settle()
  expect(result.current.data).toBe('b')
})

test('a null loader never runs', () => {
  const { result } = testing.renderHook(() => useLoaded<number>(null, [], 7))
  expect(result.current).toMatchObject({ data: 7, loading: false })
})

test('a failed reload keeps the last good data', async () => {
  let fail = false
  const { result } = testing.renderHook(() =>
    useLoaded(
      () => (fail ? Promise.reject(new Error('offline')) : Promise.resolve('good')),
      [],
      '',
    ),
  )
  await settle()
  expect(result.current.data).toBe('good')
  fail = true
  testing.act(() => result.current.reload())
  await settle()
  expect(result.current.data).toBe('good')
  expect(result.current.error).toBeInstanceOf(Error)
})
