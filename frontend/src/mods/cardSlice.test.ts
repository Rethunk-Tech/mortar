import { expect, test } from 'bun:test'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { cardSlice, ownSlice } from './overlayRows.ts'

const profile = (entry: Record<string, unknown>) =>
  ({ id: 'p', entries: [{ key: 'k', source: { kind: 'local' }, ...entry }] }) as unknown as Profile

test('a card ignores a pin but not its tags, while a row slice sees both', () => {
  const plain = profile({})
  const pinned = profile({ pinned: true, pinReason: 'Frozen with the profile' })
  expect(cardSlice(pinned, 'k')).toBe(cardSlice(plain, 'k'))
  expect(ownSlice(pinned, 'k')).not.toBe(ownSlice(plain, 'k'))
  expect(cardSlice(profile({ tags: ['a'] }), 'k')).not.toBe(cardSlice(plain, 'k'))
})
