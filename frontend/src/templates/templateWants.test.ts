import { expect, test } from 'bun:test'
import type { Source } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { templateWants } from './templateWants.ts'

test('one want per missing archive; local and present mods are skipped', () => {
  const mod = (name: string, entryKey: string, source: Source) => ({
    uniqueId: name,
    name,
    entryKey,
    source,
  })
  const template = {
    bundle: [
      mod('A', 'nexus-1-2', { kind: 'nexus', name: 'a.zip', modId: 1, fileId: 2, version: '1.0' }),
      mod('A2', 'nexus-1-2', { kind: 'nexus', name: 'a.zip', modId: 1, fileId: 2, version: '1.0' }),
      mod('B', 'gh-b', { kind: 'github', name: 'b', repo: 'o/b', tag: 'v1', asset: 'b.zip' }),
      mod('C', 'local-c', { kind: 'local', name: 'c' }),
      mod('D', 'nexus-4-5', { kind: 'nexus', name: 'd.zip', modId: 4, fileId: 5 }),
    ],
  }
  expect(templateWants(template, ['A', 'A2', 'B', 'C']).map((w) => w.name)).toEqual(['A', 'o/b'])
})
