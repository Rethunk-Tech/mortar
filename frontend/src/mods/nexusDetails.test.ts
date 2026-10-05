import { beforeEach, expect, mock, test } from 'bun:test'
import type { File } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import type { Details } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/models.ts'
import { useNexus } from '../settings/nexus.ts'
import {
  isNewSinceLooked,
  primeDetails,
  useNexusDetails,
  useNexusSeen,
  watermarkOf,
} from './nexusDetails.ts'

const loads: number[] = []
let cached: Record<string, Details> = {}

const page = (id: number): Details => ({
  page: {
    modId: id,
    name: '',
    summary: '',
    description: '',
    pictureUrl: '',
    version: '',
    author: '',
    uploadedBy: '',
    uploaderUrl: '',
    categoryId: 0,
    endorsements: 0,
    endorsement: '',
    downloads: 0,
    uniqueDownloads: 0,
    created: '',
    updated: '',
    adult: false,
    status: '',
    available: true,
  },
  category: 'Food',
  files: null,
  changelogs: null,
})

mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts', () => ({
  CachedDetails: async () => cached,
  Details: async (_game: string, id: number) => {
    loads.push(id)
    return page(id)
  },
  Seen: async () => ({}),
  MarkSeen: async () => undefined,
}))

beforeEach(() => {
  loads.length = 0
  cached = {}
  useNexus.setState(useNexus.getInitialState(), true)
  useNexusDetails.setState(useNexusDetails.getInitialState(), true)
  useNexusSeen.setState(useNexusSeen.getInitialState(), true)
})

test('sign-in starts details reads that were skipped while signed out', async () => {
  await primeDetails([2400])
  expect(loads).toEqual([])
  useNexus.setState({ signedIn: true, name: 'pat', premium: false })
  await Promise.resolve()
  await Promise.resolve()
  expect(loads).toEqual([2400])
})

test('primeDetails applies cache over an error-only entry', async () => {
  const details = page(2400)
  cached = { '2400': details }
  useNexusDetails.setState({ byId: { 2400: { error: 'offline' } } })
  await primeDetails([2400])
  expect(useNexusDetails.getState().byId[2400]).toEqual({ details })
  expect(loads).toEqual([])
})

test('primeDetails enqueues an error-only entry when cache has nothing', async () => {
  useNexusDetails.setState({ byId: { 2400: { error: 'offline' } } })
  await primeDetails([2400])
  expect(loads).toEqual([])
  useNexus.setState({ signedIn: true, name: 'pat', premium: false })
  await Promise.resolve()
  await Promise.resolve()
  expect(loads).toEqual([2400])
})

const file = (category: string, uploaded: string): File => ({
  fileId: 1,
  fileName: 'a.zip',
  name: 'a',
  description: '',
  version: '1',
  modVersion: '1',
  category,
  sizeKb: 1,
  isPrimary: false,
  replacedBy: 0,
  uploaded,
})

test('isNewSinceLooked is false until the user has looked', () => {
  const now = watermarkOf({
    files: [file('MAIN', '2025-01-01T00:00:00Z')],
    changelogs: [{ version: '1.1.0', notes: [] }],
  })
  expect(isNewSinceLooked(undefined, now)).toBe(false)
})

test('isNewSinceLooked is true when a current file is newer than last look', () => {
  const older = watermarkOf({ files: [file('MAIN', '2024-01-01T00:00:00Z')], changelogs: [] })
  const newer = watermarkOf({ files: [file('MAIN', '2025-06-01T00:00:00Z')], changelogs: [] })
  expect(isNewSinceLooked(older, newer)).toBe(true)
  expect(isNewSinceLooked(newer, newer)).toBe(false)
})

test('isNewSinceLooked is true when a changelog version is newer than last look', () => {
  const seen = watermarkOf({
    files: [file('MAIN', '2024-01-01T00:00:00Z')],
    changelogs: [{ version: '1.0.0', notes: [] }],
  })
  const now = watermarkOf({
    files: [file('MAIN', '2024-01-01T00:00:00Z')],
    changelogs: [{ version: '1.1.0', notes: [] }],
  })
  expect(isNewSinceLooked(seen, now)).toBe(true)
})
