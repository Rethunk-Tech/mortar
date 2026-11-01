import { expect, test } from 'bun:test'
import { identifierAt } from './symbolAt.ts'

test('identifierAt finds the name a click lands in, and nothing on punctuation', () => {
  const line = 'public Camera gameplayCamera;'
  expect(identifierAt(line, 16)).toBe('gameplayCamera')
  // Just past the last letter, on the right half of the 'a'.
  expect(identifierAt(line, 28)).toBe('gameplayCamera')
  expect(identifierAt(line, 8)).toBe('Camera')
  expect(identifierAt('a + b', 2)).toBe('')
})
