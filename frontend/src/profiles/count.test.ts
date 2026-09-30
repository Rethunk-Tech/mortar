import { expect, test } from 'bun:test'
import { userModCount, userModEntries } from './count.ts'

test('bundled SMAPI mods are not counted', () => {
  const entry = (kind: string, n: number) => ({
    source: { kind },
    mods: Array.from({ length: n }, () => ({})),
  })
  expect(
    userModCount({ entries: [entry('smapi', 2), entry('mortar', 1), entry('nexus', 3)] }),
  ).toBe(3)
})

test('user mod entries omit SMAPI and Mortar bridge sources', () => {
  const entries = [
    { source: { kind: 'smapi' }, key: 'smapi-1' },
    { source: { kind: 'mortar' }, key: 'bridge-1' },
    { source: { kind: 'nexus' }, key: 'nexus-1' },
  ]
  expect(userModEntries(entries).map((e) => e.key)).toEqual(['nexus-1'])
})
