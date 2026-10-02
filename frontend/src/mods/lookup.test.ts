import { expect, test } from 'bun:test'
import type { Copy } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import {
  type Drift,
  DriftKind,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  entryHasDrift,
  installableUpdate,
  listedAgainstNexus,
  modStatusProblem,
  nexusKeepKey,
  offersUpdate,
  preselect,
  problemCount,
  reshow,
  updateCount,
  updateFor,
} from './lookup.ts'

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

test('Keep the Nexus copy is the recommended key when exactly one copy is from Nexus', () => {
  expect(nexusKeepKey([copy('a', false), copy('b', true, true)])).toBe('b')
  expect(nexusKeepKey([copy('a', true, true), copy('b', true, true)])).toBeNull()
})

test('a newer archive beats an older Nexus copy', () => {
  expect(preselect([copy('nexus', false, true), copy('archive', true)])).toBe('archive')
})

test('preselect falls back to Nexus, then the first, when versions are unknown', () => {
  expect(preselect([copy('a', false), copy('b', false, true)])).toBe('b')
  expect(preselect([copy('a', false), copy('b', false)])).toBe('a')
  expect(preselect([])).toBe('')
})

test("drift on an entry flags that entry's mods for status grouping", () => {
  const mod = { key: 'k', uniqueId: 'me.a', name: 'A', enabled: true }
  const drift: Drift = { kind: DriftKind.DriftChanged, folder: 'k', key: 'k' }
  const result = {
    missing: [],
    duplicates: [],
    broken: [],
    assetConflicts: [],
    settings: [],
    runErrors: [],
    drift: [drift],
    unknown: false,
  }
  expect(
    modStatusProblem(
      result,
      mod as import('../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts').Mod,
    ),
  ).toBe(true)
  expect(entryHasDrift(result, 'k')).toBe(true)
  expect(
    entryHasDrift(
      { ...result, drift: [{ ...drift, kind: DriftKind.DriftUnknown, key: 'loose' }] },
      'k',
    ),
  ).toBe(false)
})

test('problems and updates are counted per finding', () => {
  expect(problemCount(null)).toBe(0)
  expect(
    problemCount({
      missing: [],
      duplicates: [{ uniqueId: 'me.a', name: 'A', copies: [] }],
      broken: [{ key: 'k', uniqueId: 'me.b', name: 'B', status: 'broken', brokeIn: '' }],
      assetConflicts: [],
      settings: [],
      runErrors: [
        {
          key: 'k2',
          uniqueId: 'me.c',
          name: 'C',
          count: 1,
          first: 'err',
          severe: false,
          runId: 'run-1',
        },
      ],
      drift: [{ kind: DriftKind.DriftChanged, folder: 'k', key: 'k' }],
      unknown: false,
    }),
  ).toBe(4)
  expect(updateCount(null)).toBe(0)
})

test('an update belongs to one copy of a mod', () => {
  const update = {
    key: 'a-1',
    uniqueId: 'Me.A',
    name: 'A',
    installed: '1',
    version: '2',
    url: '',
    nexusId: 0,
    githubRepo: '',
    unofficial: false,
  }
  const result = { updates: [update], unknown: false }
  expect(updateCount(result)).toBe(1)
  expect(updateFor(result, { key: 'a-1', uniqueId: 'me.a' })).toBe(update)
  expect(updateFor(result, { key: 'a-2', uniqueId: 'me.a' })).toBeUndefined()
})

test('a pin or skipped version hides that update', () => {
  expect(offersUpdate(undefined, '2.0.0')).toBe(true)
  expect(offersUpdate({ pinned: true }, '2.0.0')).toBe(false)
  expect(offersUpdate({ skipVersion: '2.0.0' }, '2.0.0')).toBe(false)
  expect(offersUpdate({ skipVersion: '2.0.0' }, '2.1.0')).toBe(true)
  const update = {
    key: 'a-1',
    uniqueId: 'me.a',
    name: 'A',
    installed: '1',
    version: '2.0.0',
    url: '',
    nexusId: 0,
    githubRepo: '',
    unofficial: false,
  }
  const result = { updates: [update], unknown: false }
  const pinned = {
    id: 'p',
    name: 'P',
    notes: '',
    cover: '',
    order: 0,
    hidden: false,
    created: '',
    updated: '',
    entries: [
      {
        key: 'a-1',
        previousKey: '',
        source: { kind: 'nexus', name: '' },
        mods: [],
        disabled: [],
        pinned: true,
      },
    ],
  }
  expect(updateCount(result, pinned)).toBe(0)
  expect(updateFor(result, { key: 'a-1', uniqueId: 'me.a' }, pinned)).toBeUndefined()
})

test('reshow keeps the loaded extras when the open mod is shown again', () => {
  const extras = { id: 'a/b' }
  const open = { detailId: 'a/b', extras }
  expect(reshow(open, { key: 'a', uniqueId: 'b' })).toEqual(open)
  expect(reshow(open, { key: 'a', uniqueId: 'c' })).toEqual({ detailId: 'a/c', extras: null })
  expect(reshow(open, null)).toEqual({ detailId: '', extras: null })
})

test('a removed Nexus page and unofficial versions are not installed by Update all', () => {
  const row = {
    key: 'k',
    uniqueId: 'me.a',
    name: 'A',
    installed: '1',
    version: '2',
    url: '',
    nexusId: 1,
    githubRepo: '',
    unofficial: false,
  }
  expect(listedAgainstNexus(row, { status: 'published', available: true })).toBe(true)
  expect(listedAgainstNexus(row, { status: 'deleted', available: true })).toBe(false)
  expect(installableUpdate({ ...row, unofficial: true, githubRepo: 'a/b', nexusId: 0 })).toBe(false)
  expect(installableUpdate({ ...row, unofficial: false, githubRepo: 'a/b', nexusId: 0 })).toBe(true)
})
