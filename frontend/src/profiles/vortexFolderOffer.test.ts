import { expect, test } from 'bun:test'
import type { SourceInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/migrate/models.ts'
import { offerVortexFolder } from './vortexFolderOffer.ts'

const source = (kind: string) => ({ kind, name: kind, profiles: [] }) as unknown as SourceInfo

test('offered only for Vortex games with no Vortex install listed', () => {
  expect(offerVortexFolder(true, [])).toBe(true)
  expect(offerVortexFolder(true, [source('mo2')])).toBe(true)
  expect(offerVortexFolder(true, [source('vortex')])).toBe(false)
  expect(offerVortexFolder(false, [])).toBe(false)
})
