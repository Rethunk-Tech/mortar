import { expect, test } from 'bun:test'
import { userModCount } from './count.ts'

test('bundled SMAPI mods are not counted', () => {
  const entry = (kind: string, n: number) => ({
    source: { kind },
    mods: Array.from({ length: n }, () => ({})),
  })
  expect(userModCount({ entries: [entry('smapi', 2), entry('nexus', 3)] })).toBe(3)
})
