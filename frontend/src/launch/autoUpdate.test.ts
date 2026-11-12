import { expect, test } from 'bun:test'
import type { Update } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type {
  Item,
  Request,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import {
  acknowledgeUpdateCaution,
  cautionAcknowledged,
  planAutoUpdates,
  queueItemSucceeded,
} from './autoUpdate.ts'

// The queue request comes from Go with the update; this stands in for it.
const update = (overrides: Partial<Update>): Update => {
  const u = base(overrides)
  const source = u.githubRepo ? { repo: u.githubRepo } : { modId: u.nexusId }
  return {
    ...u,
    want: {
      kind: 'update',
      ...source,
      name: u.name,
      version: u.version,
      currentKey: u.key,
    } as unknown as Request,
  }
}

const base = (overrides: Partial<Update>): Update => ({
  key: 'old',
  id: 'mod',
  name: 'Mod',
  installed: '1.0.0',
  version: '2.0.0',
  url: '',
  nexusId: 42,
  githubRepo: '',
  source: '',
  unofficial: false,
  want: { kind: 'update' } as Request,
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
