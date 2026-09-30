import { expect, test } from 'bun:test'
import type { Copy } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { preselect } from './lookup.ts'

const copy = (key: string, newest: boolean, nexus = false): Copy => ({
  key,
  name: 'S',
  version: '1',
  source: 'local',
  nexus,
  newest,
  needed: [],
  tooOld: [],
})

test('preselect keeps the newest copy', () => {
  expect(preselect([copy('old', false), copy('new', true)])).toBe('new')
})

test('preselect prefers the Nexus copy among equals', () => {
  expect(preselect([copy('a', true), copy('b', true, true)])).toBe('b')
})

test('a newer archive beats an older Nexus copy', () => {
  expect(preselect([copy('nexus', false, true), copy('archive', true)])).toBe('archive')
})

test('preselect falls back to Nexus, then the first, when versions are unknown', () => {
  expect(preselect([copy('a', false), copy('b', false, true)])).toBe('b')
  expect(preselect([copy('a', false), copy('b', false)])).toBe('a')
  expect(preselect([])).toBe('')
})
