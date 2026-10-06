import { expect, test } from 'bun:test'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { entryFileLabel, extraFileArchive, extraFileLabel } from './menu.ts'

test('a file is labelled by its mod, then its file name', () => {
  const entry = {
    key: 'nexus-22743-175656',
    source: { kind: 'nexus', name: 'Alchemistry-22743-2-0-2.zip', version: '2.0.2' },
    mods: [{ name: 'Alchemistry (Bush Bloom Mod)', version: '2.0.2' }],
  } as Entry
  expect(entryFileLabel(entry)).toBe('Alchemistry (Bush Bloom Mod) (2.0.2)')
  expect(entryFileLabel({ ...entry, mods: [] } as Entry)).toBe(
    'Alchemistry-22743-2-0-2.zip (2.0.2)',
  )
})

test('an extra file is labelled by its display name, with its archive name apart', () => {
  const files = [
    {
      fileId: 127_365,
      fileName: 'Grampleton Fields-3753-1-15-9-1741905961.zip',
      name: 'Grampleton Fields',
      version: '1.15.9',
    },
  ]
  const entry = { key: 'nexus-3753-135998', mods: [] } as unknown as Entry
  expect(extraFileLabel(entry, 'nexus-3753-127365', files)).toBe('Grampleton Fields (1.15.9)')
  expect(extraFileArchive('nexus-3753-127365', files)).toBe(
    'Grampleton Fields-3753-1-15-9-1741905961.zip',
  )
  expect(extraFileArchive('nexus-3753-127365', undefined)).toBe('')
})
