import { expect, mock, test } from 'bun:test'

let calls = 0
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts', () => ({
  LastRunStarted: () => {
    calls += 1
    return Promise.resolve('2026-01-02T03:04:05Z')
  },
}))

test('Home callers share one last-run read per profile stamp', async () => {
  const { lastRunStarted } = await import('./useLastRun.ts')
  const all = await Promise.all([
    lastRunStarted('stardew', 'p1', '1'),
    lastRunStarted('stardew', 'p1', '1'),
    lastRunStarted('stardew', 'p1', '1'),
  ])
  expect(all).toEqual(['2026-01-02T03:04:05Z', '2026-01-02T03:04:05Z', '2026-01-02T03:04:05Z'])
  expect(calls).toBe(1)
  await lastRunStarted('stardew', 'p1', '2')
  expect(calls).toBe(2)
})
