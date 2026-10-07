import { expect, test } from 'bun:test'
import { redundantRowCount } from './sameJobGroups.ts'

const item = (kind: string, key: string, by: string[], covered = false) => ({
  kind,
  key,
  id: key,
  name: key,
  by: by.map((k) => ({ key: k, name: k })),
  covered,
})

test('the Redundant tab count is one row per group of mods doing the same job and one per other item', () => {
  expect(redundantRowCount([])).toBe(0)
  expect(
    redundantRowCount([
      item('sameJob', 'a', ['b']),
      item('sameJob', 'b', ['c']),
      item('superseded', 'd', ['e']),
    ]),
  ).toBe(2)
})
