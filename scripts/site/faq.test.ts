import { expect, test } from 'bun:test'
import { gamesWinner } from './faq.ts'

const row = (theirs: string) => `| Games | 292 | ${theirs} |`

test('an "Over N" games count is Mortar\'s only by a clear margin', () => {
  expect(gamesWinner(row('Over 250 (README)'), 292)).toBe(
    '| Games | !win 292 | Over 250 (README) |',
  )
  expect(gamesWinner(row('Over 280'), 292)).toBe(row('Over 280'))
  expect(gamesWinner(row('307 Thunderstore games'), 292)).toBe(
    '| Games | 292 | !win 307 Thunderstore games |',
  )
})
