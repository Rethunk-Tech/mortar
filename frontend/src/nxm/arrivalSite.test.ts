import { expect, test } from 'bun:test'
import type { Arrival } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nxmsvc/models.ts'
import { arrivalSite } from './store.ts'

test('an arrival names its site from the source registry, not a typed string', () => {
  const at = (a: Partial<Arrival>) => arrivalSite(a as Arrival)
  expect(at({ source: 'vsmoddb' })).toBe('VS Mod DB')
  expect(at({ source: 'curseforge' })).toBe('CurseForge')
  expect(at({ package: 'Author-Mod' })).toBe('Thunderstore')
  expect(at({})).toBe('Nexus')
})
