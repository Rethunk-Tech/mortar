import { expect, test } from 'bun:test'
import type { SourceInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/migrate/models.ts'
import { offerMO2Folder, offerVortexFolder } from './vortexFolderOffer.ts'

const source = (kind: string) => ({ kind, name: kind, profiles: [] }) as unknown as SourceInfo

test('offered only for Vortex games with no Vortex install listed', () => {
  expect(offerVortexFolder(true, [])).toBe(true)
  expect(offerVortexFolder(true, [source('mo2')])).toBe(true)
  expect(offerVortexFolder(true, [source('vortex')])).toBe(false)
  expect(offerVortexFolder(false, [])).toBe(false)
})

test('the MO2 chooser is offered until an MO2 instance is detected', () => {
  expect(offerMO2Folder(true, [source('vortex')])).toBe(true)
  expect(offerMO2Folder(true, [source('mo2')])).toBe(false)
  expect(offerMO2Folder(false, [])).toBe(false)
})
