import { expect, mock, test } from 'bun:test'
import { useNexus } from '../settings/nexus.ts'

let primeCalls = 0
let offline = true
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts', () => ({
  CachedDetails: async () => ({}),
  Details: async () => ({}),
  PrimeDetails: async () => {
    primeCalls++
    if (offline) {
      throw new Error('Nexus is unreachable')
    }
    return { '541': { page: { name: 'Mod' }, partial: true } }
  },
  Seen: async () => ({}),
  MarkSeen: async () => undefined,
}))
const { primeDetails, useNexusDetails } = await import('./nexusDetails.ts')

test('a prime that failed while offline is asked again once Nexus answers', async () => {
  useNexus.setState({ signedIn: true } as never)
  await primeDetails([541])
  expect(useNexusDetails.getState().byId[541]?.details).toBeUndefined()
  offline = false
  await primeDetails([541])
  expect(primeCalls).toBe(2)
  expect(useNexusDetails.getState().byId[541]?.details).toBeDefined()
})
