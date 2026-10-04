import { describe, expect, test } from 'bun:test'
import type { EverywherePreview } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { mergePreviews } from './mergePreviews.ts'

describe('mergePreviews', () => {
  test('keeps skipped profiles out of affected', () => {
    const a: EverywherePreview = {
      affected: [{ profileId: 'ok', name: 'Ok', oldKey: 'a-1' }],
      skipped: [{ profileId: 'pin', name: 'Pin', reason: 'pinned' }],
    }
    const b: EverywherePreview = {
      affected: [{ profileId: 'ok', name: 'Ok', oldKey: 'a-1' }],
      skipped: [{ profileId: 'lock', name: 'Lock', reason: 'locked' }],
    }
    const got = mergePreviews([a, b])
    expect(got.affected?.map((r) => r.profileId)).toEqual(['ok'])
    expect(got.skipped?.map((r) => r.reason).sort()).toEqual(['locked', 'pinned'])
  })
})
