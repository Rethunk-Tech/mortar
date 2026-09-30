import { expect, test } from 'bun:test'
import { firstCoverSrc, hasPickedCover } from './cover.ts'

test('hasPickedCover uses the saved cover when nothing is staged', () => {
  expect(hasPickedCover('picked.png', undefined)).toBe(true)
  expect(hasPickedCover('', undefined)).toBe(false)
  expect(hasPickedCover(undefined, undefined)).toBe(false)
})

test('hasPickedCover follows a staged pick or clear', () => {
  expect(hasPickedCover('', '/tmp/cover.png')).toBe(true)
  expect(hasPickedCover('picked.png', null)).toBe(false)
})

test('firstCoverSrc skips failed urls', () => {
  expect(firstCoverSrc(['a', 'b'], ['a'])).toBe('b')
  expect(firstCoverSrc(['a'], ['a'])).toBeUndefined()
})
