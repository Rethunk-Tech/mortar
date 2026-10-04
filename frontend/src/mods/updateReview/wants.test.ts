import { expect, test } from 'bun:test'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Entry } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useNexusDetails } from '../nexusDetails.ts'
import { updateWant, withOptionalLoaded } from './wants.ts'

const update = { key: 'nexus-7-1', nexusId: 7, name: 'Mod', version: '2.0' } as Update
const entry = (key: string, fileId: number, extra: Partial<Entry> = {}) =>
  ({ key, source: { kind: 'nexus', name: key, modId: 7, fileId }, mods: [], ...extra }) as Entry

test('an update with no optional files is just the main request', async () => {
  const profile = { entries: [entry('nexus-7-1', 1)] }
  expect(await withOptionalLoaded(update, profile)).toEqual([updateWant(update)])
})

test('an update also asks for newer versions of the optional files laid over it', async () => {
  const profile = {
    entries: [entry('nexus-7-1', 1), entry('nexus-7-2', 2, { overlayOf: 'nexus-7-1' })],
  }
  const files = [
    { fileId: 2, fileName: 'a.zip', name: '', version: '1', category: 'OPTIONAL', replacedBy: 3 },
    { fileId: 3, fileName: 'a2.zip', name: '', version: '2', category: 'OPTIONAL', replacedBy: 0 },
  ]
  useNexusDetails.setState({ byId: { 7: { details: { files } as never } } })
  const wants = await withOptionalLoaded(update, profile)
  expect(wants.map((w) => [w.currentKey, w.fileId])).toEqual([
    ['nexus-7-1', undefined],
    ['nexus-7-2', 3],
  ])
})
