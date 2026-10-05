import { expect, test } from 'bun:test'
import { groupEntries } from './archiveTree.ts'

test('groups by top folder, fills implicit folders and marks manifests', () => {
  const groups = groupEntries(
    [
      { path: 'Mod-b/x.dll', size: 5, isDir: false },
      { path: 'Mod/manifest.json', size: 2, isDir: false },
      { path: 'Mod/i18n/en.json', size: 3, isDir: false },
      { path: 'readme.txt', size: 1, isDir: false },
    ],
    [{ folder: 'Mod', id: 'a.Mod', name: 'Mod', version: '1.0' }],
  )
  expect(groups.map((g) => g.folder)).toEqual(['Mod', 'Mod-b', ''])
  const [mod] = groups
  expect(mod?.rows.map((r) => [r.path, r.depth])).toEqual([
    ['Mod', 0],
    ['Mod/i18n', 1],
    ['Mod/i18n/en.json', 2],
    ['Mod/manifest.json', 1],
  ])
  expect(mod?.rows[0]?.manifest?.id).toBe('a.Mod')
  expect(mod?.size).toBe(5)
})
