import { expect, test } from 'bun:test'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { groupBySource } from './groups.ts'

test('groups sources in Nexus, GitHub, Thunderstore, then alphabetical order', () => {
  const rows = ['Modrinth', 'Thunderstore', 'GitHub', 'Nexus Mods', 'CurseForge', 'GitHub'].map(
    (source) => ({ source }) as Update,
  )
  expect(groupBySource(rows).map((g) => `${g.name}:${g.list.length}`)).toEqual([
    'Nexus Mods:1',
    'GitHub:2',
    'Thunderstore:1',
    'CurseForge:1',
    'Modrinth:1',
  ])
})
