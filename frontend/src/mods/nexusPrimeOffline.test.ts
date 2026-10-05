import { expect, mock, test } from 'bun:test'
import { useNexus } from '../settings/nexus.ts'

const asked: number[][] = []
let offline = true
const page = { page: { name: 'Mod' }, partial: true }
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts', () => ({
  CachedDetails: async () => ({}),
  Details: async () => ({}),
  PrimeDetails: async (_game: string, ids: number[]) => {
    asked.push(ids)
    // Offline, only what the cache held comes back, with the reason the rest did not.
    return offline
      ? { details: { '541': page }, error: 'Nexus is unreachable' }
      : { details: Object.fromEntries(ids.map((id) => [`${id}`, page])) }
  },
  Seen: async () => ({}),
  MarkSeen: async () => undefined,
}))
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/netstate/service.ts', () => ({
  States: async () => [{ id: 'nexus', unreachable: offline }],
}))
const { nexusChanged, primeDetails, useNexusDetails } = await import('./nexusDetails.ts')

test('a prime keeps what arrived and asks for the rest when Nexus is reachable again', async () => {
  useNexus.setState({ signedIn: true } as never)
  await primeDetails([541, 542])
  expect(useNexusDetails.getState().byId[541]?.details).toBeDefined()
  expect(useNexusDetails.getState().byId[542]?.details).toBeUndefined()
  offline = false
  await nexusChanged()
  expect(asked).toEqual([[541, 542], [542]])
  expect(useNexusDetails.getState().byId[542]?.details).toBeDefined()
})
