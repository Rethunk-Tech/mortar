import { expect, test } from 'bun:test'
import type { Update } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { planAutoUpdates } from './autoUpdate.ts'

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
