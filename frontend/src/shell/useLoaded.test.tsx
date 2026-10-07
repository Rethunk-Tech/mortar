import { afterAll, expect, test } from 'bun:test'
import { GlobalRegistrator } from '@happy-dom/global-registrator'

GlobalRegistrator.register({ url: 'http://localhost/' })
const { act, cleanup, renderHook, waitFor } = await import('@testing-library/react')
const { useLoaded } = await import('./useLoaded.ts')

afterAll(async () => {
  cleanup()
  await GlobalRegistrator.unregister()
})

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
  const { result } = renderHook(() => useLoaded(next, [], 0))
  expect(result.current.loading).toBe(true)
  await waitFor(() => expect(result.current.data).toBe(1))
  expect(result.current.loading).toBe(false)
  act(() => result.current.reload())
  await waitFor(() => expect(result.current.data).toBe(2))
})

test('a run superseded by new deps is dropped', async () => {
  const first = deferred<string>()
  const second = deferred<string>()
  const { result, rerender } = renderHook(
    ({ key }) => useLoaded(() => (key === 'a' ? first.promise : second.promise), [key], ''),
    { initialProps: { key: 'a' } },
  )
  rerender({ key: 'b' })
  second.resolve('b')
  await waitFor(() => expect(result.current.data).toBe('b'))
  first.resolve('a')
  await act(() => first.promise)
  expect(result.current.data).toBe('b')
})

test('an answer after unmount is ignored and errors reach onError once', async () => {
  const late = deferred<string>()
  const seen: unknown[] = []
  const gone = renderHook(() =>
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
  const { result } = renderHook(() =>
    useLoaded(
      () => Promise.reject(boom),
      [],
      'fallback',
      (e) => seen.push(e),
    ),
  )
  await waitFor(() => expect(result.current.error).toBe(boom))
  expect(result.current.data).toBe('fallback')
  expect(result.current.loading).toBe(false)
  expect(seen).toEqual([boom])
})

test('a null loader never runs', () => {
  const { result } = renderHook(() => useLoaded<number>(null, [], 7))
  expect(result.current).toMatchObject({ data: 7, loading: false })
})
