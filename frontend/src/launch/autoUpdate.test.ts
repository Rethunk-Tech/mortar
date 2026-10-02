import { expect, test } from 'bun:test'
import type { Update } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import {
  acknowledgeUpdateCaution,
  cautionAcknowledged,
  planAutoUpdates,
  queueItemSucceeded,
} from './autoUpdate.ts'

const update = (overrides: Partial<Update>): Update => ({
  key: 'old',
  uniqueId: 'mod',
  name: 'Mod',
  installed: '1.0.0',
  version: '2.0.0',
  url: '',
  nexusId: 42,
  githubRepo: '',
  unofficial: false,
  ...overrides,
})

test('plans installable updates except pinned entries', () => {
  const plan = planAutoUpdates(
    [
      update({ key: 'nexus' }),
      update({ key: 'github', githubRepo: 'owner/repo', nexusId: 0 }),
      update({ key: 'pinned' }),
      update({ key: 'unofficial', unofficial: true }),
    ],
    new Set(['pinned']),
  )

  expect(plan.updates.map((item) => item.key)).toEqual(['nexus', 'github'])
  expect(plan.wants).toEqual([
    {
      kind: 'update',
      modId: 42,
      name: 'Mod',
      version: '2.0.0',
      currentKey: 'nexus',
    },
    {
      kind: 'update',
      repo: 'owner/repo',
      name: 'Mod',
      version: '2.0.0',
      currentKey: 'github',
    },
  ])
})

test('treats queue items that disappeared as done and free waiting clicks as successful', () => {
  expect(queueItemSucceeded(undefined, true)).toBe(true)
  expect(queueItemSucceeded({ state: 'waiting-click' } as Item, false)).toBe(true)
  expect(queueItemSucceeded({ state: 'waiting-click' } as Item, true)).toBe(false)
})

test('keeps caution acknowledgements for auto-update', () => {
  const candidate = update({ key: 'cautious' })
  acknowledgeUpdateCaution('profile', candidate, false)
  expect(cautionAcknowledged('profile', candidate)).toBe(false)
  acknowledgeUpdateCaution('profile', candidate, true)
  expect(cautionAcknowledged('profile', candidate)).toBe(true)
  acknowledgeUpdateCaution('profile', candidate, false)
})
