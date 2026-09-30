import { beforeEach, expect, mock, test } from 'bun:test'
import type { Details } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/models.ts'
import { useNexus } from '../settings/nexus.ts'
import { primeDetails, useNexusDetails } from './nexusDetails.ts'

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

mock.module('../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts', () => ({
  CachedDetails: async () => cached,
  Details: async (id: number) => {
    loads.push(id)
    return page(id)
  },
}))

beforeEach(() => {
  loads.length = 0
  cached = {}
  useNexus.setState(useNexus.getInitialState(), true)
  useNexusDetails.setState(useNexusDetails.getInitialState(), true)
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
