import { expect, test } from 'bun:test'
import { checkedWhen } from './updates.ts'

const MINUTE = 60_000

test('checkedWhen uses the check timestamp when one exists', () => {
  expect(checkedWhen(null, 1)).toBeNull()
  expect(checkedWhen(1, 1)).toEqual({ unit: 'now', n: 0 })
  expect(checkedWhen(1, 1 + 5 * MINUTE)).toEqual({ unit: 'minute', n: 5 })
})
