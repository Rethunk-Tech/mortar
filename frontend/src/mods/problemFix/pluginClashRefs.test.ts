import { expect, test } from 'bun:test'
import type { PluginCopy } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { losingRefs } from './pluginClashRefs.ts'

test('keep newer disables every component of each losing package, not just its first', () => {
  const copies = [
    { key: 'ts-a', id: 'a1' },
    { key: 'ts-b', id: 'b1' },
  ] as PluginCopy[]
  const mods = [
    { key: 'ts-a', id: 'a1' },
    { key: 'ts-b', id: 'b1' },
    { key: 'ts-b', id: 'b2' },
  ]
  expect(losingRefs(copies, 'ts-a', mods)).toEqual([
    { key: 'ts-b', id: 'b1' },
    { key: 'ts-b', id: 'b2' },
  ])
  expect(losingRefs(copies, 'ts-b', [])).toEqual([{ key: 'ts-a', id: 'a1' }])
})
