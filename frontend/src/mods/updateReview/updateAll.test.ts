import { afterEach, expect, mock, spyOn, test } from 'bun:test'
import { CancellablePromise } from '@wailsio/runtime'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { HistoryEvent } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import * as profileService from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import * as queueService from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { useProfiles } from '../../profiles/store.ts'
import { useQueue } from '../../queue/store.ts'
import { useToasts } from '../../toasts/store.ts'
import { changesAfterBatch } from './undoAll.ts'
import { listedBySource, needChoiceUpdates, sameSourceUpdates, updateAll } from './updateAll.ts'
import { updateWant } from './wants.ts'

afterEach(() => {
  mock.restore()
  useToasts.setState(useToasts.getInitialState(), true)
})

const update = (key: string, extra: Partial<Update> = {}) =>
  ({ key, id: key, name: key, version: '2', githubRepo: `o/${key}`, ...extra }) as Update

test('a batch of mixed sources is one Add holding only the same-source updates', async () => {
  useProfiles.setState({ game: { id: 'stardew' } as never, openId: 'p1' })
  const add = spyOn(queueService, 'Add').mockImplementation(() => CancellablePromise.resolve([]))
  spyOn(profileService, 'BeginUpdateBatch').mockImplementation(() =>
    CancellablePromise.resolve({ before: 'before', batch: 'batch-1' }),
  )
  const list = [
    update('a'),
    update('b', { switch: true }),
    update('c'),
    update('d', { githubRepo: '', nexusId: 22_743, pickFile: true }),
  ]
  const wants = sameSourceUpdates(list).map(updateWant)
  expect(await updateAll('stardew', 'p1', wants, 1)).toBe(true)
  expect(add).toHaveBeenCalledTimes(1)
  const reqs = add.mock.calls[0]?.[0]
  expect(reqs?.map((r) => r.repo)).toEqual(['o/a', 'o/c'])
  expect(new Set(reqs?.map((r) => r.batchId))).toEqual(new Set(['batch-1']))
  // The empty batch settles on the next queue change; let it here so its summary toast stays in this file.
  useQueue.setState({})
  expect(useToasts.getState().toasts.map((t) => t.title)).toEqual([
    'Updated 0 mods; 1 need your choice',
  ])
})

test('an update from another site that Mortar cannot download is still listed', () => {
  const list = [
    update('a'),
    update('b', { switch: true }),
    update('c', { switch: true, notDistributable: true }),
  ]
  expect(needChoiceUpdates(list).map((u) => u.key)).toEqual(['b'])
  expect(listedBySource(list).map((u) => u.key)).toEqual(['a', 'c'])
})

test('changes after the batch are the events past its own bulk event', () => {
  const ev = (id: string, kind: string) => ({ id, kind }) as HistoryEvent
  const newestFirst = [ev('pinned', 'pinned'), ev('batch', 'bulk'), ev('before', 'restored')]
  expect(changesAfterBatch(newestFirst, 'before').map((e) => e.id)).toEqual(['pinned'])
  expect(changesAfterBatch(newestFirst.slice(1), 'before')).toEqual([])
})
