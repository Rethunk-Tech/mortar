import { beforeEach, expect, mock, test } from 'bun:test'
import type { Details } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/models.ts'
import { useNexus } from '../settings/nexus.ts'
import { primeDetails, useNexusDetails } from './nexusDetails.ts'

const loads: number[] = []

mock.module('../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts', () => ({
  CachedDetails: async () => ({}),
  Details: async (id: number) => {
    loads.push(id)
    return {
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
    } satisfies Details
  },
}))

beforeEach(() => {
  loads.length = 0
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
