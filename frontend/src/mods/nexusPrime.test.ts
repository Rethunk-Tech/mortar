import { expect, mock, test } from 'bun:test'
import { useNexus } from '../settings/nexus.ts'

let primeCalls = 0
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts', () => ({
  CachedDetails: async () => ({}),
  Details: async () => ({}),
  // Nexus returns nothing for 9999: hidden, removed, or a wrong id from an update key.
  PrimeDetails: async () => {
    primeCalls++
    return {}
  },
  Seen: async () => ({}),
  MarkSeen: async () => undefined,
}))
const { primeDetails, useNexusDetails } = await import('./nexusDetails.ts')

test('priming a mod Nexus does not know settles: one request, no store churn', async () => {
  useNexus.setState({ signedIn: true } as never)
  let wakes = 0
  const unsub = useNexusDetails.subscribe(() => {
    wakes++
    if (wakes < 50) {
      primeDetails([9999])
    }
  })
  await primeDetails([9999])
  await primeDetails([9999])
  await new Promise((r) => setTimeout(r, 30))
  unsub()
  expect(primeCalls).toBe(1)
  expect(wakes).toBe(0)
})
