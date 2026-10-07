import { expect, test } from 'bun:test'
import { winnerSentence } from './describe.ts'

test('a winner is named once, whether the checker or the user chose it', () => {
  expect(winnerSentence('NPC Mr Ginger', [])).toBe(' NPC Mr Ginger wins.')
  expect(winnerSentence('NPC Mr Ginger wins', [])).toBe(' NPC Mr Ginger wins.')
  expect(winnerSentence('[CP] MayorMod wins', ['A'])).toBe(' [CP] MayorMod wins; A overridden.')
})

test('an unclear, per-entry or absent winner has its own sentence', () => {
  expect(winnerSentence('unclear', [])).toBe(' The winner is unclear.')
  expect(winnerSentence('decided per entry', [])).toBe(' You chose which mod wins each clash.')
  expect(winnerSentence('', [])).toBe('')
  expect(winnerSentence(null, [])).toBe('')
})
