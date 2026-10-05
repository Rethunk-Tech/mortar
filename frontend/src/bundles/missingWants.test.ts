import { expect, test } from 'bun:test'
import type { Source } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { bundleWants } from './missingWants.ts'

test('one want per archive; mods with no downloadable source are skipped', () => {
  const mod = (name: string, entryKey: string, source: Source) => ({
    id: name,
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
  expect(bundleWants(template.bundle).map((w) => w.name)).toEqual(['A', 'B', 'D'])
})
