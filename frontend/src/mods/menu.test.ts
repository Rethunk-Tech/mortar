import { expect, test } from 'bun:test'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { entryFileLabel } from './menu.ts'

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
