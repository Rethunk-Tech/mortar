import { expect, test } from 'bun:test'
import type { UpdatesResult } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Profile } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { keptUpdates } from './kept.ts'

const update = (key: string) => ({ key, id: key, name: key, version: '2', installed: '1' })

test('only updates of pinned entries are kept back', () => {
  const profile = {
    entries: [{ key: 'a', pinned: true }, { key: 'b' }],
  } as unknown as Profile
  const result = { updates: [update('a'), update('b'), update('a')] } as unknown as UpdatesResult
  expect(keptUpdates(result, profile).map((u) => u.key)).toEqual(['a'])
  expect(keptUpdates(null, profile)).toEqual([])
})
