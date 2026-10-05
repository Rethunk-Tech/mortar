import { expect, mock, test } from 'bun:test'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { HistoryEvent } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

const added: unknown[][] = []

const queueService = '../../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
const profileService =
  '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
const realQueue = await import(queueService)
const realProfile = await import(profileService)

mock.module(queueService, () => ({
  ...realQueue,
  Add: async (reqs: unknown[]) => {
    added.push(reqs)
    return []
  },
}))
mock.module(profileService, () => ({ ...realProfile, Baseline: async () => 'before' }))

const { useProfiles } = await import('../../profiles/store.ts')
const { updateAll, sameSourceUpdates } = await import('./updateAll.ts')
const { updateWant } = await import('./wants.ts')
const { changesAfterBatch } = await import('./undoAll.ts')

const update = (key: string, extra: Partial<Update> = {}) =>
  ({ key, id: key, name: key, version: '2', githubRepo: `o/${key}`, ...extra }) as Update

test('a batch of mixed sources is one Add holding only the same-source updates', async () => {
  useProfiles.setState({ game: { id: 'stardew' } as never, openId: 'p1' })
  const list = [update('a'), update('b', { switch: true }), update('c')]
  const wants = sameSourceUpdates(list).map(updateWant)
  expect(await updateAll('stardew', 'p1', wants, 1)).toBe(true)
  expect(added).toHaveLength(1)
  const [reqs] = added as { repo: string; batchId: string }[][]
  expect(reqs?.map((r) => r.repo)).toEqual(['o/a', 'o/c'])
  expect(new Set(reqs?.map((r) => r.batchId)).size).toBe(1)
})

test('changes after the batch are the events past its own bulk event', () => {
  const ev = (id: string, kind: string) => ({ id, kind, label: id }) as HistoryEvent
  const newestFirst = [ev('pinned', 'pinned'), ev('batch', 'bulk'), ev('before', 'restored')]
  expect(changesAfterBatch(newestFirst, 'before').map((e) => e.id)).toEqual(['pinned'])
  expect(changesAfterBatch(newestFirst.slice(1), 'before')).toEqual([])
})
