import { expect, test } from 'bun:test'
import { beginUsageLoad } from '../usageLoad.ts'

test('unmount stops UsageProgress ticks and ignores a late Usage result', async () => {
  const bytes: number[] = []
  const usages: string[] = []
  let tick: (() => void) | undefined
  let release: (u: {
    path: string
    profiles: null
    store: number
    cache: number
    backups: number
    trash: number
    total: number
  }) => void = () => undefined
  const usage = new Promise<Parameters<typeof release>[0]>((resolve) => {
    release = resolve
  })
  const stop = beginUsageLoad({
    usage: () => usage,
    progress: async () => ({ measuring: true, bytes: 9 }),
    setBytes: (n) => {
      bytes.push(n)
    },
    setUsage: (u) => {
      usages.push(u.path)
    },
    onError: () => undefined,
    every: (fn) => {
      tick = fn
      return 1 as unknown as ReturnType<typeof setInterval>
    },
    stopEvery: () => {
      tick = undefined
    },
  })
  tick?.()
  await Promise.resolve()
  expect(bytes).toEqual([9])
  stop()
  tick?.()
  await Promise.resolve()
  release({
    path: '/late',
    profiles: null,
    store: 0,
    cache: 0,
    backups: 0,
    trash: 0,
    total: 0,
  })
  await usage
  expect(bytes).toEqual([9])
  expect(usages).toEqual([])
})
