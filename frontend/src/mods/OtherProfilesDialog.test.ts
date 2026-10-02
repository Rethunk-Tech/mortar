import { describe, expect, test } from 'bun:test'
import { selectableProfileIds } from './otherProfiles.ts'

describe('selectableProfileIds', () => {
  test('does not select locked or pinned profiles', () => {
    expect(selectableProfileIds(['a', 'b', 'c'], new Set(['b']))).toEqual(['a', 'c'])
  })
})
