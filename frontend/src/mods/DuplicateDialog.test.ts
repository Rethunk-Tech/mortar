import { expect, test } from 'bun:test'
import type { Copy } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { nexusKeepKey } from './lookup.ts'

const copy = (key: string, nexus: boolean): Copy => ({
  key,
  name: 'S',
  version: '1',
  source: 'local',
  nexus,
  newest: true,
  needed: [],
  tooOld: [],
})

test('Keep the Nexus copy is offered when exactly one copy is from Nexus', () => {
  expect(nexusKeepKey([copy('archive', false), copy('nexus', true)])).toBe('nexus')
})

test('Keep the Nexus copy is not offered when none or several copies are from Nexus', () => {
  expect(nexusKeepKey([copy('a', false), copy('b', false)])).toBeNull()
  expect(nexusKeepKey([copy('a', true), copy('b', true)])).toBeNull()
})
