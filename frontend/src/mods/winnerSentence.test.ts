import { expect, test } from 'bun:test'
import { winnerSentence } from './describe.ts'

test('a named winner reads the same whether the checker ranked it or the user chose it', () => {
  expect(winnerSentence('top', 'NPC Mr Ginger', [])).toBe(' NPC Mr Ginger wins.')
  expect(winnerSentence('chosen', 'NPC Mr Ginger', [])).toBe(' NPC Mr Ginger wins.')
  expect(winnerSentence('top', '[CP] MayorMod', ['A'])).toBe(' [CP] MayorMod wins; A overridden.')
})

test('a load-order winner says so', () => {
  expect(winnerSentence('load-order', 'Addon', [])).toBe(' Addon wins by load order.')
  expect(winnerSentence('load-order', 'Addon', ['Base'])).toBe(
    ' Addon wins by load order; Base overridden.',
  )
})

test('winners that are not one named mod have their own sentence', () => {
  expect(winnerSentence('neither', '', [])).toBe(' Content Patcher applies neither.')
  expect(winnerSentence('unclear', '', [])).toBe(' The winner is unclear.')
  expect(winnerSentence('per-entry', '', [])).toBe(' You chose which mod wins each clash.')
  expect(winnerSentence('', '', [])).toBe('')
  expect(winnerSentence(null, null, [])).toBe('')
})
