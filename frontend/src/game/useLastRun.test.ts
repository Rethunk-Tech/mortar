import { afterEach, expect, mock, spyOn, test } from 'bun:test'
import * as launchsvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { lastRunStarted } from './useLastRun.ts'

afterEach(() => {
  mock.restore()
})

test('Home callers share one last-run read per profile stamp', async () => {
  const read = spyOn(launchsvc, 'LastRunStarted').mockResolvedValue('2026-01-02T03:04:05Z')
  const all = await Promise.all([
    lastRunStarted('stardew', 'p1', '1'),
    lastRunStarted('stardew', 'p1', '1'),
    lastRunStarted('stardew', 'p1', '1'),
  ])
  expect(all).toEqual(['2026-01-02T03:04:05Z', '2026-01-02T03:04:05Z', '2026-01-02T03:04:05Z'])
  expect(read).toHaveBeenCalledTimes(1)
  await lastRunStarted('stardew', 'p1', '2')
  expect(read).toHaveBeenCalledTimes(2)
})
