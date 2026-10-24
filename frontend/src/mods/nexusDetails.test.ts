import { afterEach, beforeEach, expect, mock, test } from 'bun:test'
import type { File } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import type { Details } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/models.ts'
import * as nexussvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { useNexus } from '../settings/nexus.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import {
  initNexusSeen,
  isNewSinceLooked,
  isNewSource,
  mergeCachedDetails,
  newestOffered,
  nexusEntryOf,
  primeDetails,
  useNexusDetails,
  useNexusSeen,
  watermarkOf,
} from './nexusDetails.ts'

const loads: number[] = []
const primes: number[][] = []
let cached: Record<string, Details> = {}
let seenSnap: Record<string, { newestFileUnix: number; newestChange: string }> = {}

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
    requirements: [],
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

beforeEach(() => {
  stubBinding(nexussvc, 'CachedDetails', () => cached)
  stubBinding(nexussvc, 'Details', (_game: string, id: number) => {
    loads.push(id)
    return page(id)
  })
  stubBinding(nexussvc, 'PrimeDetails', (_game: string, ids: number[]) => {
    primes.push(ids)
    return {
      details: Object.fromEntries(ids.map((id) => [`${id}`, { ...page(id), partial: true }])),
    }
  })
  stubBinding(nexussvc, 'Seen', () => seenSnap)
  stubBinding(nexussvc, 'MarkSeen', () => undefined)
  loads.length = 0
  primes.length = 0
  cached = {}
  useNexus.setState(useNexus.getInitialState(), true)
  useNexusDetails.setState(useNexusDetails.getInitialState(), true)
  useNexusSeen.setState(useNexusSeen.getInitialState(), true)
})

afterEach(() => {
  mock.restore()
})

test('sign-in primes the page data skipped while signed out in one batch', async () => {
  await primeDetails('stardew', [2400, 2401])
  expect(primes).toEqual([])
  useNexus.setState({ signedIn: true, name: 'pat', premium: false })
  await new Promise((resolve) => setTimeout(resolve, 0))
  expect(primes).toEqual([[2400, 2401]])
  expect(loads).toEqual([])
  expect(nexusEntryOf(useNexusDetails.getState().byKey, 'stardew', 2400)?.details?.partial).toBe(
    true,
  )
})

test('primeDetails applies cache over an error-only entry', async () => {
  const details = page(2400)
  cached = { '2400': details }
  useNexusDetails.setState({ byKey: { 'stardew:2400': { error: 'offline' } } })
  await primeDetails('stardew', [2400])
  expect(nexusEntryOf(useNexusDetails.getState().byKey, 'stardew', 2400)).toEqual({ details })
  expect(loads).toEqual([])
})

const named = (id: number, name: string): Details => ({
  ...page(id),
  page: { ...page(id).page, name },
})

test("one mod id in two games keeps each game's details", async () => {
  cached = { '2400': named(2400, 'Stardew mod') }
  await mergeCachedDetails('stardew', [2400])
  cached = { '2400': named(2400, 'Lethal mod') }
  await mergeCachedDetails('lethal-company', [2400])
  const { byKey } = useNexusDetails.getState()
  expect(nexusEntryOf(byKey, 'stardew', 2400)?.details?.page.name).toBe('Stardew mod')
  expect(nexusEntryOf(byKey, 'lethal-company', 2400)?.details?.page.name).toBe('Lethal mod')
})

test('a look is kept per game, and a look recorded by bare id still reads in any game', async () => {
  seenSnap = {
    'nexus:stardew:541': { newestFileUnix: 9, newestChange: '2.0' },
    '542': { newestFileUnix: 7, newestChange: '1.0' },
  }
  await initNexusSeen()
  const s = useNexusSeen.getState()
  expect(s.byKey['stardew:541']?.newestChange).toBe('2.0')
  expect(s.byKey['lethal-company:541']).toBeUndefined()
  expect(s.legacy[542]?.newestChange).toBe('1.0')
  seenSnap = {}
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

test('a mod from another source is new only once a later version than the one looked at is offered', () => {
  expect(isNewSource(undefined, '2.0')).toBe(false)
  expect(isNewSource('1.0', '1.0')).toBe(false)
  expect(isNewSource('1.0', undefined)).toBe(false)
  expect(isNewSource('1.0', '1.1')).toBe(true)
  expect(newestOffered('1.0', '1.1')).toBe('1.1')
  expect(newestOffered('1.2', '1.1')).toBe('1.2')
})
