import { afterEach, expect, mock, test } from 'bun:test'
import * as netstate from '../../bindings/github.com/Rethunk-Tech/mortar/internal/netstate/service.ts'
import * as nexussvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { useNexus } from '../settings/nexus.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { nexusChanged, primeDetails, useNexusDetails } from './nexusDetails.ts'

const asked: number[][] = []
let offline = true
const page = { page: { name: 'Mod' }, partial: true }
afterEach(() => {
  mock.restore()
})

test('a prime keeps what arrived and asks for the rest when Nexus is reachable again', async () => {
  stubBinding(nexussvc, 'PrimeDetails', (_game: string, ids: number[]) => {
    asked.push(ids)
    // Offline, only what the cache held comes back, with the reason the rest did not.
    return offline
      ? { details: { '541': page }, error: 'Nexus is unreachable' }
      : { details: Object.fromEntries(ids.map((id) => [`${id}`, page])) }
  })
  stubBinding(netstate, 'States', () => [{ id: 'nexus', unreachable: offline }])
  useNexus.setState({ signedIn: true } as never)
  await primeDetails('stardew', [541, 542])
  expect(useNexusDetails.getState().byKey['stardew:541']?.details).toBeDefined()
  expect(useNexusDetails.getState().byKey['stardew:542']?.details).toBeUndefined()
  offline = false
  await nexusChanged()
  expect(asked).toEqual([[541, 542], [542]])
  expect(useNexusDetails.getState().byKey['stardew:542']?.details).toBeDefined()
})
