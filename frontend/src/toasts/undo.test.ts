import { expect, test } from 'bun:test'
import {
  downloadWantsForEntries,
  entriesForKeys,
  entriesMatchingMissing,
  entryFieldsOf,
  fieldsStillUndoable,
  missingModNames,
  parseMissingStoreList,
  type UndoEntry,
  undoRevertTarget,
  unfetchableNames,
} from './undo.ts'

const entry = (patch: Partial<UndoEntry> & Pick<UndoEntry, 'key' | 'source'>): UndoEntry => ({
  mods: [{ name: 'SpaceCore' }],
  ...patch,
})

test('parseMissingStoreList reads named mods from a revert error', () => {
  expect(parseMissingStoreList('other')).toEqual([])
  expect(
    parseMissingStoreList('missing from the store: SpaceCore, Pathoschild.ContentPatcher'),
  ).toEqual(['SpaceCore', 'Pathoschild.ContentPatcher'])
})

test('entriesMatchingMissing and missingModNames name the failed mods', () => {
  const entries = [
    entry({ key: 'k1', source: { kind: 'nexus', modId: 1, fileId: 2, name: 'sc.zip' } }),
    entry({
      key: 'k2',
      mods: [{ name: 'Other' }],
      source: { kind: 'local' },
    }),
  ]
  const hit = entriesMatchingMissing(entries, ['SpaceCore'])
  expect(hit.map((e) => e.key)).toEqual(['k1'])
  expect(missingModNames(hit)).toEqual(['SpaceCore'])
})

test('downloadWantsForEntries builds nexus and github installs', () => {
  expect(
    downloadWantsForEntries([
      entry({
        key: 'k1',
        source: { kind: 'nexus', modId: 9, fileId: 4, name: 'a.zip', version: '1.0' },
      }),
      entry({
        key: 'k2',
        source: { kind: 'github', repo: 'owner/repo', tag: 'v1', asset: 'm.zip', name: 'm' },
      }),
      entry({ key: 'k3', source: { kind: 'local', name: 'x.zip' } }),
    ]),
  ).toEqual([
    { kind: 'install', name: 'a.zip', version: '1.0', modId: 9, fileId: 4 },
    { kind: 'install', name: 'm', version: '', repo: 'owner/repo', tag: 'v1', asset: 'm.zip' },
  ])
})

test('entryFieldsOf and entriesForKeys keep the previous bulk values', () => {
  const entries = [
    entry({
      key: 'k1',
      pinned: true,
      skipVersion: '2.0',
      tags: ['farm'],
      categoryOverride: 'Crops',
      source: { kind: 'nexus' },
    }),
    entry({ key: 'k2', source: { kind: 'nexus' } }),
  ]
  expect(entryFieldsOf(entries, ['k1'])).toEqual([
    { key: 'k1', pinned: true, skipVersion: '2.0', tags: ['farm'], categoryOverride: 'Crops' },
  ])
  expect(entriesForKeys(entries, ['k2']).map((e) => e.key)).toEqual(['k2'])
})

test('undoRevertTarget is the latest history event', () => {
  expect(undoRevertTarget([])).toBe('')
  expect(undoRevertTarget([{ id: 'new' }, { id: 'old' }])).toBe('new')
})

test('fieldsStillUndoable is true only while the previous values differ', () => {
  const prev = entryFieldsOf(
    [
      entry({
        key: 'k1',
        pinned: false,
        tags: [],
        source: { kind: 'nexus' },
      }),
    ],
    ['k1'],
  )
  expect(
    fieldsStillUndoable(
      [entry({ key: 'k1', pinned: true, tags: ['farm'], source: { kind: 'nexus' } })],
      prev,
    ),
  ).toBe(true)
  expect(
    fieldsStillUndoable(
      [entry({ key: 'k1', pinned: false, tags: [], source: { kind: 'nexus' } })],
      prev,
    ),
  ).toBe(false)
})

test('unfetchableNames lists entries with no source to download from', () => {
  const entries = [
    entry({ key: 'k1', source: { kind: 'nexus', modId: 1, fileId: 2 } }),
    entry({ key: 'k2', mods: [{ name: 'Local' }], source: { kind: 'local' } }),
  ]
  expect(unfetchableNames(entries)).toEqual(['Local'])
})
