import { expect, test } from 'bun:test'
import { PROFILE_CARD_LIMIT, profileCardsSlice } from './profileCardsSlice.ts'

test('profile card grid shows at most six profiles and counts the rest', () => {
  const ids = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h']
  const { visible, more } = profileCardsSlice(ids)
  expect(visible).toHaveLength(PROFILE_CARD_LIMIT)
  expect(visible).toEqual(['a', 'b', 'c', 'd', 'e', 'f'])
  expect(more).toBe(2)
})

test('profile card grid omits “more” when every profile fits', () => {
  const { visible, more } = profileCardsSlice(['only'])
  expect(visible).toEqual(['only'])
  expect(more).toBe(0)
})
