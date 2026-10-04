import { describe, expect, test } from 'bun:test'
import type { File } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexus/models.ts'
import type { Update } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type { Entry } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  alternativeGroups,
  installedFileIds,
  newerVersion,
  nexusOptionalFiles,
  optionalUpdateWants,
  plainDescription,
} from './optionalFiles.ts'

const file = (fileId: number, extra: Partial<File> = {}) =>
  ({
    fileId,
    fileName: `file-${fileId}.zip`,
    name: '',
    description: '',
    version: '1.0',
    category: 'OPTIONAL',
    replacedBy: 0,
    ...extra,
  }) as File

const entry = (key: string, modId: number, fileId: number, extra: Partial<Entry> = {}) =>
  ({ key, source: { kind: 'nexus', name: key, modId, fileId }, mods: [], ...extra }) as Entry

describe('nexusOptionalFiles', () => {
  test('keeps Optional and Miscellaneous files only', () => {
    const files = [
      file(1, { category: 'MAIN' }),
      file(2),
      file(3, { category: 'MISCELLANEOUS' }),
      file(4, { category: 'OLD_VERSION' }),
      file(5, { category: 'ARCHIVED' }),
    ]
    expect(nexusOptionalFiles(files).map((f) => f.fileId)).toEqual([2, 3])
  })
})

describe('newerVersion', () => {
  test('follows the file_updates chain to the newest listed file', () => {
    const files = [file(2, { replacedBy: 5 }), file(5, { replacedBy: 9 }), file(9)]
    expect(newerVersion(files, 2)?.fileId).toBe(9)
    expect(newerVersion(files, 9)).toBeUndefined()
  })
  test('matches by display name, then by archive name without its version', () => {
    const named = [
      file(2, { name: 'Dark' }),
      file(7, { name: 'dark ' }),
      file(8, { name: 'Light' }),
    ]
    expect(newerVersion(named, 2)?.fileId).toBe(7)
    const stems = [
      file(2, { fileName: 'Dark Theme 1.0.zip' }),
      file(6, { fileName: 'Dark Theme-1.2.zip' }),
      file(8, { fileName: 'Dark Theme 2.0.zip', category: 'OLD_VERSION' }),
    ]
    expect(newerVersion(stems, 2)?.fileId).toBe(6)
  })
})

describe('optionalUpdateWants', () => {
  const update = { key: 'nexus-7-1', nexusId: 7, name: 'Mod' } as Update
  test('asks for each optional file of the updated entry that has a newer version, naming its file', () => {
    const profile = {
      entries: [
        entry('nexus-7-1', 7, 1),
        entry('nexus-7-2', 7, 2, { overlayOf: 'nexus-7-1' }),
        entry('nexus-7-3', 7, 3, { overlayOf: 'nexus-7-1' }),
        entry('nexus-8-4', 8, 4),
      ],
    }
    const files = [
      file(2, { replacedBy: 12 }),
      file(12, { version: '2.0', fileName: 'alt-2.zip' }),
      file(3),
    ]
    expect(optionalUpdateWants(profile, update, files)).toEqual([
      {
        kind: 'update',
        modId: 7,
        fileId: 12,
        name: 'Mod',
        fileName: 'alt-2.zip',
        version: '2.0',
        currentKey: 'nexus-7-2',
      },
    ])
    expect(installedFileIds(profile, 7)).toEqual(new Set([1, 2, 3]))
  })
})

describe('alternativeGroups', () => {
  test('joins optional files that replace the same files, in profile order', () => {
    const sets = [
      { key: 'a', replaces: [], adds: [], alternatives: ['c'] },
      { key: 'b', replaces: [], adds: [], alternatives: [] },
      { key: 'c', replaces: [], adds: [], alternatives: ['a', 'd'] },
      { key: 'd', replaces: [], adds: [], alternatives: ['c'] },
    ]
    expect(alternativeGroups(sets)).toEqual([['a', 'c', 'd']])
  })
})

test('plainDescription flattens BBCode to one line', () => {
  expect(plainDescription('[b]Dark[/b] look\n\n[i]for[/i] the menus')).toBe(
    'Dark look for the menus',
  )
})

test('the list and grid share the details sidebar that lists optional files, and cards keep the count chip', async () => {
  const read = (name: string) => Bun.file(new URL(name, import.meta.url)).text()
  expect(await read('./Sidebar.tsx')).toContain('<OptionalFiles mod={mod} profile={profile} />')
  expect(await read('./ModsTab.tsx')).toContain('<ModSidebar profile={profile} />')
  expect(await read('./ModCards.tsx')).toContain('<OverlayCountChip mod={m} profile={profile} />')
})
