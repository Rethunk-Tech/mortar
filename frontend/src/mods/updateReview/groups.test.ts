import { expect, test } from 'bun:test'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type {
  Mod,
  Profile,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { groupBySource, isNexusSource, reportsElsewhere, splitBySource } from './groups.ts'

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

test('both names Nexus updates come under read as Nexus', () => {
  // A SMAPI channel update names its site "Nexus"; a source-driver update names it "Nexus Mods".
  expect(isNexusSource('Nexus')).toBe(true)
  expect(isNexusSource('Nexus Mods')).toBe(true)
  expect(isNexusSource('GitHub')).toBe(false)
  const rows = ['Nexus Mods', 'Nexus'].map((source) => ({ source }) as Update)
  expect(groupBySource(rows).map((g) => g.name)).toEqual(['Nexus', 'Nexus Mods'])
})

test('a Nexus mod’s update is not reported elsewhere when Nexus itself reports it', () => {
  for (const source of ['Nexus', 'Nexus Mods']) {
    expect(reportsElsewhere('nexus', source)).toBe(false)
  }
  expect(reportsElsewhere('nexus', 'GitHub')).toBe(true)
  expect(reportsElsewhere('nexus', '')).toBe(false)
  expect(reportsElsewhere('github', 'GitHub')).toBe(false)
})

test('a download’s support parts move to Libraries and leave the source groups', () => {
  // Two mods of one download: the support part requires the main one, which the download is named after.
  const main: Mod = {
    key: 'pack',
    id: 'me.Main',
    name: 'Main',
    needs: [],
    contentPackFor: '',
  } as unknown as Mod
  const support: Mod = {
    key: 'pack',
    id: 'me.Support',
    name: 'Support',
    needs: ['me.Main'],
    contentPackFor: '',
  } as unknown as Mod
  const profile = { id: 'p', name: 'P', entries: [] } as unknown as Profile
  const rows = [
    { key: 'pack', id: 'me.Main', name: 'Main', source: 'GitHub' },
    { key: 'pack', id: 'me.Support', name: 'Support', source: 'GitHub' },
  ] as Update[]
  const { groups, libraries } = splitBySource(rows, [main, support], profile)
  expect(libraries.map((u) => u.id)).toEqual(['me.Support'])
  expect(groups.map((g) => `${g.name}:${g.list.map((u) => u.id).join(',')}`)).toEqual([
    'GitHub:me.Main',
  ])
})
