import { expect, test } from 'bun:test'
import { loader } from './loader.ts'

const flush = () => new Promise<void>((resolve) => setTimeout(resolve, 0))

function harness(keepAcrossKeys: boolean) {
  const log: string[] = []
  const pending: (() => void)[] = []
  const runs = loader<string>(
    {
      value: (v) => log.push(`value ${v}`),
      error: (e) => log.push(`error ${String(e)}`),
      loading: (on) => log.push(on ? 'loading' : 'idle'),
      reset: () => log.push('reset'),
    },
    {
      keepAcrossKeys,
      // A deferred start waits in `pending` until the test runs it; cancelling empties its slot.
      defer: (go) => {
        const i = pending.push(go) - 1
        return () => {
          pending[i] = () => undefined
        }
      },
    },
  )
  return { log, pending, runs }
}

// The branches useLoaded and useDebouncedLoad share: a superseded answer and error are dropped, new deps reset or keep
// the value, a null load resets, a start cancelled by new deps never runs, and stop drops what is in flight.
test('only the newest load lands', async () => {
  for (const keep of [false, true]) {
    const { log, pending, runs } = harness(keep)
    const a = Promise.withResolvers<string>()
    const b = Promise.withResolvers<string>()
    runs.deps(['a'], () => a.promise)
    runs.deps(['a'], () => b.promise)
    expect(pending).toHaveLength(1)
    pending[0]?.()
    runs.deps(['b'], () => b.promise)
    pending[1]?.()
    b.resolve('b')
    a.reject(new Error('a failed'))
    await flush()
    expect(log).toEqual([
      'loading',
      'loading',
      ...(keep ? [] : ['reset']),
      'loading',
      'loading',
      'value b',
      'idle',
    ])

    log.length = 0
    const c = Promise.withResolvers<string>()
    const older = runs.run(() => c.promise)
    const newest = runs.run(() => Promise.reject(new Error('newest')))
    c.resolve('c')
    await Promise.all([older, newest])
    expect(log).toEqual(['loading', 'loading', 'error Error: newest', 'idle'])

    log.length = 0
    runs.deps(['c'], () => c.promise)
    runs.deps(['d'], null)
    pending.at(-1)?.()
    await flush()
    expect(log).toEqual([...(keep ? [] : ['reset']), 'loading', 'reset', 'idle'])

    log.length = 0
    await runs.run(null)
    const late = runs.run(() => Promise.resolve('late'))
    runs.stop()
    await late
    expect(log).toEqual(['reset', 'idle', 'loading'])
  }
})
